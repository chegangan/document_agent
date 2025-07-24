package svc

import (
	"document_agent/app/llmcenter/cmd/api/internal/config"
	"document_agent/app/llmcenter/cmd/rpc/llmcenter"

	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config       config.Config
	LLMCenterRpc llmcenter.LlmCenter
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:       c,
		LLMCenterRpc: llmcenter.NewLlmCenter(zrpc.MustNewClient(c.LLMCenterRpcConf)),
	}
}
