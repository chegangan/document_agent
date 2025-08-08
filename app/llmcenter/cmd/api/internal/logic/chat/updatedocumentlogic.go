package chat

import (
	"context"
	"fmt"
	"net/http"

	"document_agent/app/llmcenter/cmd/api/internal/svc"
	"document_agent/app/llmcenter/cmd/api/internal/types"
	"document_agent/app/llmcenter/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateDocumentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 手动修改公文内容
func NewUpdateDocumentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateDocumentLogic {
	return &UpdateDocumentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateDocumentLogic) UpdateDocument(req *types.UpdateDocumentRequest) (*types.UpdateDocumentResponse, error) {
	_, err := l.svcCtx.LLMCenterRpc.UpdateDocument(l.ctx, &pb.UpdateDocumentRequest{
		ConversationId: req.Conversation_id,
		MessageId:      req.Message_id,
		Prompt:         req.Prompt,
	})
	if err != nil {
		return nil, fmt.Errorf("调用 RPC 更新文档失败: %v, ConversationId: %s, MessageId: %s: %w",
			err, req.Conversation_id, req.Message_id, http.StatusInternalServerError)
	}

	return &types.UpdateDocumentResponse{Success: true}, nil
}
