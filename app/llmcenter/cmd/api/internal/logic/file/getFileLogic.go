// internal/logic/file/getfilelogic.go
package file

import (
	"context"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"document_agent/app/llmcenter/cmd/api/internal/svc"
	"document_agent/app/llmcenter/cmd/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetFileLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetFileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetFileLogic {
	return &GetFileLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// 直接输出文件流
func (l *GetFileLogic) GetFile(req *types.GetFileReq, w http.ResponseWriter, r *http.Request) error {
	// 0. 参数校验
	if req.Path == "" {
		http.Error(w, "missing path", http.StatusBadRequest)
		return nil
	}

	base := filepath.Clean(l.svcCtx.Config.Upload.BaseDir)
	// 去掉开头的斜杠，防止 Join 时被认为是绝对路径
	rel := filepath.Clean(strings.TrimLeft(req.Path, `/\`))
	full := filepath.Join(base, rel)

	// 防目录穿越
	relCheck, err := filepath.Rel(base, full)
	if err != nil || strings.HasPrefix(relCheck, "..") {
		http.Error(w, "invalid path", http.StatusForbidden)
		return nil
	}

	l.Infof("GetFile path=%q base=%q full=%q relCheck=%q", req.Path, base, full, relCheck)

	f, err := os.Open(full)
	if err != nil {
		l.Errorf("open file %s error: %v", full, err)
		if os.IsNotExist(err) {
			http.NotFound(w, r)
			return nil
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return err
	}
	if fi.IsDir() {
		http.Error(w, "not a file", http.StatusBadRequest)
		return nil
	}

	// Content-Type
	if ct := mime.TypeByExtension(filepath.Ext(full)); ct != "" {
		w.Header().Set("Content-Type", ct)
	} else {
		buf := make([]byte, 512)
		n, _ := f.Read(buf)
		w.Header().Set("Content-Type", http.DetectContentType(buf[:n]))
		_, _ = f.Seek(0, 0)
	}

	// 缓存、长度
	w.Header().Set("Cache-Control", "public, max-age=31536000")
	w.Header().Set("Content-Length", strconv.FormatInt(fi.Size(), 10))
	// 可选：inline/attachment
	// w.Header().Set("Content-Disposition", `inline; filename="`+fi.Name()+`"`)

	// 真正写出
	if _, err = io.Copy(w, f); err != nil {
		l.Errorf("copy file error: %v", err)
		// 这里一般不再回错，因为头和部分内容已写出
	}
	return nil
}
