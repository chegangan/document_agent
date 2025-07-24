package model

import (
	"context"
	"fmt"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ ConversationsModel = (*customConversationsModel)(nil)

type (
	// ConversationsModel is an interface to be customized, add more methods here,
	// and implement the added methods in customConversationsModel.
	ConversationsModel interface {
		conversationsModel

		FindAllByUser(ctx context.Context, userId string) ([]*Conversations, error)

		withSession(session sqlx.Session) ConversationsModel
	}

	customConversationsModel struct {
		*defaultConversationsModel
	}
)

// NewConversationsModel returns a model for the database table.
func NewConversationsModel(conn sqlx.SqlConn) ConversationsModel {
	return &customConversationsModel{
		defaultConversationsModel: newConversationsModel(conn),
	}
}

func (m *customConversationsModel) withSession(session sqlx.Session) ConversationsModel {
	return NewConversationsModel(sqlx.NewSqlConnFromSession(session))
}

func (m *defaultConversationsModel) FindAllByUser(ctx context.Context, userId string) ([]*Conversations, error) {
	query := fmt.Sprintf("SELECT %s FROM %s WHERE `user_id` = ?", conversationsRows, m.table)
	var resp []*Conversations
	err := m.conn.QueryRowsCtx(ctx, &resp, query, userId)
	return resp, err
}
