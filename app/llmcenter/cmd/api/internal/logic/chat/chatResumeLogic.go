package chat

import (
	"context"

	"document_agent/app/llmcenter/cmd/api/internal/svc"
	"document_agent/app/llmcenter/cmd/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ChatResumeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 在工作流中断后, 发送用户编辑好的内容以继续流程
func NewChatResumeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ChatResumeLogic {
	return &ChatResumeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ChatResumeLogic) ChatResume(req *types.ChatResumeRequest) (resp *types.ChatResumeResponse, err error) {
	// todo: add your logic here and delete this line

	return
}
