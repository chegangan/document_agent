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

type EditDocumentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	w      http.ResponseWriter
	r      *http.Request
}

func NewEditDocumentLogic(ctx context.Context, svcCtx *svc.ServiceContext, w http.ResponseWriter, r *http.Request) *EditDocumentLogic {
	return &EditDocumentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		w:      w,
		r:      r,
	}
}

func (l *EditDocumentLogic) EditDocument(req *types.EditDocumentRequest) error {
	userID, err := ctxdata.GetUidFromCtx(l.ctx)
	if err != nil {
		http.Error(l.w, "Unauthorized", http.StatusUnauthorized)
		return nil
	}

	rpcReq := &pb.EditDocumentRequest{
		UserId:           userID,
		ConversationId:   req.ConversationID,
		MessageId:        req.MessageID,
		Prompt:           req.Prompt,
		UseKnowledgeBase: req.UseKnowledgeBase,
		KnowledgeBaseId:  req.KnowledgeBaseID,
	}

	// ✅ 1) 用 l.r.Context()，让客户端断开能级联取消
	stream, err := l.svcCtx.LLMCenterRpc.EditDocument(l.r.Context(), rpcReq)
	if err != nil {
		http.Error(l.w, fmt.Sprintf("RPC error: %v", err), http.StatusInternalServerError)
		return nil
	}

	// ✅ 2) 关键响应头
	l.w.Header().Set("Content-Type", "text/event-stream")
	l.w.Header().Set("Cache-Control", "no-cache")
	l.w.Header().Set("Connection", "keep-alive")
	l.w.Header().Set("Access-Control-Allow-Origin", "*")
	l.w.Header().Set("X-Accel-Buffering", "no") // 告诉 Nginx 不要缓冲
	l.w.Header().Del("Content-Length")          // 避免被当成定长响应缓存

	flusher, ok := l.w.(http.Flusher)
	if !ok {
		http.Error(l.w, "Streaming not supported", http.StatusInternalServerError)
		return nil
	}

	// ✅ 3) 首包打洞：注释行是 SSE 允许的格式
	fmt.Fprint(l.w, ": connected\n\n")
	flusher.Flush()

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			st, _ := status.FromError(err)
			_ = l.sendSSE("error", map[string]any{
				"code":    st.Code(),
				"message": st.Message(),
			})
			return nil
		}

		switch event := resp.Event.(type) {
		case *pb.EditDocumentResponse_Message:
			_ = l.sendSSE("message", event.Message)
		case *pb.EditDocumentResponse_End:
			_ = l.sendSSE("end", event.End)
			return nil
		}

		flusher.Flush()
	}
}

func (l *EditDocumentLogic) sendSSE(event string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(l.w, "event: %s\ndata: %s\n\n", event, jsonData)
	if flusher, ok := l.w.(http.Flusher); ok {
		flusher.Flush()
	}
	return err
}
