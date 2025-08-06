package logic

import (
	"context"
	"encoding/json"
	"fmt"

	"document_agent/app/llmcenter/cmd/rpc/internal/llm"
	"document_agent/app/llmcenter/cmd/rpc/internal/svc"
	"document_agent/app/llmcenter/cmd/rpc/pb"
	"document_agent/app/llmcenter/cmd/rpc/types"
	"document_agent/app/llmcenter/model"
	"document_agent/pkg/tool"
	"document_agent/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type ChatResumeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewChatResumeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ChatResumeLogic {
	return &ChatResumeLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// ChatResume 是处理中断后继续生成的核心 RPC 方法
func (l *ChatResumeLogic) ChatResume(in *pb.ChatResumeRequest, stream pb.LlmCenter_ChatResumeServer) error {
	// 1. 验证会话的有效性
	if err := l.validateConversation(in.UserId, in.ConversationId); err != nil {
		return err
	}

	// 2. 从 Redis 中获取并删除中断事件ID
	eventID, err := l.getAndDelInterruptEventID(in.ConversationId)
	if err != nil {
		return err
	}

	// 4. 构建对大模型 Resume API 的请求
	llmReq := types.LLMResumeApiRequest{
		EventID:   eventID,
		EventType: "resume",
		Content:   in.Content,
	}
	reqBody, err := json.Marshal(llmReq)
	if err != nil {
		l.Errorf("failed to marshal llm resume request: %v :%w", err, xerr.ErrRequestParam)
		return err
	}

	// 5. 调用大模型 Resume API 并处理流式响应
	xingchenClient := llm.NewXingChenClient(l.ctx, l.svcCtx)
	assistantReply, err := xingchenClient.StreamResume(reqBody, stream)
	if err != nil {
		return err // 错误已在 StreamResume 中包装
	}

	// 6. 保存最终生成的完整文章
	assistantMessageID, err := l.saveFinalDocument(in.ConversationId, assistantReply)
	if err != nil {
		// 记录错误，但仍然继续向客户端发送结束信号
		l.Errorf("saveFinalDocument failed: %v", err)
	}

	// 7. 向客户端发送结束事件
	return l.sendEndEvent(stream, in.ConversationId, assistantMessageID)
}

// validateConversation 验证会话是否存在且属于该用户
func (l *ChatResumeLogic) validateConversation(userID int64, convID string) error {
	conversation, err := l.svcCtx.ConversationModel.FindOne(l.ctx, convID)
	if err != nil {
		if err == model.ErrNotFound {
			return fmt.Errorf("validateConversation db conversation FindOne not found err:%+v, conversationId:%s: %w", err, convID, xerr.ErrConversationNotFound)
		}
		return fmt.Errorf("validateConversation db conversation FindOne err:%+v, conversationId:%s: %w", err, convID, xerr.ErrDbError)
	}
	if conversation.UserId != userID {
		return fmt.Errorf("validateConversation user %d is not allowed to access conversation %s: %w", userID, convID, xerr.ErrConversationAccessDenied)
	}
	return nil
}

// getAndDelInterruptEventID 从 Redis 中获取中断事件ID，成功后立即删除
func (l *ChatResumeLogic) getAndDelInterruptEventID(convID string) (string, error) {
	redisKey := fmt.Sprintf("llm:interrupt:%s", convID)
	eventID, err := l.svcCtx.RedisClient.Get(redisKey)
	if eventID == "" {
		return "", fmt.Errorf("ChatResume no interrupt event found for conv %s, maybe expired: %w",
			convID, xerr.ErrLLMInterruptEventNotFound)
	}
	if err != nil {
		return "", fmt.Errorf("ChatResume failed to get event_id from redis: %v: %w", err, xerr.ErrDbError)
	}
	// 获取成功后立即删除该key，防止被重复使用
	l.svcCtx.RedisClient.Del(redisKey)
	return eventID, nil
}

// saveFinalDocument 保存最终生成的完整文章
func (l *ChatResumeLogic) saveFinalDocument(conversationID, content string) (string, error) {
	if content == "" {
		return "", nil
	}

	// 生成唯一的 message_id
	documentID := tool.GenerateULID()

	// 插入到 documents 表
	err := l.svcCtx.DocumentsModel.InsertDocument(l.ctx, documentID, conversationID, content)
	if err != nil {
		return "", fmt.Errorf("saveFinalDocument db Insert to documents err:%+v: %w", err, xerr.ErrDbError)
	}

	return documentID, nil
}

// sendEndEvent 向客户端发送结束事件
func (l *ChatResumeLogic) sendEndEvent(stream pb.LlmCenter_ChatResumeServer, conversationID, messageID string) error {
	endEvent := &pb.SSEEndEvent{
		ConversationId: conversationID,
		MessageId:      messageID,
	}
	if err := stream.Send(&pb.ChatResumeResponse{Event: &pb.ChatResumeResponse_End{End: endEvent}}); err != nil {
		return fmt.Errorf("failed to send end event to client: %v:%w", err, xerr.ErrLLMApiCancel)
	}
	return nil
}
