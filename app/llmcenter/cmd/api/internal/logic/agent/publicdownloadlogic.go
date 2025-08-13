package agent

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"document_agent/app/llmcenter/cmd/api/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type PublicDownloadLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 公开下载（免 Header，签名校验）
func NewPublicDownloadLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PublicDownloadLogic {
	return &PublicDownloadLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PublicDownloadLogic) PublicDownload(w http.ResponseWriter, r *http.Request, pathParam string, exp int64, sig string) error {
	// 1) 过期校验
	if time.Now().Unix() > exp {
		http.Error(w, "link expired", http.StatusForbidden)
		return nil
	}

	// 2) 签名校验
	toSign := fmt.Sprintf("%s|%d", pathParam, exp)
	want := signHMAC(toSign, l.svcCtx.Config.PublicDownload.SignKey)
	if !hmac.Equal([]byte(want), []byte(sig)) {
		http.Error(w, "invalid signature", http.StatusForbidden)
		return nil
	}

	// 3) 路径清理 + 防目录穿越（只允许纯文件名）
	name := filepath.Base(filepath.Clean(pathParam))
	if name != pathParam || strings.ContainsAny(name, `/\`) {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return nil
	}

	abs := filepath.Join(l.svcCtx.Config.Upload.BaseDir, name)
	f, err := os.Open(abs)
	if err != nil {
		return err
	}
	defer f.Close()

	// 4) Content-Type & 强制下载
	ext := filepath.Ext(name)
	ctype := mime.TypeByExtension(ext)
	if ctype == "" {
		ctype = "application/octet-stream"
	}
	w.Header().Set("Content-Type", ctype)
	w.Header().Set("Content-Disposition",
		fmt.Sprintf(`attachment; filename="%s"`, url.PathEscape(name)))

	stat, _ := f.Stat()
	modTime := time.Time{}
	if stat != nil {
		modTime = stat.ModTime()
	}

	// 5) 传输
	http.ServeContent(w, r, name, modTime, f)
	return nil
}

func signHMAC(data, key string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(data))
	sum := mac.Sum(nil)
	return base64.RawURLEncoding.EncodeToString(sum)
}
