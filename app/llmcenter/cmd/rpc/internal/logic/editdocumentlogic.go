package logic

import (
	"context"
	"database/sql"
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

type EditDocumentLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewEditDocumentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *EditDocumentLogic {
	return &EditDocumentLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *EditDocumentLogic) EditDocument(in *pb.EditDocumentRequest, stream pb.LlmCenter_EditDocumentServer) error {
	assistantMessageID := tool.GenerateULID()
	// 查询原文
	doc, err := l.svcCtx.DocumentsModel.FindOne(l.ctx, in.MessageId)
	if err != nil {
		return fmt.Errorf("document not found: %w", err)
	}

	// 构造 prompt
	prompt := fmt.Sprintf("修改：请根据以下提示修改文档内容：\n\n原文：\n%s\n\n修改提示：%s", doc.Content, in.Prompt)

	llmReq := types.LLMApiRequest{
		FlowID: l.svcCtx.Config.XingChen.FlowID,
		UID:    fmt.Sprintf("%d", in.UserId),
		Parameters: types.LLMParameters{
			AgentUserInput: prompt,
		},
		Stream: true,
	}

	reqBody, err := json.Marshal(llmReq)
	if err != nil {
		return xerr.ErrRequestParam
	}

	client := llm.NewXingChenClient(l.ctx, l.svcCtx)
	result, err := client.StreamChatForEdit(reqBody, stream, in.ConversationId)
	if err != nil {
		return err
	}

	// 插入用户输入（prompt）为 user 消息
	userMessageID := tool.GenerateULID()
	_, err = l.svcCtx.MessageModel.Insert(l.ctx, &model.Messages{
		MessageId:      userMessageID,
		ConversationId: in.ConversationId,
		Role:           "user",
		Content:        in.Prompt,
		ContentType:    "text",
		Metadata:       sql.NullString{Valid: false},
	})
	if err != nil {
		return fmt.Errorf("保存用户消息失败: %w", err)
	}

	// 保存到 messages 表
	if result != "" {
		msg := &model.Messages{
			MessageId:      assistantMessageID,
			ConversationId: in.ConversationId,
			Role:           "assistant",
			Content:        result,
			ContentType:    "text",
			Metadata:       sql.NullString{Valid: false},
		}
		_, err := l.svcCtx.MessageModel.Insert(l.ctx, msg)
		if err != nil {
			return fmt.Errorf("保存消息失败: %w", err)
		}
	}

	// 更新 documents 表
	err = l.svcCtx.DocumentsModel.UpdateContent(l.ctx, in.MessageId, result)
	if err != nil {
		return fmt.Errorf("更新 documents 表失败: %w", err)
	}
	// 发送结束事件
	return stream.Send(&pb.EditDocumentResponse{
		Event: &pb.EditDocumentResponse_End{
			End: &pb.SSEEndEvent{
				ConversationId: in.ConversationId,
				MessageId:      in.MessageId,
			},
		},
	})
}
