package svc

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"document_agent/app/llmcenter/cmd/api/internal/config"
	"document_agent/app/llmcenter/cmd/api/internal/logic/cleaner"
	"document_agent/app/llmcenter/cmd/rpc/llmcenter"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/zrpc"
	"go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

type ServiceContext struct {
	Config       config.Config
	LlmCenterRpc llmcenter.LlmCenter
}

func NewServiceContext(c config.Config) *ServiceContext {
	s := &ServiceContext{Config: c}

	if c.FileCleaner.Enable {
		go s.startFileCleaner() // <--- 这一句一定要有
	}

	dir := c.Upload.BaseDir
	if !filepath.IsAbs(dir) {
		if abs, err := filepath.Abs(dir); err == nil {
			dir = abs
		}
	}
	dir = filepath.Clean(dir)
	_ = os.MkdirAll(dir, 0o755)

	c.Upload.BaseDir = dir
	return &ServiceContext{
		Config:       c,
		LlmCenterRpc: llmcenter.NewLlmCenter(zrpc.MustNewClient(c.LlmCenterRpcConf)),
	}
}

func (s *ServiceContext) startFileCleaner() {
	interval := time.Duration(s.Config.FileCleaner.IntervalMinutes) * time.Minute
	retention := time.Duration(s.Config.FileCleaner.RetentionDays) * 24 * time.Hour

	cl := cleaner.NewFileCleaner(cleaner.CleanerCfg{
		Dir:          s.Config.FileCleaner.Dir, // 已是绝对路径
		Retention:    retention,
		MaxSizeBytes: s.Config.FileCleaner.MaxSizeMB * 1024 * 1024,
	})

	// 先跑一次
	s.runWithLock(cl)

	tk := time.NewTicker(interval)
	defer tk.Stop()

	for range tk.C {
		s.runWithLock(cl)
	}
}

// 带 etcd 分布式锁
func (s *ServiceContext) runWithLock(cl *cleaner.FileCleaner) {
	if !s.Config.FileCleaner.UseEtcdLock {
		_ = cl.CleanOnce(context.Background())
		return
	}

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   s.Config.Etcd.Hosts,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		logx.Errorf("etcd client error: %v", err)
		return
	}
	defer cli.Close()

	sess, err := concurrency.NewSession(cli, concurrency.WithTTL(10))
	if err != nil {
		logx.Errorf("etcd session error: %v", err)
		return
	}
	defer sess.Close()

	mu := concurrency.NewMutex(sess, s.Config.FileCleaner.LockKey)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err = mu.Lock(ctx); err != nil {
		logx.Infof("filecleaner lock not acquired: %v", err)
		return
	}
	defer func() { _ = mu.Unlock(context.Background()) }()

	if err := cl.CleanOnce(context.Background()); err != nil {
		logx.Errorf("cleaner error: %v", err)
	}
}
