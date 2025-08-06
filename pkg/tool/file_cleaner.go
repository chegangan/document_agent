package tool

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"document_agent/app/llmcenter/model"
	"github.com/zeromicro/go-zero/core/logx"
)

type FileCleanerCfg struct {
	Dir          string        // 绝对路径
	Retention    time.Duration // 过期时间，比如 7*24h
	MaxSizeBytes int64         // >0 超过就删, 0 不限制
}

// StartFileCleaner 启动一个循环定时任务
func StartFileCleaner(cfg FileCleanerCfg, interval time.Duration, filesModel model.FilesModel) {
	_ = CleanOnce(context.Background(), cfg, filesModel)

	tk := time.NewTicker(interval)
	defer tk.Stop()

	for range tk.C {
		_ = CleanOnce(context.Background(), cfg, filesModel)
	}
}

func CleanOnce(ctx context.Context, cfg FileCleanerCfg, filesModel model.FilesModel) error {
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

				// 删除数据库记录
				storedName := filepath.Base(path)
				if err := filesModel.DeleteByStoredName(ctx, storedName); err != nil {
					log.Errorf("删除数据库记录失败 stored_name=%s: %v", storedName, err)
				} else {
					log.Infof("数据库记录已删除 stored_name=%s", storedName)
				}
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
