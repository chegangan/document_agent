package model

import (
	"context"
	"fmt"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ DocumentsModel = (*customDocumentsModel)(nil)

type (
	// DocumentsModel is an interface to be customized, add more methods here,
	// and implement the added methods in customDocumentsModel.
	DocumentsModel interface {
		documentsModel
		InsertDocument(ctx context.Context, messageID, conversationID, content string) error
		FindByConversationId(ctx context.Context, conversationId string) ([]*Documents, error)
		UpdateContent(ctx context.Context, messageID, content string) error
		withSession(session sqlx.Session) DocumentsModel
	}

	customDocumentsModel struct {
		*defaultDocumentsModel
	}
)

// NewDocumentsModel returns a model for the database table.
func NewDocumentsModel(conn sqlx.SqlConn) DocumentsModel {
	return &customDocumentsModel{
		defaultDocumentsModel: newDocumentsModel(conn),
	}
}

func (m *customDocumentsModel) withSession(session sqlx.Session) DocumentsModel {
	return NewDocumentsModel(sqlx.NewSqlConnFromSession(session))
}

func (m *customDocumentsModel) InsertDocument(ctx context.Context, messageID, conversationID, content string) error {
	query := fmt.Sprintf("INSERT INTO %s (`message_id`, `conversation_id`, `content`) VALUES (?, ?, ?)", m.table)
	_, err := m.conn.ExecCtx(ctx, query, messageID, conversationID, content)
	return err
}

func (m *defaultDocumentsModel) FindByConversationId(ctx context.Context, conversationId string) ([]*Documents, error) {
	query := fmt.Sprintf("SELECT %s FROM %s WHERE `conversation_id` = ? ORDER BY `created_at` ASC", documentsRows, m.table)

	var resp []*Documents
	err := m.conn.QueryRowsCtx(ctx, &resp, query, conversationId)
	return resp, err
}

func (m *defaultDocumentsModel) UpdateContent(ctx context.Context, messageID, content string) error {
	query := "UPDATE documents SET content = ? WHERE message_id = ?"
	_, err := m.conn.ExecCtx(ctx, query, content, messageID)
	return err
}
