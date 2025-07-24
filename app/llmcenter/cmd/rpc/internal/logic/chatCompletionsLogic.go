package logic

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	// 2. 获取或创建会话，并获取历史消息
	conversationID, historyMessages, err := l.getOrCreateConversation(in.UserId, in.ConversationId, in.Prompt)
	if err != nil {
		return err
	}

	// 3. 保存当前用户发送的消息
	userMessage := &model.Messages{
		MessageId:      tool.GenerateULID(),
		ConversationId: conversationID,
		Role:           "user",
		Content:        in.Prompt,
		ContentType:    "text",
	}
	_, err = l.svcCtx.MessageModel.Insert(l.ctx, userMessage)
	if err != nil {
		return fmt.Errorf("ChatCompletions db message Insert err:%+v, message:%+v: %w", err, userMessage, xerr.ErrDbError)
	}

	// 4. 构建对大模型 API 的请求
	llmReq := l.buildLLMRequest(in.UserId, conversationID, in.Prompt, historyMessages)
	reqBody, err := json.Marshal(llmReq)
	if err != nil {
		l.Errorf("failed to marshal llm request: %v :%w", err, xerr.ErrRequestParam)
		return err
	}

	// 5. 调用大模型 API 并处理流式响应
	return l.processLLMStream(reqBody, stream, conversationID, in.References)
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
func (l *ChatCompletionsLogic) buildLLMRequest(userID int64, convID, prompt string, history []*pb.Message) types.LLMApiRequest {
	// 只取最近的10条历史消息，history是按照时间顺序排列的，所以最新的消息在最后
	start := 0
	if len(history) > 10 {
		start = len(history) - 10
	}
	apiHistory := make([]types.LLMMessage, 0, len(history)-start)
	for _, msg := range history[start:] {
		apiHistory = append(apiHistory, types.LLMMessage{
			Role:        msg.Role,
			ContentType: msg.ContentType,
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
		},
		Stream:  true,
		ChatID:  convID,
		History: apiHistory,
	}
}

// processLLMStream 调用大模型API，处理返回的流，并推送到客户端gRPC流
func (l *ChatCompletionsLogic) processLLMStream(reqBody []byte, stream pb.LlmCenter_ChatCompletionsServer, conversationID string, references []*pb.Reference) error {
	apiURL := l.svcCtx.Config.XingChen.ApiURL
	apiKey := l.svcCtx.Config.XingChen.ApiKey
	apiSecret := l.svcCtx.Config.XingChen.ApiSecret
	authToken := fmt.Sprintf("Bearer %s:%s", apiKey, apiSecret)

	// 优化点 2：使用 http.NewRequestWithContext 传递上下文
	// l.ctx 是从 gRPC 请求中来的，如果 gRPC 连接断开，l.ctx 会被取消。
	req, err := http.NewRequestWithContext(l.ctx, "POST", apiURL, bytes.NewReader(reqBody))
	if err != nil {
		// 注意：如果 l.ctx 此时已经被取消，这里会立刻返回错误
		return fmt.Errorf("failed to create http request with context: %+v:%w", err, xerr.ErrLLMApiCancel)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authToken)
	req.Header.Set("Accept", "text/event-stream")

	// 优化点 1：使用在 ServiceContext 中初始化的可复用 HTTP 客户端
	client := l.svcCtx.LlmApiClient
	resp, err := client.Do(req)
	if err != nil {
		// 如果是因为 context 取消导致的错误，日志会记录下来，函数会优雅退出。
		return fmt.Errorf("failed to call llm api: %+v:%w", err, xerr.ErrLLMApiCancel)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("llm api returned non-200 status: %d, body: %s :%w",
			resp.StatusCode, string(bodyBytes), xerr.ErrLLMApiError)
	}

	scanner := bufio.NewScanner(resp.Body)
	var assistantReply strings.Builder
	var assistantMessageID string

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var apiResp types.LLMApiResponse
		line = strings.TrimPrefix(line, "data: ")
		if err := json.Unmarshal([]byte(line), &apiResp); err != nil {
			l.Errorf("failed to unmarshal llm stream line: %s, error: %v", line, err)
			continue
		}

		if apiResp.Code != 0 {
			// 可以选择向客户端发送一个错误事件，或者直接中断
			return fmt.Errorf("LLM API error response: code=%d, message=%s :%w",
				apiResp.Code, apiResp.Message, xerr.ErrLLMApiError)
		}

		// 检查是否是中断事件
		if apiResp.EventData != nil && apiResp.EventData.EventType == "interrupt" {
			// 将会话ID和中断事件ID存入Redis，过期时间20分钟
			redisKey := fmt.Sprintf("llm:interrupt:%s", conversationID)
			// 使用 Setex 方法设置带过期时间的键
			// 1200 秒 = 20 分钟
			err := l.svcCtx.RedisClient.Setex(redisKey, apiResp.EventData.EventID, 1200)
			if err != nil {
				// 记录错误并返回，因为redis无法设置eventid
				return fmt.Errorf("failed to set interrupt key in redis for conv %s: %v :%w",
					conversationID, err, xerr.ErrLLMInterruptEventNotSet)
			}
			l.Infof("Interrupt event received for conv %s. Storing EventID %s in Redis.", conversationID, apiResp.EventData.EventID)

			// 向客户端发送中断事件
			interruptEvent := &pb.SSEInterruptEvent{
				ConversationId: conversationID,
				// MessageId:      ... // 可以生成一个临时的ID
				ContentType: apiResp.EventData.Value.Type,
				Content:     apiResp.EventData.Value.Content,
			}
			if err := stream.Send(&pb.ChatCompletionsResponse{Event: &pb.ChatCompletionsResponse_Interrupt{Interrupt: interruptEvent}}); err != nil {
				return fmt.Errorf("failed to send interrupt event to client: %v:%w", err, xerr.ErrLLMInterruptEventNotSet)
			}
			return nil // 中断后，当前流式交互结束
		}

		// 处理正常消息流
		if len(apiResp.Choices) > 0 {
			chunk := apiResp.Choices[0].Delta.Content
			if chunk != "" {
				assistantReply.WriteString(chunk)
				// 向客户端发送消息块
				messageEvent := &pb.SSEMessageEvent{Chunk: chunk}
				if err := stream.Send(&pb.ChatCompletionsResponse{Event: &pb.ChatCompletionsResponse_Message{Message: messageEvent}}); err != nil {
					return fmt.Errorf("failed to send message chunk to client: %v:%w",
						err, xerr.ErrLLMApiCancel)
				}
			}

			// 检查是否是结束帧
			if apiResp.Choices[0].FinishReason == "stop" {
				break // 正常结束，跳出循环
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading llm stream: %v:%w", err, xerr.ErrLLMApiError)
	}

	// 6. 保存完整的助手回复,这里注意如果用户信息有图片，也需要把url存进去
	referencesData, err := json.Marshal(references)
	if err != nil {
		return fmt.Errorf("failed to marshal references: %v:%w", err, xerr.ErrRequestParam)
	}
	referencesDataStr := string(referencesData)
	assistantMessageID = tool.GenerateULID()
	assistantMessage := &model.Messages{
		MessageId:      assistantMessageID,
		ConversationId: conversationID,
		Role:           "assistant",
		Content:        assistantReply.String(),
		ContentType:    "document_outline",
		Metadata: sql.NullString{
			String: referencesDataStr, // 正确地创建 sql.NullString 实例
			Valid:  true,
		},
	}

	_, err = l.svcCtx.MessageModel.Insert(l.ctx, assistantMessage)
	if err != nil {
		l.Errorf("saveAssistantMessage failed: %v", err)
		// 即使保存失败，也应该通知前端结束，所以不在这里 return err
	}

	// 7. 向客户端发送结束事件。客户端需要在resume结束之后手动创建新的对话
	endEvent := &pb.SSEEndEvent{
		ConversationId: conversationID,
		MessageId:      assistantMessageID,
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
