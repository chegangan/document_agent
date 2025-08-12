package agent

import (
	"context"

	"document_agent/app/llmcenter/cmd/api/internal/svc"
	"document_agent/app/llmcenter/cmd/api/internal/types"
	"document_agent/app/llmcenter/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type ConvertMarkdownLinkLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Markdown 转文件并返回下载链接
func NewConvertMarkdownLinkLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ConvertMarkdownLinkLogic {
	return &ConvertMarkdownLinkLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ConvertMarkdownLinkLogic) ConvertMarkdownLink(req *types.ConvertMarkdownLinkRequest) (*types.ConvertMarkdownLinkResponse, error) {
	resp, err := l.svcCtx.LLMCenterRpc.ConvertMarkdownLink(l.ctx, &pb.ConvertMarkdownLinkRequest{
		Type: req.Type, Markdown: req.Markdown,
	})
	if err != nil {
		return nil, err
	}
	return &types.ConvertMarkdownLinkResponse{
		Filename: resp.Filename, ContentType: resp.ContentType, Path: resp.Path, Url: resp.Url,
	}, nil
}
