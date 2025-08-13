package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"document_agent/app/llmcenter/cmd/api/internal/svc"
	"document_agent/app/llmcenter/cmd/api/internal/types"
	"document_agent/app/llmcenter/cmd/rpc/pb"
	"document_agent/pkg/ctxdata"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/status"
)

type ChatResumeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	w      http.ResponseWriter
	r      *http.Request
}

// [建议] 修改 handler 层，在创建 Logic 实例时传入 w 和 r
func NewChatResumeLogic(ctx context.Context, svcCtx *svc.ServiceContext, w http.ResponseWriter, r *http.Request) *ChatResumeLogic {
	return &ChatResumeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		w:      w,
		r:      r,
	}
}

// ChatResume 是处理中断后继续生成文档的 SSE 流式方法
func (l *ChatResumeLogic) ChatResume(req *types.ChatResumeRequest) error {
	// 1. 从 JWT Token 中获取用户ID
	userID, err := ctxdata.GetUidFromCtx(l.ctx)
	if err != nil {
		http.Error(l.w, "Unauthorized: Invalid token or user ID missing", http.StatusUnauthorized)
		return nil // 返回 nil 防止 go-zero 框架写入它自己的错误响应
	}

	// 2. 准备 RPC 请求
	rpcReq := &pb.ChatResumeRequest{
		UserId:         userID,
		ConversationId: req.ConversationID,
		Content:        req.Content,
		TemplateId:     req.TemplateID,
	}

	// 3. 调用 RPC 层的流式方法
	// [注意] 我们需要使用 request 的 context (l.r.Context()) 而不是 gRPC 的 context (l.ctx)，
	// 这样当客户端断开 HTTP 连接时，可以及时取消后端的 RPC 调用。
	stream, err := l.svcCtx.LLMCenterRpc.ChatResume(l.r.Context(), rpcReq)
	if err != nil {
		l.Errorf("Failed to call ChatResume RPC: %v", err)
		http.Error(l.w, fmt.Sprintf("Internal server error: %v", err), http.StatusInternalServerError)
		return nil
	}

	// 4. 设置 SSE (Server-Sent Events) 响应头
	l.w.Header().Set("Content-Type", "text/event-stream")
	l.w.Header().Set("Cache-Control", "no-cache")
	l.w.Header().Set("Connection", "keep-alive")
	l.w.Header().Set("Access-Control-Allow-Origin", "*") // [安全注意] 生产环境请设置为你的前端域名

	// 告诉 Nginx / OpenResty 不要缓冲
	l.w.Header().Set("X-Accel-Buffering", "no")
	// 某些代理看到 Content-Length 会缓存，确保没有 Content-Length
	l.w.Header().Del("Content-Length")

	flusher, ok := l.w.(http.Flusher)
	if !ok {
		l.Errorf("Streaming not supported")
		http.Error(l.w, "Streaming not supported", http.StatusInternalServerError)
		return nil
	}

	// 5. 循环接收 RPC 流并推送到 HTTP 客户端
	for {
		// 从 gRPC 流中接收消息
		resp, err := stream.Recv()

		if err == io.EOF {
			l.Infof("Resume stream finished for conversation %s.", req.ConversationID)
			return nil // 正常结束
		}

		if err != nil {
			// 从 gRPC 错误中解析 status
			st, _ := status.FromError(err)
			l.Errorf("Error receiving from resume stream forCode: %d, Message: %s",
				st.Code(), st.Message())

			// 向客户端发送一个统一的错误事件
			errorData := map[string]interface{}{
				"code":    st.Code(),
				"message": st.Message(),
			}
			// 忽略发送错误，因为此时连接可能已经断开
			_ = l.sendSSE("error", errorData)

			return nil // 返回 nil，因为错误已经通过 SSE 推送
		}

		// Resume 流程只包含 message 和 end 事件
		switch event := resp.Event.(type) {
		case *pb.ChatResumeResponse_Message:
			if err := l.sendSSE("message", event.Message); err != nil {
				l.Errorf("Failed to send message event: %v", err)
				return nil
			}
		case *pb.ChatResumeResponse_End:
			if err := l.sendSSE("end", event.End); err != nil {
				l.Errorf("Failed to send end event: %v", err)
				return nil
			}
			return nil // 收到 end 事件，流结束
		}

		// 刷新缓冲区，确保数据立即发送到客户端
		flusher.Flush()
	}
}

// sendSSE 是一个辅助函数，用于格式化并发送 SSE 事件
// [建议] 你可以将这个函数提取到一个公共的 `sse` 包中，与 ChatCompletionsLogic 共享。
func (l *ChatResumeLogic) sendSSE(event string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data for SSE: %w", err)
	}

	if _, err := fmt.Fprintf(l.w, "event: %s\n", event); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(l.w, "data: %s\n\n", jsonData); err != nil {
		return err
	}

	// 确保数据立即发送
	if flusher, ok := l.w.(http.Flusher); ok {
		flusher.Flush()
	}
	return nil
}
