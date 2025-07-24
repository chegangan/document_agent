package tool

import (
	"context"
	"time"
)

// StartFileCleaner 启动一个循环定时任务
func StartFileCleaner(cfg FileCleanerCfg, interval time.Duration) {
	// 先跑一次
	_ = CleanOnce(context.Background(), cfg)

	tk := time.NewTicker(interval)
	defer tk.Stop()

	for range tk.C {
		_ = CleanOnce(context.Background(), cfg)
	}
}
