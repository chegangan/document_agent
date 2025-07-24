package logic

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"document_agent/app/llmcenter/cmd/rpc/internal/svc"
	"document_agent/app/llmcenter/cmd/rpc/pb"

	"document_agent/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetConversationsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetConversationsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetConversationsLogic {
	return &GetConversationsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// RPC 方法: GetConversations
func (l *GetConversationsLogic) GetConversations(in *pb.GetConversationsRequest) (*pb.GetConversationsResponse, error) {
	// 1. 查询所有会话
	userId := strconv.FormatInt(in.UserId, 10)

	convs, err := l.svcCtx.ConversationsModel.FindAllByUser(l.ctx, userId)
	if err != nil {
		l.Logger.Error("查询会话列表失败:", err)
		// 按 loginByMobile 样式包装错误
		return nil, fmt.Errorf("查询会话列表失败: %v: %w", err, xerr.ErrDbError)
	}

	// 2. 组装返回
	var list []*pb.Conversation
	for _, c := range convs {
		list = append(list, &pb.Conversation{
			ConversationId: c.ConversationId,
			Title:          c.Title,
			UpdatedAt:      c.UpdatedAt.Format(time.RFC3339),
		})
	}

	if len(list) == 0 {
		return nil, fmt.Errorf("没有找到任何会话: %w", xerr.ErrConversationNotFound)
	}

	return &pb.GetConversationsResponse{
		Data: list,
	}, nil
}
