package model

import (
	"context"
	"fmt"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ MessagesModel = (*customMessagesModel)(nil)

type (
	// MessagesModel is an interface to be customized, add more methods here,
	// and implement the added methods in customMessagesModel.
	MessagesModel interface {
		messagesModel
		FindAllByConversation(ctx context.Context, conversationId string) ([]*Messages, error)

		withSession(session sqlx.Session) MessagesModel
	}

	customMessagesModel struct {
		*defaultMessagesModel
	}
)

// NewMessagesModel returns a model for the database table.
func NewMessagesModel(conn sqlx.SqlConn) MessagesModel {
	return &customMessagesModel{
		defaultMessagesModel: newMessagesModel(conn),
	}
}

func (m *customMessagesModel) withSession(session sqlx.Session) MessagesModel {
	return NewMessagesModel(sqlx.NewSqlConnFromSession(session))
}

func (m *defaultMessagesModel) FindAllByConversation(ctx context.Context, conversationId string) ([]*Messages, error) {
	// 假设 messagesRows 是生成文件里定义的所有列名拼接
	query := fmt.Sprintf("SELECT %s FROM %s WHERE `conversation_id` = ? ORDER BY `created_at` ASC",
		messagesRows, m.table)
	var resp []*Messages
	err := m.conn.QueryRowsCtx(ctx, &resp, query, conversationId)
	return resp, err
}
