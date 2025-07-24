package logic

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

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
// 前端在调用这个方法后，应该马上切换下一个对话
func (l *ChatResumeLogic) ChatResume(in *pb.ChatResumeRequest, stream pb.LlmCenter_ChatResumeServer) error {
	// 1. 验证会话的有效性
	if err := l.validateConversation(in.UserId, in.ConversationId); err != nil {
		return err
	}

	// 2. 从 Redis 中获取中断事件ID
	redisKey := fmt.Sprintf("llm:interrupt:%s", in.ConversationId)
	eventID, err := l.svcCtx.RedisClient.Get(redisKey)
	if eventID == "" {
		return fmt.Errorf("ChatResume no interrupt event found for conv %s, maybe expired: %w",
			in.ConversationId, xerr.ErrLLMInterruptEventNotFound)
	}
	if err != nil {
		return fmt.Errorf("ChatResume failed to get event_id from redis: %v: %w", err, xerr.ErrDbError)
	}
	// [建议] 获取成功后立即删除该key，防止被重复使用
	l.svcCtx.RedisClient.Del(redisKey)

	// 3. 保存用户提交的、编辑后的 "document_outline"
	// 这是对话历史的重要一环，记录了用户的确认操作
	// TODO: template_id 在这一步可以添加进去，一起当作提示词
	userMessage := &model.Messages{
		MessageId:      tool.GenerateULID(),
		ConversationId: in.ConversationId,
		Role:           "user", // 代表这是用户的输入
		Content:        in.Content,
		ContentType:    "document_outline", // 表明这是用户确认后的内容清单
		// [建议] 如果 template_id 很重要，可以考虑将其存入 metadata 字段
	}
	if _, err := l.svcCtx.MessageModel.Insert(l.ctx, userMessage); err != nil {
		return fmt.Errorf("ChatResume db message Insert err:%+v, message:%+v: %w", err, userMessage, xerr.ErrDbError)
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
	return l.processLLMResumeStream(reqBody, stream, in.ConversationId)
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

// processLLMResumeStream 调用大模型Resume API，处理返回的流，并推送到客户端gRPC流
func (l *ChatResumeLogic) processLLMResumeStream(reqBody []byte, stream pb.LlmCenter_ChatResumeServer, conversationID string) error {
	// [手动修改] 确保你的配置文件中有 ResumeApiURL
	apiURL := l.svcCtx.Config.XingChen.ApiURL
	apiKey := l.svcCtx.Config.XingChen.ApiKey
	apiSecret := l.svcCtx.Config.XingChen.ApiSecret
	authToken := fmt.Sprintf("Bearer %s:%s", apiKey, apiSecret)

	req, err := http.NewRequestWithContext(l.ctx, "POST", apiURL, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create http request with context: %+v:%w", err, xerr.ErrLLMApiCancel)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authToken)
	req.Header.Set("Accept", "text/event-stream")

	client := l.svcCtx.LlmApiClient
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call llm resume api: %+v:%w", err, xerr.ErrLLMApiCancel)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("llm resume api returned non-200 status: %d, body: %s :%w",
			resp.StatusCode, string(bodyBytes), xerr.ErrLLMApiError)
	}

	scanner := bufio.NewScanner(resp.Body)
	var assistantReply strings.Builder
	var assistantMessageID string

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// [注意] Resume API 的返回结构和 Completions API 一致，可以复用解析逻辑
		var apiResp types.LLMApiResponse
		line = strings.TrimPrefix(line, "data: ")
		if err := json.Unmarshal([]byte(line), &apiResp); err != nil {
			l.Errorf("failed to unmarshal llm resume stream line: %s, error: %v", line, err)
			continue
		}

		if apiResp.Code != 0 {
			return fmt.Errorf("LLM Resume API error response: code=%d, message=%s :%w",
				apiResp.Code, apiResp.Message, xerr.ErrLLMApiError)
		}

		// [注意] Resume 流程预期不会再有 interrupt 事件，因此我们只处理 message 和 end
		if len(apiResp.Choices) > 0 {
			chunk := apiResp.Choices[0].Delta.Content
			if chunk != "" {
				assistantReply.WriteString(chunk)
				// 向客户端发送消息块
				messageEvent := &pb.SSEMessageEvent{Chunk: chunk}
				if err := stream.Send(&pb.ChatResumeResponse{Event: &pb.ChatResumeResponse_Message{Message: messageEvent}}); err != nil {
					return fmt.Errorf("failed to send message chunk to client: %v:%w", err, xerr.ErrLLMApiCancel)
				}
			}

			if apiResp.Choices[0].FinishReason == "stop" {
				break
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading llm resume stream: %v:%w", err, xerr.ErrLLMApiError)
	}

	// 6. 保存最终生成的完整文章
	assistantMessageID = tool.GenerateULID()
	assistantMessage := &model.Messages{
		MessageId:      assistantMessageID,
		ConversationId: conversationID,
		Role:           "assistant",
		Content:        assistantReply.String(),
		ContentType:    "final_document",                          // 标记为最终生成的文档
		Metadata:       sql.NullString{String: "{}", Valid: true}, // 初始化为空JSON对象
	}

	_, err = l.svcCtx.MessageModel.Insert(l.ctx, assistantMessage)
	if err != nil {
		// 记录错误，但仍然继续向客户端发送结束信号
		l.Errorf("saveAssistantMessage (final document) failed: %v", err)
	}

	// 7. 向客户端发送结束事件
	endEvent := &pb.SSEEndEvent{
		ConversationId: conversationID,
		MessageId:      assistantMessageID,
	}
	if err := stream.Send(&pb.ChatResumeResponse{Event: &pb.ChatResumeResponse_End{End: endEvent}}); err != nil {
		return fmt.Errorf("failed to send end event to client: %v:%w", err, xerr.ErrLLMApiCancel)
	}

	return nil
}
