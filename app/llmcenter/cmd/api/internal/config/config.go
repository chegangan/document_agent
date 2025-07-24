package config

import (
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	rest.RestConf
	Auth struct {
		AccessSecret string
	}
	Upload struct {
		BaseDir string
	}

	Etcd struct {
		Hosts []string
		Key   string
	}
	FileCleaner struct {
		Enable          bool
		Dir             string
		RetentionDays   int
		IntervalMinutes int
		MaxSizeMB       int64
		UseEtcdLock     bool
		LockKey         string
	}
	LlmCenterRpcConf zrpc.RpcClientConf
}
