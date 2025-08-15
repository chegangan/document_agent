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
	// 1) 校验 type
	t := strings.ToLower(strings.TrimSpace(req.Type))
	if t != "pdf" && t != "docx" {
		return nil, fmt.Errorf("type 仅支持 pdf 或 docx")
	}
	// 2) 校验 prompt
	if strings.TrimSpace(req.Prompt) == "" {
		return nil, fmt.Errorf("prompt 不能为空")
	}

	// 3) 映射 information -> []*pb.InfoItem
	var infos []*pb.InfoItem
	if len(req.Information) > 0 {
		infos = make([]*pb.InfoItem, 0, len(req.Information))
		for _, it := range req.Information {
			// 允许空白项透传，RPC 端会再做兜底
			infos = append(infos, &pb.InfoItem{
				Type:    it.Type,
				Contant: it.Contant,
			})
		}
	}

	// 4) 调 RPC（新版字段：Prompt / Type / Information）
	rpcResp, err := l.svcCtx.LLMCenterRpc.ConvertMarkdown(l.ctx, &pb.ConvertMarkdownRequest{
		Markdown:    req.Prompt, // ✅ 新字段
		Type:        t,
		Information: infos, // ✅ 新字段
		// Markdown: 仍可不传；RPC 端已兼容 prompt 优先、fallback markdown
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
