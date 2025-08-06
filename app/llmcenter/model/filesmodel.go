package model

import (
	"context"
	"fmt"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"time"
)

var _ FilesModel = (*customFilesModel)(nil)

type File struct {
	Id         string    `db:"id"`
	Filename   string    `db:"filename"`
	StoredName string    `db:"stored_name"`
	CreatedAt  time.Time `db:"created_at"`
}

type (
	// FilesModel is an interface to be customized, add more methods here,
	// and implement the added methods in customFilesModel.
	FilesModel interface {
		filesModel
		InsertFile(ctx context.Context, filename, storedName string) error
		FindByStoredName(ctx context.Context, storedName string) (*File, error)
		DeleteByStoredName(ctx context.Context, storedName string) error
		withSession(session sqlx.Session) FilesModel
	}

	customFilesModel struct {
		*defaultFilesModel
	}
)

// NewFilesModel returns a model for the database table.
func NewFilesModel(conn sqlx.SqlConn) FilesModel {
	return &customFilesModel{
		defaultFilesModel: newFilesModel(conn),
	}
}

func (m *customFilesModel) withSession(session sqlx.Session) FilesModel {
	return NewFilesModel(sqlx.NewSqlConnFromSession(session))
}

func (m *customFilesModel) InsertFile(ctx context.Context, filename, storedName string) error {
	query := fmt.Sprintf("INSERT INTO %s (`filename`, `stored_name`) VALUES (?, ?)", m.table)
	_, err := m.conn.ExecCtx(ctx, query, filename, storedName)
	return err
}

func (m *defaultFilesModel) FindByStoredName(ctx context.Context, storedName string) (*File, error) {
	query := "SELECT id, filename, stored_name, created_at FROM files WHERE stored_name = ? LIMIT 1"
	var file File
	err := m.conn.QueryRowCtx(ctx, &file, query, storedName)
	if err != nil {
		return nil, err
	}
	return &file, nil
}

func (m *defaultFilesModel) DeleteByStoredName(ctx context.Context, storedName string) error {
	query := "DELETE FROM files WHERE stored_name = ?"
	_, err := m.conn.ExecCtx(ctx, query, storedName)
	return err
}
