package svc

import (
	"os"
	"path/filepath"
	"time"

	"document_agent/app/llmcenter/cmd/api/internal/config"
	"document_agent/app/llmcenter/cmd/rpc/llmcenter"
	"document_agent/pkg/tool"

	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config       config.Config
	LLMCenterRpc llmcenter.LlmCenter
}

func NewServiceContext(c config.Config) *ServiceContext {
	// 1. 归一化上传目录
	dir := c.Upload.BaseDir
	if !filepath.IsAbs(dir) {
		if abs, err := filepath.Abs(dir); err == nil {
			dir = abs
		}
	}
	dir = filepath.Clean(dir)
	_ = os.MkdirAll(dir, 0o755)
	c.Upload.BaseDir = dir

	// 2. 初始化 svc
	svc := &ServiceContext{
		Config:       c,
		LLMCenterRpc: llmcenter.NewLlmCenter(zrpc.MustNewClient(c.LlmCenterRpcConf)),
	}

	// 3. 启动文件清理（无 etcd 锁，后期可加）
	if c.FileCleaner.Enable {
		go tool.StartFileCleaner(
			tool.FileCleanerCfg{
				Dir:          c.FileCleaner.Dir,
				Retention:    time.Duration(c.FileCleaner.RetentionDays) * 24 * time.Hour,
				MaxSizeBytes: c.FileCleaner.MaxSizeMB * 1024 * 1024,
			},
			time.Duration(c.FileCleaner.IntervalMinutes)*time.Minute,
		)
	}

	return svc
}
