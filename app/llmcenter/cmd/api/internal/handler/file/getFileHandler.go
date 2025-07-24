package file

import (
	"net/http"

	"document_agent/app/llmcenter/cmd/api/internal/logic/file"
	"document_agent/app/llmcenter/cmd/api/internal/svc"
	"document_agent/app/llmcenter/cmd/api/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 按相对路径获取文件
func GetFileHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetFileReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := file.NewGetFileLogic(r.Context(), svcCtx)
		if err := l.GetFile(&req, w, r); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		}
	}
}
