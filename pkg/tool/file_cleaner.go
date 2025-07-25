package tool

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

type FileCleanerCfg struct {
	Dir          string        // 绝对路径
	Retention    time.Duration // 过期时间，比如 7*24h
	MaxSizeBytes int64         // >0 超过就删, 0 不限制
}

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

func CleanOnce(ctx context.Context, cfg FileCleanerCfg) error {
	log := logx.WithContext(ctx)
	now := time.Now()

	var scanned, deleted int
	var freed int64

	err := filepath.Walk(cfg.Dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Errorf("walk err: %v", err)
			return nil
		}
		if info.IsDir() {
			return nil
		}
		scanned++

		expired := now.Sub(info.ModTime()) > cfg.Retention
		oversize := cfg.MaxSizeBytes > 0 && info.Size() > cfg.MaxSizeBytes
		if expired || oversize {
			if rmErr := os.Remove(path); rmErr != nil {
				log.Errorf("delete %s failed: %v", path, rmErr)
			} else {
				deleted++
				freed += info.Size()
				log.Infof("deleted: %s", path)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	log.Infof("FileCleaner: scanned=%d, deleted=%d, freed=%s",
		scanned, deleted, byteCountIEC(freed))
	return nil
}

func byteCountIEC(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}
