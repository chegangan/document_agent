package svc

import (
	"document_agent/app/llmcenter/cmd/api/internal/config"
	"document_agent/app/llmcenter/cmd/rpc/llmcenter"

	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config       config.Config
	LlmCenterRpc llmcenter.LlmCenter
}

/*
func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:       c,
		LlmCenterRpc: llmcenter.NewLlmCenter(zrpc.MustNewClient(c.LlmCenterRpcConf)),
	}
}*/

func NewServiceContext(c config.Config) *ServiceContext {
	rpcClientConf := zrpc.RpcClientConf{
		Endpoints: []string{"localhost:8080"},
		NonBlock:  true,
	}

	return &ServiceContext{
		Config:       c,
		LlmCenterRpc: llmcenter.NewLlmCenter(zrpc.MustNewClient(rpcClientConf)),
	}
}
