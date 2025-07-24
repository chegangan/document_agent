// [手动修改]
// go-zero 生成的 handler 默认不传入 w 和 r，你需要手动修改它。
package chat

import (
	"net/http"

	"document_agent/app/llmcenter/cmd/api/internal/logic/chat"
	"document_agent/app/llmcenter/cmd/api/internal/svc"
	"document_agent/app/llmcenter/cmd/api/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func ChatResumeHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ChatResumeRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		// 注意这里的变化：我们直接将 w 和 r 传给了 NewChatResumeLogic
		l := chat.NewChatResumeLogic(r.Context(), svcCtx, w, r)
		// 因为 Logic 内部直接处理了响应流，所以这里的 err 通常为 nil
		// 即使有错误，也已经在 Logic 内部通过 http.Error 处理了
		if err := l.ChatResume(&req); err != nil {
			// 通常这里的错误已经在 logic 层处理并返回了，这里只做日志记录
			// logx.FromContext(r.Context()).Errorf("ChatResumeHandler error: %v", err)
		}
	}
}
