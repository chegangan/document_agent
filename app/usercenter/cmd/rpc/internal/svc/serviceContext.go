package svc

import (
	"document_agent/app/usercenter/cmd/rpc/internal/config"
	"document_agent/app/usercenter/model"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ServiceContext struct {
	Config    config.Config
	UserModel model.UserModel
}

func NewServiceContext(c config.Config) *ServiceContext {

	sqlConn := sqlx.NewMysql(c.DB.DataSource)

	return &ServiceContext{
		Config:    c,
		UserModel: model.NewUserModel(sqlConn),
	}
}
