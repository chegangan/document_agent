package conversation

import (
	"context"
	"fmt"

	"document_agent/app/llmcenter/cmd/api/internal/svc"
	"document_agent/app/llmcenter/cmd/api/internal/types"
	rpcpb "document_agent/app/llmcenter/cmd/rpc/pb"
	"document_agent/pkg/ctxdata"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetConversationsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取当前用户的会话列表
func NewGetConversationsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetConversationsLogic {
	return &GetConversationsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetConversationsLogic) GetConversations(req *types.GetConversationsRequest) (*types.GetConversationsResponse, error) {
	// 调用后端 RPC
	userId1 := ctxdata.GetUidFromCtx(l.ctx)
	fmt.Printf("GetConversationsLogic userId: %d\n", userId1)
	rpcResp, err := l.svcCtx.LlmCenterRpc.GetConversations(l.ctx, &rpcpb.GetConversationsRequest{})
	if err != nil {
		l.Logger.Error("RPC GetConversations failed:", err)
		return nil, err
	}

	// 转换到 HTTP types
	var list []types.Conversation
	for _, c := range rpcResp.Data {
		list = append(list, types.Conversation{
			ConversationID: c.ConversationId,
			Title:          c.Title,
			UpdatedAt:      c.UpdatedAt,
		})
	}

	return &types.GetConversationsResponse{Data: list}, nil
}
