package model

import (
	"context"
	"fmt"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ HistorydatasModel = (*customHistorydatasModel)(nil)

type (
	// HistorydatasModel is an interface to be customized, add more methods here,
	// and implement the added methods in customHistorydatasModel.
	HistorydatasModel interface {
		historydatasModel
		FindByConversationId(ctx context.Context, conversationId string) ([]*Historydatas, error)
		withSession(session sqlx.Session) HistorydatasModel
	}

	customHistorydatasModel struct {
		*defaultHistorydatasModel
	}
)

// NewHistorydatasModel returns a model for the database table.
func NewHistorydatasModel(conn sqlx.SqlConn) HistorydatasModel {
	return &customHistorydatasModel{
		defaultHistorydatasModel: newHistorydatasModel(conn),
	}
}

func (m *customHistorydatasModel) withSession(session sqlx.Session) HistorydatasModel {
	return NewHistorydatasModel(sqlx.NewSqlConnFromSession(session))
}

func (m *defaultHistorydatasModel) FindByConversationId(ctx context.Context, conversationId string) ([]*Historydatas, error) {
	query := fmt.Sprintf("SELECT %s FROM %s WHERE conversation_id = ? ORDER BY created_at", historydatasRows, m.table)

	var resp []*Historydatas
	err := m.conn.QueryRowsCtx(ctx, &resp, query, conversationId)
	return resp, err
}
