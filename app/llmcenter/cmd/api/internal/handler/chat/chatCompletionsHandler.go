package chat

import (
	"net/http"

	"document_agent/app/llmcenter/cmd/api/internal/logic/chat"
	"document_agent/app/llmcenter/cmd/api/internal/svc"
	"document_agent/app/llmcenter/cmd/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
	xhttp "github.com/zeromicro/x/http"
)

// 发起新对话或在现有对话中发送消息
func ChatCompletionsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ChatCompletionsRequest
		if err := httpx.Parse(r, &req); err != nil {
			xhttp.JsonBaseResponseCtx(r.Context(), w, err)
			return
		}

		// 注意这里的变化
		l := chat.NewChatCompletionsLogic(r.Context(), svcCtx, w, r)
		err := l.ChatCompletions(&req)

		// 因为所有响应（包括错误）都在 logic 中直接写入 ResponseWriter，
		// 所以这里不需要再调用 httpx.OkJsonCtx 或 httpx.ErrorCtx。
		// 如果 logic 返回错误，仅用于记录日志。
		if err != nil {
			logx.WithContext(r.Context()).Errorf("ChatCompletionsHandler error: %v", err)
		}
	}
}
