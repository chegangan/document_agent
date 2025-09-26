package logic

import (
	"context"

	"document_agent/app/llmcenter/cmd/rpc/internal/svc"
	"document_agent/app/llmcenter/cmd/rpc/pb"
	"document_agent/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateDocumentLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateDocumentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateDocumentLogic {
	return &UpdateDocumentLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// RPC 方法: UpdateDocumentRequest
// func (l *UpdateDocumentLogic) UpdateDocument(in *pb.UpdateDocumentRequest) (*pb.UpdateDocumentResponse, error) {
// 	err := l.svcCtx.DocumentsModel.UpdateContent(l.ctx, in.MessageId, in.Prompt)
// 	if err != nil {
// 		return nil, fmt.Errorf("更新文档失败: %v, MessageId: %s, ConversationId: %s: %w",
// 			err, in.MessageId, in.ConversationId, xerr.ErrDbError)
// 	}
// 	return &pb.UpdateDocumentResponse{Success: true}, nil
// }

// UpdateDocument handles manual updates from the user.
func (l *UpdateDocumentLogic) UpdateDocument(in *pb.UpdateDocumentRequest) (*pb.UpdateDocumentResponse, error) {
	// Use the repository to update the document.
	// This single call handles both the database update and cache invalidation.
	err := l.svcCtx.DocRepo.UpdateDocumentContent(l.ctx, in.MessageId, in.Prompt)
	if err != nil {
		// Log the detailed error for debugging
		l.Errorf("UpdateDocument failed: %v, MessageId: %s", err, in.MessageId)
		// Return a generic database error to the client
		return nil, xerr.ErrDbError
	}

	return &pb.UpdateDocumentResponse{Success: true}, nil
}
