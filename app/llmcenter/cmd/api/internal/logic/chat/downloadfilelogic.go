package chat

import (
	"context"
	"fmt"
	"strings"

	"document_agent/app/llmcenter/cmd/api/internal/svc"
	"document_agent/app/llmcenter/cmd/api/internal/types"
	"document_agent/app/llmcenter/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type DownloadFileLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 将Markdown转为相应格式并下载
func NewDownloadFileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DownloadFileLogic {
	return &DownloadFileLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

type DownloadFileResp struct {
	Filename    string
	ContentType string
	Data        []byte
}

func (l *DownloadFileLogic) DownloadFile(req *types.DownloadFileRequest) (*DownloadFileResp, error) {
	t := strings.ToLower(strings.TrimSpace(req.Type))
	if t != "pdf" && t != "docx" {
		return nil, fmt.Errorf("type 仅支持 pdf 或 docx")
	}
	if strings.TrimSpace(req.Prompt) == "" {
		return nil, fmt.Errorf("prompt 不能为空")
	}

	// 调 RPC
	rpcResp, err := l.svcCtx.LLMCenterRpc.ConvertMarkdown(l.ctx, &pb.ConvertMarkdownRequest{
		Markdown: req.Prompt,
		Type:     t,
	})
	if err != nil {
		return nil, err
	}
	return &DownloadFileResp{
		Filename:    rpcResp.Filename,
		ContentType: rpcResp.ContentType,
		Data:        rpcResp.Data,
	}, nil
}
