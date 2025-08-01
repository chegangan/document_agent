package logic

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"document_agent/app/llmcenter/cmd/rpc/internal/llm"
	"document_agent/app/llmcenter/cmd/rpc/internal/svc"
	"document_agent/app/llmcenter/cmd/rpc/pb"
	"document_agent/app/llmcenter/cmd/rpc/types"
	"document_agent/app/llmcenter/model"
	"document_agent/pkg/fileprocessor"
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
	// 1. 获取或创建会话
	conversationID, historyMessages, err := l.getOrCreateConversation(in.UserId, in.ConversationId, in.Prompt)
	if err != nil {
		return err
	}

	// 2. 处理文件引用，并构建最终的 prompt
	finalPrompt, imgURL, err := l.processReferences(in.Prompt, in.References)
	if err != nil {
		// 在处理文件引用失败时，可以只记录日志，然后继续使用原始 prompt
		l.Errorf("processReferences failed: %v. proceeding with original prompt.", err)
		finalPrompt = in.Prompt
	}

	// 3. 保存用户消息
	if err := l.saveUserMessage(conversationID, finalPrompt); err != nil {
		return err
	}

	// 4. 构建 LLM 请求
	llmReq := l.buildLLMRequest(in.UserId, conversationID, finalPrompt, historyMessages, imgURL)
	reqBody, err := json.Marshal(llmReq)
	if err != nil {
		l.Errorf("failed to marshal llm request: %v :%w", err, xerr.ErrRequestParam)
		return err
	}

	// 5. 调用大模型 API 并处理流式响应
	xingchenClient := llm.NewXingChenClient(l.ctx, l.svcCtx)
	assistantReply, err := xingchenClient.StreamChat(reqBody, stream, conversationID)
	if err != nil {
		return err // 错误已在 StreamChat 中包装
	}

	// 如果没有回复内容（例如，在中断事件后），则不保存消息直接发送结束事件
	if assistantReply == "" {
		return l.sendEndEvent(stream, conversationID, "")
	}

	// 6. 保存助手回复
	assistantMessageID, err := l.saveAssistantMessage(conversationID, assistantReply, in.References)
	if err != nil {
		// 保存失败也应通知前端结束，所以只记录日志不返回错误
		l.Errorf("saveAssistantMessage failed: %v", err)
	}

	// 7. 发送结束事件
	return l.sendEndEvent(stream, conversationID, assistantMessageID)
}

// processReferences 处理文件引用，增强 prompt
func (l *ChatCompletionsLogic) processReferences(prompt string, references []*pb.Reference) (string, string, error) {
	var imgURL string
	var fileContents []string
	reImg := regexp.MustCompile(`(?i)\.(jpg|jpeg|png)$`)
	reDoc := regexp.MustCompile(`(?i)\.(txt|md|csv|docx|pdf)$`)
	xingchenClient := llm.NewXingChenClient(l.ctx, l.svcCtx)

	for _, ref := range references {
		if ref.Type != "file" {
			continue
		}

		localPath := filepath.Join(l.svcCtx.Config.Upload.BaseDir, ref.FileId)
		ext := strings.ToLower(filepath.Ext(ref.FileId))

		if reImg.MatchString(ref.FileId) {
			url, err := xingchenClient.UploadImage(localPath)
			if err != nil {
				l.Errorf("图片上传失败：file_id=%s err=%v", ref.FileId, err)
				continue
			}
			imgURL = url // 目前只支持一张图片
		} else if reDoc.MatchString(ref.FileId) {
			var content string
			var err error
			switch ext {
			case ".txt", ".md", ".csv":
				content, err = fileprocessor.ReadTextFile(localPath)
			case ".docx":
				content, err = fileprocessor.ReadDocxFile(localPath)
			case ".pdf":
				content, err = fileprocessor.ReadPdfFile(localPath)
			}

			if err != nil {
				l.Errorf("读取文件失败：file_id=%s err=%v", ref.FileId, err)
				continue
			}

			if len(content) > 5000 {
				content = content[:5000] + "...(已截断)"
			}
			fileContents = append(fileContents, fmt.Sprintf("一份%s文件内容如下：\n%s", ext, content))
		}
	}

	if len(fileContents) > 0 {
		prompt += "\n\n用户提供了以下文件内容作为参考：\n" + strings.Join(fileContents, "\n\n")
	}

	return prompt, imgURL, nil
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
