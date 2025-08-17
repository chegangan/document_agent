package logic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"document_agent/app/llmcenter/cmd/rpc/internal/llm"
	"document_agent/app/llmcenter/cmd/rpc/internal/svc"
	"document_agent/app/llmcenter/cmd/rpc/pb"
	"document_agent/app/llmcenter/cmd/rpc/types"
	"document_agent/app/llmcenter/model"
	"document_agent/pkg/tool"
	"document_agent/pkg/xerr"

	"document_agent/pkg/fileprocessor"
	"github.com/zeromicro/go-zero/core/logx"
)

type ChatResumeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewChatResumeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ChatResumeLogic {
	return &ChatResumeLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// ChatResume 现在改为“像正常对话一样直接继续生成”，不再走 Resume/事件ID 机制。
func (l *ChatResumeLogic) ChatResume(in *pb.ChatResumeRequest, stream pb.LlmCenter_ChatResumeServer) error {
	// 1) 校验会话归属
	if err := l.validateConversation(in.UserId, in.ConversationId); err != nil {
		return err
	}

	// 2) 取该会话最近的历史消息（与 ChatCompletions 一致）
	history, err := l.getRecentHistory(in.ConversationId)
	if err != nil {
		return err
	}

	// 3) 构建大模型请求体（沿用与 ChatCompletions 一致的 StreamChat 请求结构）
	llmReq := l.buildLLMRequest(in.UserId, in.ConversationId, in.Documenttype, in.Content, history, in.References)

	reqBody, err := json.Marshal(llmReq)
	if err != nil {
		l.Errorf("failed to marshal llm request: %v :%w", err, xerr.ErrRequestParam)
		return err
	}

	// 4) 直接调用大模型 StreamChat（不再使用 Resume API）
	xingchenClient := llm.NewXingChenClient(l.ctx, l.svcCtx)
	assistantReply, err := xingchenClient.StreamChatForResume(reqBody, stream, in.ConversationId)

	if err != nil {
		return err // 内部已做错误包装
	}

	// 若模型没有返回任何正文，也照样结束（但不落库）
	if assistantReply == "" {
		return l.sendEndEvent(stream, in.ConversationId, "")
	}

	// 5) 将最终生成的完整内容保存到 documents（复用你原有的保存逻辑）
	assistantMessageID, err := l.saveFinalDocument(in.ConversationId, assistantReply)
	if err != nil {
		// 记录错误，但不中断结束事件
		l.Errorf("saveFinalDocument failed: %v", err)
	}

	// 6) 发送结束事件（带 message_id）
	return l.sendEndEvent(stream, in.ConversationId, assistantMessageID)
}

// validateConversation 验证会话是否存在且属于该用户
func (l *ChatResumeLogic) validateConversation(userID int64, convID string) error {
	conversation, err := l.svcCtx.ConversationModel.FindOne(l.ctx, convID)
	if err != nil {
		if err == model.ErrNotFound {
			return fmt.Errorf("validateConversation db conversation FindOne not found err:%+v, conversationId:%s: %w", err, convID, xerr.ErrConversationNotFound)
		}
		return fmt.Errorf("validateConversation db conversation FindOne err:%+v, conversationId:%s: %w", err, convID, xerr.ErrDbError)
	}
	if conversation.UserId != userID {
		return fmt.Errorf("validateConversation user %d is not allowed to access conversation %s: %w", userID, convID, xerr.ErrConversationAccessDenied)
	}
	return nil
}

// getRecentHistory 拉取会话历史并仅保留最近 10 条（与 ChatCompletions 的做法一致）
func (l *ChatResumeLogic) getRecentHistory(convID string) ([]*pb.Message, error) {
	getConversationDetailLogic := NewGetConversationDetailLogic(l.ctx, l.svcCtx)
	resp, err := getConversationDetailLogic.GetConversationDetail(&pb.GetConversationDetailRequest{
		ConversationId: convID,
	})
	if err != nil {
		return nil, fmt.Errorf("getRecentHistory db message FindAllByConversationID err:%+v, conversationId:%s: %w", err, convID, xerr.ErrMessageNotFound)
	}
	history := resp.GetHistory()

	// 只取最近 10 条
	if len(history) > 10 {
		history = history[len(history)-10:]
	}
	return history, nil
}

// buildLLMRequest 与 ChatCompletions 的组装逻辑保持一致（不含图片）
func (l *ChatResumeLogic) buildLLMRequest(userID int64, convID string, documentType string, prompt string, history []*pb.Message, references []*pb.Reference) types.LLMApiRequest {
	// 先拼接 documenttype
	basePrompt := fmt.Sprintf("请根据用户给的内容清单中的内容生成一篇%s", documentType)

	// 处理文件引用（图片OCR + 文本）
	enrichedPrompt, err := l.enrichPromptWithReferences(basePrompt, references)
	if err != nil {
		l.Errorf("enrichPromptWithReferences failed: %v", err)
		enrichedPrompt = basePrompt
	}

	enrichedPrompt = fmt.Sprintf("%s\n\n清单内容如下：%s", enrichedPrompt, strings.TrimSpace(prompt))

	// 加上开头标识码（FlagCode2）
	flag := l.svcCtx.Config.XingChen.FlagCode2
	if flag != "" {
		enrichedPrompt = flag + enrichedPrompt
	}

	// 只保留最近 10 条
	if len(history) > 10 {
		history = history[len(history)-10:]
	}

	apiHistory := make([]types.LLMMessage, 0, len(history))
	for _, msg := range history {
		apiHistory = append(apiHistory, types.LLMMessage{
			Role:        msg.Role,
			ContentType: "text",
			Content:     msg.Content,
		})
	}

	return types.LLMApiRequest{
		FlowID: l.svcCtx.Config.XingChen.FlowID,
		UID:    fmt.Sprintf("%d", userID),
		Parameters: types.LLMParameters{
			AgentUserInput: enrichedPrompt,
			Img:            "", // OCR 文本已并入 prompt
		},
		Stream:  true,
		ChatID:  convID,
		History: apiHistory,
	}
}

// saveFinalDocument 保存最终生成的完整文章
func (l *ChatResumeLogic) saveFinalDocument(conversationID, content string) (string, error) {
	if content == "" {
		return "", nil
	}
	documentID := tool.GenerateULID()
	if err := l.svcCtx.DocumentsModel.InsertDocument(l.ctx, documentID, conversationID, content); err != nil {
		return "", fmt.Errorf("saveFinalDocument db Insert to documents err:%+v: %w", err, xerr.ErrDbError)
	}
	return documentID, nil
}

// sendEndEvent 向客户端发送结束事件
func (l *ChatResumeLogic) sendEndEvent(stream pb.LlmCenter_ChatResumeServer, conversationID, messageID string) error {
	endEvent := &pb.SSEEndEvent{
		ConversationId: conversationID,
		MessageId:      messageID,
	}
	if err := stream.Send(&pb.ChatResumeResponse{Event: &pb.ChatResumeResponse_End{End: endEvent}}); err != nil {
		return fmt.Errorf("failed to send end event to client: %v:%w", err, xerr.ErrLLMApiCancel)
	}
	return nil
}

// enrichPromptWithReferences 将文件引用（图片OCR + 文本文件读取）拼进提示词
func (l *ChatResumeLogic) enrichPromptWithReferences(basePrompt string, references []*pb.Reference) (string, error) {
	if len(references) == 0 {
		return basePrompt, nil
	}

	var fileContents []string
	reImg := regexp.MustCompile(`(?i)\.(jpg|jpeg|png)$`)
	reDoc := regexp.MustCompile(`(?i)\.(txt|md|csv|docx|pdf|xlsx|pptx)$`)

	for _, ref := range references {
		if ref.Type != "file" {
			continue
		}
		localPath := filepath.Join(l.svcCtx.Config.Upload.BaseDir, ref.FileId)
		ext := strings.ToLower(filepath.Ext(ref.FileId))

		switch {
		case reImg.MatchString(ref.FileId):
			text, err := l.ocrImage(localPath, "chi_sim+eng")
			if err != nil || strings.TrimSpace(text) == "" {
				l.Errorf("ChatResume OCR失败 file_id=%s err=%v", ref.FileId, err)
				continue
			}
			// 截断，避免提示词过长
			runes := []rune(text)
			if len(runes) > 3000 {
				text = string(runes[:3000]) + "...(OCR内容已截断)"
			}
			fileContents = append(fileContents, fmt.Sprintf("一张图片（%s）识别到的文字：\n%s", ext, text))

		case reDoc.MatchString(ref.FileId):
			var (
				content string
				err     error
			)
			switch ext {
			case ".txt", ".md", ".csv":
				content, err = fileprocessor.ReadTextFile(localPath)
			case ".docx":
				content, err = fileprocessor.ReadDocxFile(localPath)
			case ".pdf":
				content, err = fileprocessor.ReadPdfFile(localPath)
			case ".xlsx":
				content, err = fileprocessor.ReadXlsxFile(localPath)
			case ".pptx":
				content, err = fileprocessor.ReadPptxFile(localPath)
			}
			if err != nil {
				l.Errorf("ChatResume 读取文件失败 file_id=%s err=%v", ref.FileId, err)
				continue
			}
			if len(content) > 5000 {
				content = content[:5000] + "...(已截断)"
			}
			fileContents = append(fileContents, fmt.Sprintf("一份%s文件内容如下：\n%s", ext, content))
		}
	}

	if len(fileContents) > 0 {
		basePrompt += "\n\n参考用户给的实例公文的风格和格式，实例公文的内容如下：\n" + strings.Join(fileContents, "\n\n")
	}
	return basePrompt, nil
}

// ocrImage 使用 Tesseract 对图片进行 OCR，返回识别到的文本
func (l *ChatResumeLogic) ocrImage(imagePath, lang string) (string, error) {
	if lang == "" {
		lang = "chi_sim+eng"
	}
	cmd := exec.CommandContext(l.ctx, "tesseract", imagePath, "stdout", "-l", lang, "--psm", "3")
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		l.Errorf("tesseract ocr failed, stderr: %s", stderr.String())
		return "", err
	}
	text := strings.TrimSpace(out.String())
	// 简单清洗：压缩多空白
	text = regexp.MustCompile(`[ \t\r\f]+`).ReplaceAllString(text, " ")
	return text, nil
}
