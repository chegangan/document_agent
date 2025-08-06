package logic

import (
	"context"
	"fmt"
	"time"

	"document_agent/app/llmcenter/cmd/rpc/internal/svc"
	"document_agent/app/llmcenter/cmd/rpc/pb"
	"document_agent/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetDocumentDetailLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetDocumentDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetDocumentDetailLogic {
	return &GetDocumentDetailLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// RPC 方法: GetDocumentDetail
func (l *GetDocumentDetailLogic) GetDocumentDetail(in *pb.GetDocumentDetailRequest) (*pb.GetDocumentDetailResponse, error) {
	docs, err := l.svcCtx.DocumentsModel.FindByConversationId(l.ctx, in.ConversationId)
	if err != nil {
		return nil, fmt.Errorf("查询 documents 失败: %v: %w", err, xerr.ErrDbError)
	}

	var result []*pb.Document
	for _, d := range docs {
		result = append(result, &pb.Document{
			MessageId: d.MessageId,
			Content:   d.Content,
			CreatedAt: d.CreatedAt.Format(time.RFC3339),
		})
	}

	return &pb.GetDocumentDetailResponse{
		ConversationId: in.ConversationId,
		Documents:      result,
	}, nil
}
