package chat

import (
	"net/http"

	"document_agent/app/llmcenter/cmd/api/internal/logic/chat"
	"document_agent/app/llmcenter/cmd/api/internal/svc"
	"document_agent/app/llmcenter/cmd/api/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 根据修改提示编辑现有文章 (SSE 流式响应)
func EditDocumentHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.EditDocumentRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := chat.NewEditDocumentLogic(r.Context(), svcCtx, w, r)
		_ = l.EditDocument(&req)
	}
}
