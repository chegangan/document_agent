package conversation

import (
	"context"

	"document_agent/app/llmcenter/cmd/api/internal/svc"
	"document_agent/app/llmcenter/cmd/api/internal/types"
	rpcpb "document_agent/app/llmcenter/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetConversationDetailLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取指定会话的详细历史消息
func NewGetConversationDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetConversationDetailLogic {
	return &GetConversationDetailLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetConversationDetailLogic) GetConversationDetail(req *types.GetConversationDetailRequest) (*types.GetConversationDetailResponse, error) {
	rpcResp, err := l.svcCtx.LLMCenterRpc.GetConversationDetail(l.ctx, &rpcpb.GetConversationDetailRequest{
		ConversationId: req.ConversationID,
	})
	if err != nil {
		l.Logger.Errorf("RPC GetConversationDetail failed: %v", err)
		return nil, err
	}

	var history []types.Message
	for _, msg := range rpcResp.History {
		history = append(history, types.Message{
			ID:          msg.Id,
			Role:        msg.Role,
			Content:     msg.Content,
			ContentType: msg.ContentType,
			CreatedAt:   msg.CreatedAt,
		})
	}

	return &types.GetConversationDetailResponse{
		ConversationID: rpcResp.ConversationId,
		Title:          rpcResp.Title,
		History:        history,
	}, nil
}
