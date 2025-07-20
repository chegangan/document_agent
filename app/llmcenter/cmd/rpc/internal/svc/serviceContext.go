package svc

import (
	"document_agent/app/llmcenter/cmd/rpc/internal/config" // 你的 Config 定义包
	"document_agent/app/llmcenter/model"                   // goctl 生成的 model 包
	"github.com/zeromicro/go-zero/core/stores/sqlx"        // 用来建 DB 连接
)

type ServiceContext struct {
	Config             config.Config
	ConversationsModel model.ConversationsModel
	MessagesModel      model.MessagesModel
	// …如果还有别的 model，也在这里加
}

func NewServiceContext(c config.Config) *ServiceContext {
	// 改为读取 c.DB.DataSource
	conn := sqlx.NewMysql(c.DB.DataSource)
	return &ServiceContext{
		Config:             c,
		ConversationsModel: model.NewConversationsModel(conn),
		MessagesModel:      model.NewMessagesModel(conn),
	}
}
