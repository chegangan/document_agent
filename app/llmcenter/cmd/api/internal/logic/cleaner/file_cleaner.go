package cleaner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

type CleanerCfg struct {
	Dir          string
	Retention    time.Duration
	MaxSizeBytes int64 // 0 表示不限制
}

type FileCleaner struct {
	cfg CleanerCfg
	log logx.Logger
}

func NewFileCleaner(cfg CleanerCfg) *FileCleaner {
	return &FileCleaner{
		cfg: cfg,
		log: logx.WithContext(context.Background()),
	}
}

func (c *FileCleaner) CleanOnce(ctx context.Context) error {
	now := time.Now()
	var deletedCount, scanned int
	var freed int64

	err := filepath.Walk(c.cfg.Dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			c.log.Errorf("walk error: %v", err)
			return nil
		}
		if info.IsDir() {
			return nil
		}
		scanned++
		expired := now.Sub(info.ModTime()) > c.cfg.Retention
		oversize := c.cfg.MaxSizeBytes > 0 && info.Size() > c.cfg.MaxSizeBytes

		if expired || oversize {
			if rmErr := os.Remove(path); rmErr != nil {
				c.log.Errorf("delete %s failed: %v", path, rmErr)
			} else {
				deletedCount++
				freed += info.Size()
				c.log.Infof("deleted: %s", path)
			}
		}
		return nil
	})

	if err != nil {
		return err
	}

	c.log.Infof("FileCleaner done: scanned=%d, deleted=%d, freed=%s",
		scanned, deletedCount, byteCountIEC(freed))
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
