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
		return nil
	}

	// —— 新增：把 []types.Reference 转换为 []*pb.Reference —— //
	var pbRefs []*pb.Reference
	if len(req.References) > 0 {
		pbRefs = make([]*pb.Reference, 0, len(req.References))
		for _, r := range req.References {
			pbRefs = append(pbRefs, &pb.Reference{
				Type:   r.Type,   // 与 llm.api / proto 对齐：string type
				FileId: r.FileID, // 注意：proto 字段是 file_id，这里用 r.FileID
			})
		}
	}

	// 2. 准备 RPC 请求 —— 补上 Documenttype 与 References
	rpcReq := &pb.ChatResumeRequest{
		UserId:         userID,
		ConversationId: req.ConversationID,
		Content:        req.Content,
		TemplateId:     req.TemplateID,
		Documenttype:   req.Documenttype, // ✅ 新增
		References:     pbRefs,           // ✅ 新增
	}

	// 3. 调用 RPC 层的流式方法（保持不变）
	stream, err := l.svcCtx.LLMCenterRpc.ChatResume(l.r.Context(), rpcReq)
	if err != nil {
		l.Errorf("Failed to call ChatResume RPC: %v", err)
		http.Error(l.w, fmt.Sprintf("Internal server error: %v", err), http.StatusInternalServerError)
		return nil
	}

	// 4. 设置 SSE 响应头（保持不变）
	l.w.Header().Set("Content-Type", "text/event-stream")
	l.w.Header().Set("Cache-Control", "no-cache")
	l.w.Header().Set("Connection", "keep-alive")
	l.w.Header().Set("Access-Control-Allow-Origin", "*")
	l.w.Header().Set("X-Accel-Buffering", "no")
	l.w.Header().Del("Content-Length")

	flusher, ok := l.w.(http.Flusher)
	if !ok {
		l.Errorf("Streaming not supported")
		http.Error(l.w, "Streaming not supported", http.StatusInternalServerError)
		return nil
	}

	// 5. 循环接收并推送 SSE（保持不变）
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			l.Infof("Resume stream finished for conversation %s.", req.ConversationID)
			return nil
		}
		if err != nil {
			st, _ := status.FromError(err)
			l.Errorf("Error receiving from resume stream forCode: %d, Message: %s", st.Code(), st.Message())
			_ = l.sendSSE("error", map[string]any{"code": st.Code(), "message": st.Message()})
			return nil
		}

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
			return nil
		}

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
