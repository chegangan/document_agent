package model

import "github.com/zeromicro/go-zero/core/stores/sqlx"

var _ ConversationsModel = (*customConversationsModel)(nil)

type (
	// ConversationsModel is an interface to be customized, add more methods here,
	// and implement the added methods in customConversationsModel.
	ConversationsModel interface {
		conversationsModel
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
