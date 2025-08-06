package conversation

import (
	"net/http"

	"document_agent/app/llmcenter/cmd/api/internal/logic/conversation"
	"document_agent/app/llmcenter/cmd/api/internal/svc"
	"document_agent/app/llmcenter/cmd/api/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 根据会话ID获取该会话的最终文档列表
func GetDocumentDetailHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetDocumentDetailRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := conversation.NewGetDocumentDetailLogic(r.Context(), svcCtx)
		resp, err := l.GetDocumentDetail(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
