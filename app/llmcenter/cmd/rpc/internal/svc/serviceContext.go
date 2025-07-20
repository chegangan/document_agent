package svc

import (
	"document_agent/app/llmcenter/cmd/rpc/internal/config"
	"document_agent/app/llmcenter/model"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ServiceContext struct {
	Config            config.Config
	ConversationModel model.ConversationsModel
}

func NewServiceContext(c config.Config) *ServiceContext {

	sqlConn := sqlx.NewMysql(c.DB.DataSource)

	return &ServiceContext{
		Config:            c,
		ConversationModel: model.NewConversationsModel(sqlConn),
	}
}
