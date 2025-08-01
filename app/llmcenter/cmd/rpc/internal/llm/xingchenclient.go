package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"document_agent/app/llmcenter/cmd/rpc/internal/svc"
	"document_agent/app/llmcenter/cmd/rpc/pb"
	"document_agent/app/llmcenter/cmd/rpc/types"
	"document_agent/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

// XingChenClient 封装了与星火大模型 API 的交互
type XingChenClient struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

// sseEventHandler 是一个函数类型，用于处理不同类型的 SSE 事件
// 它返回一个布尔值，指示流是否应该因为特定事件（如 interrupt）而提前终止
type sseEventHandler func(resp *types.LLMApiResponse) (shouldStop bool, err error)

// NewXingChenClient 创建一个新的星火客户端实例
func NewXingChenClient(ctx context.Context, svcCtx *svc.ServiceContext) *XingChenClient {
	return &XingChenClient{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// UploadImage 上传图片到星火并返回 URL
func (c *XingChenClient) UploadImage(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("无法打开文件 %s: %w", filePath, err)
	}
	defer file.Close()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return "", err
	}
	if _, err = io.Copy(part, file); err != nil {
		return "", err
	}
	writer.Close()

	req, err := http.NewRequest("POST", c.svcCtx.Config.XingChen.UploadURL, &buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", c.getAuthToken())
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			URL string `json:"url"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.Code != 0 {
		return "", fmt.Errorf("上传失败 code=%d: %s", result.Code, result.Message)
	}
	return result.Data.URL, nil
}

// StreamChat 调用大模型 API 并处理流式响应
func (c *XingChenClient) StreamChat(reqBody []byte, stream pb.LlmCenter_ChatCompletionsServer, conversationID string) (string, error) {
	resp, err := c.doStreamRequest(c.svcCtx.Config.XingChen.ApiURL, reqBody)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// 定义 Chat 流程的事件处理器
	handler := func(apiResp *types.LLMApiResponse) (bool, error) {
		// 检查中断事件
		if apiResp.EventData != nil && apiResp.EventData.EventType == "interrupt" {
			if err := c.handleInterruptEvent(apiResp, stream, conversationID); err != nil {
				return true, err // 中断处理失败，停止流
			}
			return true, nil // 中断后，当前流式交互结束
		}

		// 发送消息块
		if len(apiResp.Choices) > 0 && apiResp.Choices[0].Delta.Content != "" {
			messageEvent := &pb.SSEMessageEvent{Chunk: apiResp.Choices[0].Delta.Content}
			if err := stream.Send(&pb.ChatCompletionsResponse{Event: &pb.ChatCompletionsResponse_Message{Message: messageEvent}}); err != nil {
				return true, fmt.Errorf("failed to send message chunk to client: %v:%w", err, xerr.ErrLLMApiCancel)
			}
		}
		return false, nil
	}

	return c.processStreamResponse(resp.Body, handler)
}

// StreamResume 调用大模型 Resume API 并处理流式响应
func (c *XingChenClient) StreamResume(reqBody []byte, stream pb.LlmCenter_ChatResumeServer) (string, error) {
	resp, err := c.doStreamRequest(c.svcCtx.Config.XingChen.ApiResumeURL, reqBody)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// 定义 Resume 流程的事件处理器
	handler := func(apiResp *types.LLMApiResponse) (bool, error) {
		// Resume 流程不处理 interrupt 事件，只发送消息块
		if len(apiResp.Choices) > 0 && apiResp.Choices[0].Delta.Content != "" {
			messageEvent := &pb.SSEMessageEvent{Chunk: apiResp.Choices[0].Delta.Content}
			if err := stream.Send(&pb.ChatResumeResponse{Event: &pb.ChatResumeResponse_Message{Message: messageEvent}}); err != nil {
				return true, fmt.Errorf("failed to send message chunk to client: %v:%w", err, xerr.ErrLLMApiCancel)
			}
		}
		return false, nil
	}

	return c.processStreamResponse(resp.Body, handler)
}

// doStreamRequest 创建并执行一个流式API请求
func (c *XingChenClient) doStreamRequest(url string, reqBody []byte) (*http.Response, error) {
	req, err := http.NewRequestWithContext(c.ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %+v:%w", err, xerr.ErrLLMApiCancel)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", c.getAuthToken())
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.svcCtx.LlmApiClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call llm api: %+v:%w", err, xerr.ErrLLMApiCancel)
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close() // 确保在出错时也关闭 body
		return nil, fmt.Errorf("llm api returned non-200 status: %d, body: %s :%w",
			resp.StatusCode, string(bodyBytes), xerr.ErrLLMApiError)
	}

	return resp, nil
}

// processStreamResponse 是处理 SSE 流的核心逻辑
func (c *XingChenClient) processStreamResponse(body io.Reader, handler sseEventHandler) (string, error) {
	scanner := bufio.NewScanner(body)
	var assistantReply strings.Builder

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || !strings.HasPrefix(line, "data: ") {
			continue
		}

		line = strings.TrimPrefix(line, "data: ")
		var apiResp types.LLMApiResponse
		if err := json.Unmarshal([]byte(line), &apiResp); err != nil {
			c.Errorf("failed to unmarshal llm stream line: %s, error: %v", line, err)
			continue
		}

		if apiResp.Code != 0 {
			return "", fmt.Errorf("LLM API error response: code=%d, message=%s :%w",
				apiResp.Code, apiResp.Message, xerr.ErrLLMApiError)
		}

		// 使用回调处理特定事件
		shouldStop, err := handler(&apiResp)
		if err != nil {
			return "", err // 如果回调出错，直接返回错误
		}
		if shouldStop {
			return "", nil // 回调要求停止（如 interrupt），返回空回复
		}

		if len(apiResp.Choices) > 0 {
			assistantReply.WriteString(apiResp.Choices[0].Delta.Content)
			if apiResp.Choices[0].FinishReason == "stop" {
				break
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading llm stream: %v:%w", err, xerr.ErrLLMApiError)
	}

	return assistantReply.String(), nil
}

func (c *XingChenClient) getAuthToken() string {
	return fmt.Sprintf("Bearer %s:%s", c.svcCtx.Config.XingChen.ApiKey, c.svcCtx.Config.XingChen.ApiSecret)
}

func (c *XingChenClient) handleInterruptEvent(apiResp *types.LLMApiResponse, stream pb.LlmCenter_ChatCompletionsServer, conversationID string) error {
	redisKey := fmt.Sprintf("llm:interrupt:%s", conversationID)
	err := c.svcCtx.RedisClient.Setex(redisKey, apiResp.EventData.EventID, 1200) // 20分钟
	if err != nil {
		return fmt.Errorf("failed to set interrupt key in redis for conv %s: %v :%w",
			conversationID, err, xerr.ErrLLMInterruptEventNotSet)
	}
	c.Infof("Interrupt event received for conv %s. Storing EventID %s in Redis.", conversationID, apiResp.EventData.EventID)

	interruptEvent := &pb.SSEInterruptEvent{
		ConversationId: conversationID,
		ContentType:    "document_outline",
		Content:        apiResp.EventData.Value.Content,
	}
	if err := stream.Send(&pb.ChatCompletionsResponse{Event: &pb.ChatCompletionsResponse_Interrupt{Interrupt: interruptEvent}}); err != nil {
		return fmt.Errorf("failed to send interrupt event to client: %v:%w", err, xerr.ErrLLMInterruptEventNotSet)
	}
	return nil
}
