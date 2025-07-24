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
)

type ChatCompletionsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	// NOTE: 在 go-zero 的 http handler 中，直接操作 http.ResponseWriter 和 http.Request
	// 会更方便，因此我们将其添加到 Logic 结构体中。
	// 你需要在 handler 层将这两个对象传递过来。
	w http.ResponseWriter
	r *http.Request
}

// NewChatCompletionsLogic 创建一个新的聊天逻辑实例
// NOTE: 建议修改构造函数以接收 http.ResponseWriter 和 http.Request
func NewChatCompletionsLogic(ctx context.Context, svcCtx *svc.ServiceContext, w http.ResponseWriter, r *http.Request) *ChatCompletionsLogic {
	return &ChatCompletionsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		w:      w,
		r:      r,
	}
}

// ChatCompletions 是处理 SSE 流式聊天请求的核心方法
func (l *ChatCompletionsLogic) ChatCompletions(req *types.ChatCompletionsRequest) error {
	// --- 1. 从 JWT Token 中获取用户ID ---
	// NOTE: 这里的实现依赖于你的 JWT 中间件。
	// go-zero 通常会将解析后的声明(claims)存储在 context 中。
	// 假设你的 JWT claims 中有一个名为 "uid" 的字段。
	// 请根据你的实际情况修改此处的代码。
	userID, err := ctxdata.GetUidFromCtx(l.ctx)
	if err != nil {
		http.Error(l.w, "Unauthorized: Invalid token or user ID missing", http.StatusUnauthorized)
		return nil // 返回 nil 防止 go-zero 框架写入它自己的错误响应
	}

	// --- 2. 准备 RPC 请求 ---
	// 将 API 层的引用类型转换为 RPC 层的引用类型
	var rpcReferences []*pb.Reference
	for _, ref := range req.References {
		rpcReferences = append(rpcReferences, &pb.Reference{
			Type:   ref.Type,
			FileId: ref.FileID,
		})
	}

	rpcReq := &pb.ChatCompletionsRequest{
		UserId:           userID,
		ConversationId:   req.ConversationID,
		Prompt:           req.Prompt,
		UseKnowledgeBase: req.UseKnowledgeBase,
		KnowledgeBaseId:  req.KnowledgeBaseID,
		References:       rpcReferences,
	}

	// --- 3. 调用 RPC 层的流式方法 ---
	stream, err := l.svcCtx.LLMCenterRpc.ChatCompletions(l.ctx, rpcReq)
	if err != nil {
		l.Errorf("Failed to call ChatCompletions RPC: %v", err)
		http.Error(l.w, fmt.Sprintf("Internal server error: %v", err), http.StatusInternalServerError)
		return nil
	}

	// --- 4. 设置 SSE (Server-Sent Events) 响应头 ---
	// 这是实现流式响应的关键
	l.w.Header().Set("Content-Type", "text/event-stream")
	l.w.Header().Set("Cache-Control", "no-cache")
	l.w.Header().Set("Connection", "keep-alive")
	// 允许跨域请求，根据你的前端部署情况进行调整
	l.w.Header().Set("Access-Control-Allow-Origin", "*")

	// 获取 http.Flusher 接口，用于立即将数据发送到客户端
	flusher, ok := l.w.(http.Flusher)
	if !ok {
		l.Errorf("Streaming not supported")
		http.Error(l.w, "Streaming not supported", http.StatusInternalServerError)
		return nil
	}

	// --- 5. 循环接收 RPC 流并推送到 HTTP 客户端 ---
	for {
		// 从 gRPC 流中接收消息
		resp, err := stream.Recv()

		// 如果流已结束 (io.EOF)，则表示所有数据都已成功发送。
		if err == io.EOF {
			l.Infof("Stream finished for conversation %s.", req.ConversationID)
			return nil // 正常结束
		}
		// 如果在接收过程中发生其他错误（例如，上下文取消、网络问题）
		if err != nil {
			l.Errorf("Error receiving from stream: %v", err)
			// 在这里可以不发送 HTTP 错误，因为连接可能已经断开
			return nil
		}

		// 使用 switch 处理不同类型的 SSE 事件
		switch event := resp.Event.(type) {
		case *pb.ChatCompletionsResponse_Message:
			// 这是普通的文本流事件
			if err := l.sendSSE("message", event.Message); err != nil {
				l.Errorf("Failed to send message event: %v", err)
				return nil
			}
		case *pb.ChatCompletionsResponse_Interrupt:
			// 这是中断事件，提示前端需要用户交互
			if err := l.sendSSE("interrupt", event.Interrupt); err != nil {
				l.Errorf("Failed to send interrupt event: %v", err)
				return nil
			}
			// NOTE: 中断事件后，RPC 流会主动关闭，
			// 所以下一次 stream.Recv() 会收到 io.EOF，循环将正常退出。
		case *pb.ChatCompletionsResponse_End:
			// 这是结束事件，标志着本次交互的完成
			if err := l.sendSSE("end", event.End); err != nil {
				l.Errorf("Failed to send end event: %v", err)
				return nil
			}
			// 收到 end 事件后，也意味着流的结束。
			return nil
		}

		// 每次发送后，刷新缓冲区，确保数据立即发送到客户端
		flusher.Flush()
	}
}

// sendSSE 是一个辅助函数，用于格式化并发送 SSE 事件
func (l *ChatCompletionsLogic) sendSSE(event string, data interface{}) error {
	// 将数据结构序列化为 JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data for SSE: %w", err)
	}

	// 按照 SSE 格式写入数据
	// 格式:
	// event: <event_name>\n
	// data: <json_string>\n\n
	if _, err := fmt.Fprintf(l.w, "event: %s\n", event); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(l.w, "data: %s\n\n", jsonData); err != nil {
		return err
	}
	return nil
}
