package logic

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"document_agent/app/llmcenter/cmd/rpc/internal/svc"
	"document_agent/app/llmcenter/cmd/rpc/pb"
	"document_agent/pkg/tool"

	"github.com/zeromicro/go-zero/core/logx"
)

type ConvertMarkdownLinkLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewConvertMarkdownLinkLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ConvertMarkdownLinkLogic {
	return &ConvertMarkdownLinkLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ConvertMarkdownLinkLogic) ConvertMarkdownLink(in *pb.ConvertMarkdownLinkRequest) (*pb.ConvertMarkdownLinkResponse, error) {
	t := strings.ToLower(strings.TrimSpace(in.Type))
	if t != "pdf" && t != "docx" {
		return nil, fmt.Errorf("type 仅支持 pdf 或 docx")
	}

	// 1) 统一换行 + 预处理（\\n -> \n；转义 1.2.3. -> 1\.2\.3\.，且避开代码块/行内代码）
	md := strings.ReplaceAll(in.Markdown, "\r\n", "\n")
	md = preprocessMarkdown(md)

	// 2) 通过 Pandoc 生成目标格式
	data, err := runPandoc(l.ctx, md, t, l.svcCtx.Config.Font.Path)
	if err != nil {
		return nil, fmt.Errorf("渲染失败: %w", err)
	}

	// 3) 落盘到配置的上传目录
	absDir := l.svcCtx.Config.Upload.BaseDir
	if err := os.MkdirAll(absDir, 0o755); err != nil {
		return nil, fmt.Errorf("创建目录失败: %w", err)
	}
	ext := map[string]string{"pdf": ".pdf", "docx": ".docx"}[t]
	name := tool.GenerateULID() + ext
	absPath := filepath.Join(absDir, name)
	if err := os.WriteFile(absPath, data, 0o644); err != nil {
		return nil, fmt.Errorf("写文件失败: %w", err)
	}

	// 4) 生成签名直链
	expire := l.svcCtx.Config.Download.ExpireSeconds
	if expire <= 0 {
		expire = 600
	}
	exp := time.Now().Add(time.Duration(expire) * time.Second).Unix()
	toSign := fmt.Sprintf("%s|%d", name, exp)
	sig := signHMAC(toSign, l.svcCtx.Config.Download.SignKey)

	base := strings.TrimRight(l.svcCtx.Config.Download.BaseURL, "/")
	downloadURL := fmt.Sprintf("%s?path=%s&exp=%d&sig=%s",
		base, url.QueryEscape(name), exp, sig)

	filename := map[string]string{"pdf": "export.pdf", "docx": "export.docx"}[t]
	contentType := map[string]string{
		"pdf":  "application/pdf",
		"docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	}[t]

	return &pb.ConvertMarkdownLinkResponse{
		Filename:    filename,
		ContentType: contentType,
		Path:        name,        // 相对 BaseDir 的文件名
		Url:         downloadURL, // 免 Header 直链
	}, nil
}

func signHMAC(data, key string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(data))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
