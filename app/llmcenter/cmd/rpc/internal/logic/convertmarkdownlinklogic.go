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
	md := strings.ReplaceAll(in.Markdown, "\r\n", "\n")
	doc := parseMarkdown(md)

	var (
		data        []byte
		contentType string
		filename    string
		err         error
	)
	if t == "pdf" {
		data, err = renderPDF(doc, l.svcCtx.Config.Font.Path)
		contentType, filename = "application/pdf", "export.pdf"
	} else {
		data, err = renderDOCX(doc)
		contentType, filename = "application/vnd.openxmlformats-officedocument.wordprocessingml.document", "export.docx"
	}
	if err != nil {
		return nil, fmt.Errorf("渲染失败: %w", err)
	}

	// ✅ 落盘到 data/static 根目录
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

	// ✅ 生成签名直链（免 Header）
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

	return &pb.ConvertMarkdownLinkResponse{
		Filename:    filename,
		ContentType: contentType,
		Path:        name,        // 相对 data/static 的文件名
		Url:         downloadURL, // 免 Header 直链
	}, nil
}

func signHMAC(data, key string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(data))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
