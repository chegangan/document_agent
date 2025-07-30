package logic

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
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
	"document_agent/app/llmcenter/model"
	"document_agent/pkg/tool"
	"document_agent/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type ChatCompletionsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewChatCompletionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ChatCompletionsLogic {
	return &ChatCompletionsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// ChatCompletions 是处理聊天请求的核心 RPC 方法
func (l *ChatCompletionsLogic) ChatCompletions(in *pb.ChatCompletionsRequest, stream pb.LlmCenter_ChatCompletionsServer) error {
	// 1. 获取或创建会话，并获取历史消息
	conversationID, historyMessages, err := l.getOrCreateConversation(in.UserId, in.ConversationId, in.Prompt)
	if err != nil {
		return err
	}

	// 2. 保存当前用户发送的消息
	if err := l.saveUserMessage(conversationID, in.Prompt); err != nil {
		return err
	}

	// 3. 处理文件上传（如图片）
	imgURL, err := l.handleFileUploads(in.References)
	if err != nil {
		// 仅记录错误，不中断流程
		l.Errorf("handleFileUploads failed: %v", err)
	}

	// 4. 构建对大模型的请求
	llmReq := l.buildLLMRequest(in.UserId, conversationID, in.Prompt, historyMessages, imgURL)
	reqBody, err := json.Marshal(llmReq)
	if err != nil {
		l.Errorf("failed to marshal llm request: %v :%w", err, xerr.ErrRequestParam)
		return err
	}

	// 5. 调用大模型 API 并处理流式响应
	return l.processLLMStream(reqBody, stream, conversationID, in.References)
}

// saveUserMessage 保存用户消息到数据库
func (l *ChatCompletionsLogic) saveUserMessage(conversationID, prompt string) error {
	userMessage := &model.Messages{
		MessageId:      tool.GenerateULID(),
		ConversationId: conversationID,
		Role:           "user",
		Content:        prompt,
		ContentType:    "text",
	}
	_, err := l.svcCtx.MessageModel.Insert(l.ctx, userMessage)
	if err != nil {
		return fmt.Errorf("saveUserMessage db message Insert err:%+v, message:%+v: %w", err, userMessage, xerr.ErrDbError)
	}
	return nil
}

// handleFileUploads 处理文件上传并返回文件URL
func (l *ChatCompletionsLogic) handleFileUploads(references []*pb.Reference) (string, error) {
	if len(references) == 0 {
		return "", nil
	}
	var imgURL string
	var lastErr error
	for _, ref := range references {
		if ref.Type == "file" {
			localPath := filepath.Join(l.svcCtx.Config.Upload.BaseDir, ref.FileId)
			url, err := uploadImageToXingHuo(localPath, l.svcCtx.Config.XingChen.ApiKey, l.svcCtx.Config.XingChen.ApiSecret)
			if err != nil {
				l.Errorf("图片上传失败：file_id=%s err=%v", ref.FileId, err)
				lastErr = err // 记录最后一次错误
				continue
			}
			imgURL = url // 目前只支持一张图片
			break
		}
	}
	return imgURL, lastErr
}

// getOrCreateConversation 	获取或创建会话，并获取历史消息
func (l *ChatCompletionsLogic) getOrCreateConversation(userID int64, convID, prompt string) (string, []*pb.Message, error) { // 返回的 Message 应该是数据库模型
	if convID == "" {
		// 创建新会话
		newConvID := tool.GenerateULID()
		newConversation := &model.Conversations{
			ConversationId: newConvID,
			UserId:         userID,
			Title:          l.generateTitle(prompt), // 使用 prompt 生成一个初始标题
		}

		_, err := l.svcCtx.ConversationModel.Insert(l.ctx, newConversation)
		if err != nil {
			return "", nil, fmt.Errorf("getOrCreateConversation db conversation Insert err:%+v, conversation:%+v,: %w", err, newConversation, xerr.ErrDbError)
		}

		return newConvID, []*pb.Message{}, nil
	}

	// 如果提供了 convID，则尝试从数据库获取会话
	conversation, err := l.svcCtx.ConversationModel.FindOne(l.ctx, convID)
	if err != nil {
		return "", nil, fmt.Errorf("getOrCreateConversation db conversation FindOne err:%+v, conversationId:%s: %w", err, convID, xerr.ErrConversationNotFound)
	}
	if conversation.UserId != userID {
		return "", nil, fmt.Errorf("getOrCreateConversation 该用户id无法访问此会话 userId:%d, conversationId:%s: %w", userID, convID, xerr.ErrConversationAccessDenied)
	}

	// 从数据库获取该会话的历史消息
	getConversationDetailLogic := NewGetConversationDetailLogic(l.ctx, l.svcCtx)
	GetConversationDetailResponse, err := getConversationDetailLogic.GetConversationDetail(&pb.GetConversationDetailRequest{
		ConversationId: convID,
	})
	if err != nil {
		return "", nil, fmt.Errorf("getOrCreateConversation db message FindAllByConversationID err:%+v, conversationId:%s: %w", err, convID, xerr.ErrMessageNotFound)
	}
	return convID, GetConversationDetailResponse.GetHistory(), nil
}

// buildLLMRequest 构建发送给星火大模型 API 的请求体
func (l *ChatCompletionsLogic) buildLLMRequest(userID int64, convID, prompt string, history []*pb.Message, imgUrl string) types.LLMApiRequest {

	// 只取最近的10条历史消息，history是按照时间顺序排列的，所以最新的消息在最后
	start := 0
	if len(history) > 10 {
		start = len(history) - 10
	}
	apiHistory := make([]types.LLMMessage, 0, len(history)-start)
	for _, msg := range history[start:] {

		apiHistory = append(apiHistory, types.LLMMessage{
			Role:        msg.Role,
			ContentType: "text",
			Content:     msg.Content,
		})
	}

	flowID := l.svcCtx.Config.XingChen.FlowID
	// TODO: 这里的parameters是工作流一开始的输入，默认为空，多模态输入加在这里
	return types.LLMApiRequest{
		FlowID: flowID,
		UID:    fmt.Sprintf("%d", userID),
		Parameters: types.LLMParameters{
			AgentUserInput: prompt,
			Img:            imgUrl,
		},
		Stream:  true,
		ChatID:  convID,
		History: apiHistory,
	}
}

// processLLMStream 调用大模型API，处理返回的流，并推送到客户端gRPC流
// processLLMStream 调用大模型API，处理返回的流，并推送到客户端gRPC流
func (l *ChatCompletionsLogic) processLLMStream(reqBody []byte, stream pb.LlmCenter_ChatCompletionsServer, conversationID string, references []*pb.Reference) error {
	apiURL := l.svcCtx.Config.XingChen.ApiURL
	apiKey := l.svcCtx.Config.XingChen.ApiKey
	apiSecret := l.svcCtx.Config.XingChen.ApiSecret
	authToken := fmt.Sprintf("Bearer %s:%s", apiKey, apiSecret)

	req, err := http.NewRequestWithContext(l.ctx, "POST", apiURL, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create http request with context: %+v:%w", err, xerr.ErrLLMApiCancel)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authToken)
	req.Header.Set("Accept", "text/event-stream")

	client := l.svcCtx.LlmApiClient
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call llm api: %+v:%w", err, xerr.ErrLLMApiCancel)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("llm api returned non-200 status: %d, body: %s :%w",
			resp.StatusCode, string(bodyBytes), xerr.ErrLLMApiError)
	}

	assistantReply, err := l.handleStreamEvents(resp.Body, stream, conversationID)
	if err != nil {
		return err
	}

	assistantMessageID, err := l.saveAssistantMessage(conversationID, assistantReply, references)
	if err != nil {
		// 保存失败也应通知前端结束，所以只记录日志不返回错误
		l.Errorf("saveAssistantMessage failed: %v", err)
	}

	return l.sendEndEvent(stream, conversationID, assistantMessageID)
}

// handleStreamEvents 处理从LLM返回的SSE事件流
func (l *ChatCompletionsLogic) handleStreamEvents(body io.Reader, stream pb.LlmCenter_ChatCompletionsServer, conversationID string) (string, error) {
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
			l.Errorf("failed to unmarshal llm stream line: %s, error: %v", line, err)
			continue
		}

		if apiResp.Code != 0 {
			return "", fmt.Errorf("LLM API error response: code=%d, message=%s :%w",
				apiResp.Code, apiResp.Message, xerr.ErrLLMApiError)
		}

		// 处理中断事件
		if apiResp.EventData != nil && apiResp.EventData.EventType == "interrupt" {
			if err := l.handleInterruptEvent(&apiResp, stream, conversationID); err != nil {
				return "", err
			}
			return "", nil // 中断后，当前流式交互结束，返回空字符串表示没有完整回复
		}

		// 处理正常消息流
		if len(apiResp.Choices) > 0 {
			chunk := apiResp.Choices[0].Delta.Content
			if chunk != "" {
				assistantReply.WriteString(chunk)
				messageEvent := &pb.SSEMessageEvent{Chunk: chunk}
				if err := stream.Send(&pb.ChatCompletionsResponse{Event: &pb.ChatCompletionsResponse_Message{Message: messageEvent}}); err != nil {
					return "", fmt.Errorf("failed to send message chunk to client: %v:%w", err, xerr.ErrLLMApiCancel)
				}
			}
			if apiResp.Choices[0].FinishReason == "stop" {
				break // 正常结束
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading llm stream: %v:%w", err, xerr.ErrLLMApiError)
	}

	return assistantReply.String(), nil
}

// handleInterruptEvent 处理中断事件
func (l *ChatCompletionsLogic) handleInterruptEvent(apiResp *types.LLMApiResponse, stream pb.LlmCenter_ChatCompletionsServer, conversationID string) error {
	redisKey := fmt.Sprintf("llm:interrupt:%s", conversationID)
	// 1200 秒 = 20 分钟
	err := l.svcCtx.RedisClient.Setex(redisKey, apiResp.EventData.EventID, 1200)
	if err != nil {
		return fmt.Errorf("failed to set interrupt key in redis for conv %s: %v :%w",
			conversationID, err, xerr.ErrLLMInterruptEventNotSet)
	}
	l.Infof("Interrupt event received for conv %s. Storing EventID %s in Redis.", conversationID, apiResp.EventData.EventID)

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

// saveAssistantMessage 保存助手回复到数据库
func (l *ChatCompletionsLogic) saveAssistantMessage(conversationID string, reply string, references []*pb.Reference) (string, error) {
	if reply == "" { // 如果没有回复内容（例如，在中断事件后），则不保存
		return "", nil
	}

	referencesData, err := json.Marshal(references)
	if err != nil {
		return "", fmt.Errorf("failed to marshal references: %v:%w", err, xerr.ErrRequestParam)
	}

	assistantMessageID := tool.GenerateULID()
	assistantMessage := &model.Messages{
		MessageId:      assistantMessageID,
		ConversationId: conversationID,
		Role:           "assistant",
		Content:        reply,
		ContentType:    "document_outline",
		Metadata: sql.NullString{
			String: string(referencesData),
			Valid:  len(referencesData) > 0 && string(referencesData) != "null",
		},
	}

	_, err = l.svcCtx.MessageModel.Insert(l.ctx, assistantMessage)
	if err != nil {
		return "", fmt.Errorf("saveAssistantMessage db Insert err:%+v, message:%+v: %w", err, assistantMessage, xerr.ErrDbError)
	}
	return assistantMessageID, nil
}

// sendEndEvent 向客户端发送结束事件
func (l *ChatCompletionsLogic) sendEndEvent(stream pb.LlmCenter_ChatCompletionsServer, conversationID, messageID string) error {
	endEvent := &pb.SSEEndEvent{
		ConversationId: conversationID,
		MessageId:      messageID,
	}
	if err := stream.Send(&pb.ChatCompletionsResponse{Event: &pb.ChatCompletionsResponse_End{End: endEvent}}); err != nil {
		return fmt.Errorf("failed to send end event to client: %v:%w", err, xerr.ErrLLMApiCancel)
	}
	return nil
}

// generateTitle 根据用户的第一条消息生成一个简单的标题
func (l *ChatCompletionsLogic) generateTitle(prompt string) string {
	// 将 prompt 转换为 rune 数组以正确处理多字节字符（如中文）
	runes := []rune(prompt)
	maxLength := 15 // 标题最大长度
	if len(runes) > maxLength {
		return string(runes[:maxLength]) + "..."
	}
	return string(runes)
}

//=============================================================================================================

func uploadImageToXingHuo(filePath, apiKey, apiSecret string) (string, error) {
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

	req, err := http.NewRequest("POST", "https://xingchen-api.xf-yun.com/workflow/v1/upload_file", &buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s:%s", apiKey, apiSecret))
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
