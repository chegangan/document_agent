package logic

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"document_agent/app/llmcenter/cmd/rpc/internal/svc"
	"document_agent/app/llmcenter/cmd/rpc/pb"
	"document_agent/pkg/ctxdata"

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
	// 1. 从 gRPC Context 中获取用户ID (通常由JWT中间件注入)
	userID := ctxdata.GetUidFromCtx(l.ctx)

	// 2. 获取或创建会话，并获取历史消息
	conversationID, historyMessages, err := l.getOrCreateConversation(in.ConversationId, userID, in.Prompt)
	if err != nil {
		l.Errorf("getOrCreateConversation failed: %v", err)
		return err
	}

	// 3. 保存当前用户发送的消息
	userMessage := &pb.Message{ // TODO: 这应该是你的数据库模型结构体
		MessageId:      generateULID(),
		ConversationId: conversationID,
		Role:           "user",
		Content:        in.Prompt,
		ContentType:    "text",
		// Metadata: ... , // 如果需要，可以设置 metadata
	}
	if err := l.saveMessage(userMessage); err != nil {
		l.Errorf("saveUserMessage failed: %v", err)
		return err
	}

	// 4. 构建对大模型 API 的请求
	llmReq := l.buildLLMRequest(conversationID, userID, in.Prompt, historyMessages)
	reqBody, err := json.Marshal(llmReq)
	if err != nil {
		l.Errorf("failed to marshal llm request: %v", err)
		return err
	}

	// 5. 调用大模型 API 并处理流式响应
	return l.processLLMStream(reqBody, stream, conversationID)
}

// getUserIDFromCtx 从 context 中获取用户 ID
func (l *ChatCompletionsLogic) getUserIDFromCtx() (string, error) {
	// TODO: 在这里实现你的逻辑，例如从 JWT token 中解析用户 ID
	// var userID string
	// if jsonUid, ok := l.ctx.Value("uid").(json.Number); ok {
	// 	userID = jsonUid.String()
	// }
	// if userID == "" {
	// 	return "", errors.New("invalid user id")
	// }
	// return userID, nil
	return "mock_user_12345", nil // 这是一个示例，请替换为真实逻辑
}

// getOrCreateConversation 根据请求处理会话，如果是新会话则创建
func (l *ChatCompletionsLogic) getOrCreateConversation(convID, userID, prompt string) (string, []*pb.Message, error) { // 返回的 Message 应该是数据库模型
	if convID == "" {
		// 创建新会话
		newConvID := generateULID()
		newConversation := &pb.Conversation{ // TODO: 这应该是你的数据库模型结构体
			ConversationId: newConvID,
			UserId:         userID,
			Title:          l.generateTitle(prompt), // 使用 prompt 生成一个初始标题
		}

		_, err := l.svcCtx.ConversationModel.Insert(l.ctx, newConversation)
		if err != nil {
			return "", nil, err
		}

		return newConvID, []*pb.Message{}, nil
	}

	// TODO: 验证现有会话是否存在，并属于当前用户
	// conversation, err := l.svcCtx.ConversationModel.FindOne(l.ctx, convID)
	// if err != nil {
	// 	return "", nil, err // or custom error types.ErrConversationNotFound
	// }
	// if conversation.UserId != userID {
	// 	return "", nil, errors.New("access denied to this conversation")
	// }

	// TODO: 从数据库获取该会话的历史消息
	// history, err := l.svcCtx.MessageModel.FindAllByConversationID(l.ctx, convID)
	// if err != nil {
	//	return "", nil, err
	// }
	// return convID, history, nil

	return convID, []*pb.Message{}, nil // 这是一个示例，请替换为真实逻辑
}

// saveMessage 将消息保存到数据库
func (l *ChatCompletionsLogic) saveMessage(msg *pb.Message) error {
	// TODO: 调用数据库模型来插入新的消息记录
	// return l.svcCtx.MessageModel.Insert(l.ctx, msg)
	l.Infof("Saving message: %+v", msg) // 打印日志代替真实存储
	return nil
}

// buildLLMRequest 构建发送给星火大模型 API 的请求体
func (l *ChatCompletionsLogic) buildLLMRequest(convID, userID, prompt string, history []*pb.Message) LLMApiRequest {
	apiHistory := make([]LLMMessage, 0, len(history))
	for _, msg := range history {
		apiHistory = append(apiHistory, LLMMessage{
			Role:        msg.Role,
			ContentType: msg.ContentType,
			Content:     msg.Content,
		})
	}

	// TODO: 从你的 svcCtx.Config 中加载这些配置
	// flowID := l.svcCtx.Config.Xingchen.FlowID
	flowID := "7265177322515169282" // 示例值

	return LLMApiRequest{
		FlowID: flowID,
		UID:    userID,
		Parameters: LLMParameters{
			AgentUserInput: prompt,
		},
		Ext: LLMExt{
			BotID:  "workflow",
			Caller: "workflow",
		},
		Stream:  true,
		ChatID:  convID,
		History: apiHistory,
	}
}

// processLLMStream 调用大模型API，处理返回的流，并推送到客户端gRPC流
func (l *ChatCompletionsLogic) processLLMStream(reqBody []byte, stream pb.LlmCenter_ChatCompletionsServer, conversationID string) error {
	// TODO: 从你的 svcCtx.Config 中加载 API URL 和认证信息
	// apiURL := l.svcCtx.Config.Xingchen.ApiURL
	// apiKey := l.svcCtx.Config.Xingchen.ApiKey
	// apiSecret := l.svcCtx.Config.Xingchen.ApiSecret
	apiURL := "https://xingchen-api.xf-yun.com/workflow/v1/chat/completions"
	authToken := "Bearer YOUR_API_KEY:YOUR_API_SECRET" // TODO: 替换为真实的认证 Token

	req, err := http.NewRequest("POST", apiURL, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create http request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authToken)
	req.Header.Set("Accept", "text/event-stream") // SSE 要求

	// TODO: 使用你项目中配置的 HTTP 客户端
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call llm api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("llm api returned non-200 status: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	scanner := bufio.NewScanner(resp.Body)
	var assistantReply strings.Builder
	var assistantMessageID string

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var apiResp LLMApiResponse
		if err := json.Unmarshal([]byte(line), &apiResp); err != nil {
			l.Warnf("failed to unmarshal llm stream line: %s, error: %v", line, err)
			continue
		}

		if apiResp.Code != 0 {
			l.Errorf("LLM API error response: code=%d, message=%s", apiResp.Code, apiResp.Message)
			// 可以选择向客户端发送一个错误事件，或者直接中断
			return fmt.Errorf("llm api error: %s", apiResp.Message)
		}

		// 检查是否是中断事件
		if apiResp.EventData != nil && apiResp.EventData.EventType == "interrupt" {
			// 将会话ID存入Redis，过期时间30分钟
			// TODO: 使用你的 Redis 客户端
			// redisKey := fmt.Sprintf("llm:interrupt:%s", conversationID)
			// err := l.svcCtx.RedisClient.Setex(redisKey, apiResp.EventData.EventID, 1800) // 30 * 60 seconds
			// if err != nil {
			// 	l.Errorf("failed to set interrupt key in redis for conv %s: %v", conversationID, err)
			// 	return err // 如果关键步骤失败，则返回错误
			// }
			l.Infof("Interrupt event received for conv %s. Storing in Redis.", conversationID)

			// 向客户端发送中断事件
			interruptEvent := &pb.SSEInterruptEvent{
				ConversationId: conversationID,
				// MessageId:      ... // 可以生成一个临时的ID
				ContentType: apiResp.EventData.Value.Type,
				Content:     apiResp.EventData.Value.Content,
			}
			if err := stream.Send(&pb.ChatCompletionsResponse{Event: &pb.ChatCompletionsResponse_Interrupt{Interrupt: interruptEvent}}); err != nil {
				l.Errorf("failed to send interrupt event to client: %v", err)
				return err
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
					l.Errorf("failed to send message chunk to client: %v", err)
					return err
				}
			}

			// 检查是否是结束帧
			if apiResp.Choices[0].FinishReason == "stop" {
				break // 正常结束，跳出循环
			}
		}
	}

	if err := scanner.Err(); err != nil {
		l.Errorf("error reading llm stream: %v", err)
		return err
	}

	// 6. 保存完整的助手回复
	assistantMessageID = generateULID()
	assistantMessage := &pb.Message{ // TODO: 数据库模型
		MessageId:      assistantMessageID,
		ConversationId: conversationID,
		Role:           "assistant",
		Content:        assistantReply.String(),
		ContentType:    "text",
	}
	if err := l.saveMessage(assistantMessage); err != nil {
		l.Errorf("saveAssistantMessage failed: %v", err)
		// 即使保存失败，也应该通知前端结束，所以不在这里 return err
	}

	// 7. 向客户端发送结束事件
	endEvent := &pb.SSEEndEvent{
		ConversationId: conversationID,
		MessageId:      assistantMessageID,
	}
	if err := stream.Send(&pb.ChatCompletionsResponse{Event: &pb.ChatCompletionsResponse_End{End: endEvent}}); err != nil {
		l.Errorf("failed to send end event to client: %v", err)
		return err
	}

	return nil
}

// generateULID 生成一个 ULID 作为唯一标识符
func generateULID() string {
	return ulid.Make().String()
}

// generateTitle 根据用户的第一条消息生成一个简单的标题
func (l *ChatCompletionsLogic) generateTitle(prompt string) string {
	// 将 prompt 转换为 rune 数组以正确处理多字节字符（如中文）
	runes := []rune(prompt)
	maxLength := 20 // 标题最大长度
	if len(runes) > maxLength {
		return string(runes[:maxLength]) + "..."
	}
	return string(runes)
}
