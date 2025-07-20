package logic

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"document_agent/app/llmcenter/cmd/rpc/internal/svc"
	"document_agent/app/llmcenter/cmd/rpc/pb"
	"document_agent/app/llmcenter/model"
	"document_agent/pkg/ctxdata"
	"document_agent/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetConversationDetailLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetConversationDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetConversationDetailLogic {
	return &GetConversationDetailLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// RPC 方法: GetConversationDetail
func (l *GetConversationDetailLogic) GetConversationDetail(in *pb.GetConversationDetailRequest) (*pb.GetConversationDetailResponse, error) {
	// 1. 从上下文获取 userId，并转为 string
	userId1 := ctxdata.GetUidFromCtx(l.ctx)
	var userIdInt int64 = userId1
	userId := strconv.FormatInt(userIdInt, 10)

	// 2. 只读取当前用户 own 的会话
	conv, err := l.svcCtx.ConversationsModel.FindOneByUser(l.ctx, in.ConversationId, userId)
	if err != nil {
		if err == model.ErrNotFound {
			// 会话不存在或不属于当前用户
			return nil, fmt.Errorf("会话不存在或无权限: %w", xerr.ErrUserNotFound)
		}
		l.Logger.Errorf("查询会话[%s]失败: %v", in.ConversationId, err)
		return nil, fmt.Errorf("查询会话失败: %v: %w", err, xerr.ErrDbError)
	}

	// 3. 读取该会话下所有消息
	msgs, err := l.svcCtx.MessagesModel.FindAllByConversation(l.ctx, in.ConversationId)
	if err != nil {
		l.Logger.Errorf("查询会话[%s]消息失败: %v", in.ConversationId, err)
		return nil, fmt.Errorf("查询消息列表失败: %v: %w", err, xerr.ErrDbError)
	}

	// 4. 组装历史消息
	var history []*pb.Message
	for _, m := range msgs {
		history = append(history, &pb.Message{
			Id:          m.MessageId,
			Role:        m.Role,
			Content:     m.Content,
			ContentType: m.ContentType,
			CreatedAt:   m.CreatedAt.Format(time.RFC3339),
		})
	}

	// 5. 返回详情响应
	return &pb.GetConversationDetailResponse{
		ConversationId: conv.ConversationId,
		Title:          conv.Title,
		History:        history,
	}, nil
}
