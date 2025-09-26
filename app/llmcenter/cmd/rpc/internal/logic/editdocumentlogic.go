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
	"github.com/zeromicro/go-zero/core/stores/sqlc"
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

	// 这个在带缓存的版本
	doc, err := l.svcCtx.DocRepo.FindDocument(l.ctx, in.MessageId)
	// _, err := l.svcCtx.DocRepo.FindDocument(l.ctx, in.MessageId)
	if err != nil {
		if err == sqlc.ErrNotFound {
			return fmt.Errorf("document not found with id %s: %w", in.MessageId, err)
		}
		return fmt.Errorf("failed to find document: %w", err)
	}

	// // 这个是不带缓存的版本
	// doc, err := l.svcCtx.DocumentsModel.FindOne(l.ctx, in.MessageId)
	// // _, err := l.svcCtx.DocumentsModel.FindOne(l.ctx, in.MessageId)
	// if err != nil {
	// 	return fmt.Errorf("document not found: %w", err)
	// }

	// 2. Construct prompt and call LLM (same as before)
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

	// result := "This is a mocked LLM result."

	// 3. Save user and assistant messages (same as before)
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

	// 4. Update the document using the repository (handles DB update and cache invalidation)
	err = l.svcCtx.DocRepo.UpdateDocumentContent(l.ctx, in.MessageId, result)
	if err != nil {
		return fmt.Errorf("更新 documents 表失败: %w", err)
	}

	// 5. Send end event (same as before)
	return stream.Send(&pb.EditDocumentResponse{
		Event: &pb.EditDocumentResponse_End{
			End: &pb.SSEEndEvent{
				ConversationId: in.ConversationId,
				MessageId:      in.MessageId,
			},
		},
	})
}
