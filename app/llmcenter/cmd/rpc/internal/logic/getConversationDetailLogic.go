package logic

import (
	"context"
	"fmt"
	"time"

	"document_agent/app/llmcenter/cmd/rpc/internal/svc"
	"document_agent/app/llmcenter/cmd/rpc/pb"
	"document_agent/app/llmcenter/model"
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
	conversation, err := l.svcCtx.ConversationModel.FindOne(l.ctx, in.ConversationId)
	if err != nil {
		if err == model.ErrNotFound {
			return nil, fmt.Errorf("会话不存在或无权限: %w, ConversationId: %s", xerr.ErrConversationNotFound, in.ConversationId)
		}
		return nil, fmt.Errorf("查询会话失败: %v ,ConversationId: %s: %w", err, in.ConversationId, xerr.ErrDbError)
	}

	// 2. 查询该会话下所有消息
	msgs, err := l.svcCtx.MessageModel.FindAllByConversation(l.ctx, in.ConversationId)
	if err != nil {
		return nil, fmt.Errorf("查询消息列表失败: %v, ConversationId: %s: %w", err, in.ConversationId, xerr.ErrDbError)
	}

	// 3. 组装历史消息
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

	// 4. 返回详情响应
	return &pb.GetConversationDetailResponse{
		ConversationId: conversation.ConversationId,
		Title:          conversation.Title,
		History:        history,
	}, nil
}
