package agent

import (
	"net/http"

	"document_agent/app/llmcenter/cmd/api/internal/logic/agent"
	"document_agent/app/llmcenter/cmd/api/internal/svc"
	"document_agent/app/llmcenter/cmd/api/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 公开下载（免 Header，签名校验）
func PublicDownloadHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.PublicDownloadRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		l := agent.NewPublicDownloadLogic(r.Context(), svcCtx)
		// 传入 r
		if err := l.PublicDownload(w, r, req.Path, req.Exp, req.Sig); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		}
	}
}
