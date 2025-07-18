package conversation

import (
	"context"

	"document_agent/app/llmcenter/cmd/api/internal/svc"
	"document_agent/app/llmcenter/cmd/api/internal/types"

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

func (l *GetConversationDetailLogic) GetConversationDetail(req *types.GetConversationDetailRequest) (resp *types.GetConversationDetailResponse, err error) {
	// todo: add your logic here and delete this line

	return
}
