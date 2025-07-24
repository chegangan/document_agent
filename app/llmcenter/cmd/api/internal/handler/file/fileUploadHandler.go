package file

import (
	"net/http"

	"document_agent/app/llmcenter/cmd/api/internal/logic/file"
	"document_agent/app/llmcenter/cmd/api/internal/svc"
	xhttp "github.com/zeromicro/x/http"
)

// 上传文件, 用于后续对话
func FileUploadHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseMultipartForm(20 << 20) // 限制20MB
		if err != nil {
			xhttp.JsonBaseResponseCtx(r.Context(), w, err)
			return
		}

		logic := file.NewFileUploadLogic(r.Context(), svcCtx)
		resp, err := logic.FileUpload(r.MultipartForm)
		if err != nil {
			xhttp.JsonBaseResponseCtx(r.Context(), w, err)
		} else {
			xhttp.JsonBaseResponseCtx(r.Context(), w, resp)
		}
	}
}
