package conversation

import (
	"context"

	"document_agent/app/llmcenter/cmd/api/internal/svc"
	"document_agent/app/llmcenter/cmd/api/internal/types"
	rpcpb "document_agent/app/llmcenter/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetDocumentDetailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 根据会话ID获取该会话的最终文档列表
func NewGetDocumentDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetDocumentDetailLogic {
	return &GetDocumentDetailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetDocumentDetailLogic) GetDocumentDetail(req *types.GetDocumentDetailRequest) (*types.GetDocumentDetailResponse, error) {
	rpcResp, err := l.svcCtx.LLMCenterRpc.GetDocumentDetail(l.ctx, &rpcpb.GetDocumentDetailRequest{
		ConversationId: req.ConversationID,
	})
	if err != nil {
		l.Logger.Errorf("调用 GetDocumentDetail RPC 失败: %v", err)
		return nil, err
	}

	var docs []types.Document
	for _, d := range rpcResp.Documents {
		docs = append(docs, types.Document{
			ID:        d.MessageId,
			Content:   d.Content,
			CreatedAt: d.CreatedAt,
		})
	}

	return &types.GetDocumentDetailResponse{
		ConversationID: rpcResp.ConversationId,
		Documents:      docs,
	}, nil
}
