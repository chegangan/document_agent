package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"document_agent/app/llmcenter/cmd/rpc/internal/svc"
	"document_agent/app/llmcenter/cmd/rpc/pb"
	"document_agent/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetHistoryDataLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

type metadataRef struct {
	Type   string `json:"type"`
	FileId string `json:"file_id"`
}

func NewGetHistoryDataLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetHistoryDataLogic {
	return &GetHistoryDataLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// RPC 方法: GetHistoryData
func (l *GetHistoryDataLogic) GetHistoryData(in *pb.GetHistoryDataRequest) (*pb.GetHistoryDataResponse, error) {
	items, err := l.svcCtx.HistoryDatasModel.FindByConversationId(l.ctx, in.ConversationId)
	if err != nil {
		return nil, fmt.Errorf("查询 historydatas 失败: %v: %w", err, xerr.ErrDbError)
	}

	var result []*pb.HistoryData

	for _, item := range items {
		var references []*pb.FileReference

		if item.Metadata.Valid {
			var refs []metadataRef
			if err := json.Unmarshal([]byte(item.Metadata.String), &refs); err == nil {
				for _, ref := range refs {
					if ref.Type == "file" {
						// 查询文件表获取 filename
						file, err := l.svcCtx.FilesModel.FindByStoredName(l.ctx, ref.FileId)
						if err == nil {
							references = append(references, &pb.FileReference{
								FileId:   ref.FileId,
								Filename: file.Filename,
								Function: "file",
							})
						} else {
							references = append(references, &pb.FileReference{
								FileId:   ref.FileId,
								Filename: "文件已过期或不存在",
								Function: "file",
							})
						}
					} else if ref.Type == "formfile" {
						file, err := l.svcCtx.FilesModel.FindByStoredName(l.ctx, ref.FileId)
						if err == nil {
							references = append(references, &pb.FileReference{
								FileId:   ref.FileId,
								Filename: file.Filename,
								Function: "formfile",
							})
						} else {
							references = append(references, &pb.FileReference{
								FileId:   ref.FileId,
								Filename: "文件已过期或不存在",
								Function: "formfile",
							})
						}
					}
				}
			}
		}

		result = append(result, &pb.HistoryData{
			MessageId:    item.MessageId,
			Documenttype: item.Documenttype,
			Information:  item.Information,
			Requests:     item.Requests,
			CreatedAt:    item.CreatedAt.Format(time.RFC3339),
			References:   references,
		})
	}

	return &pb.GetHistoryDataResponse{
		ConversationId: in.ConversationId,
		Items:          result,
	}, nil
}
