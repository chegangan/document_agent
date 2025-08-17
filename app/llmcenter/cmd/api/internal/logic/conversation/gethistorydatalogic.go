package conversation

import (
	"context"

	"document_agent/app/llmcenter/cmd/api/internal/svc"
	"document_agent/app/llmcenter/cmd/api/internal/types"
	rpcpb "document_agent/app/llmcenter/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetHistoryDataLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 根据会话ID获取该会话的历史数据（historydatas 表）
func NewGetHistoryDataLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetHistoryDataLogic {
	return &GetHistoryDataLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetHistoryDataLogic) GetHistoryData(req *types.GetHistoryDataRequest) (*types.GetHistoryDataResponse, error) {
	rpcResp, err := l.svcCtx.LLMCenterRpc.GetHistoryData(l.ctx, &rpcpb.GetHistoryDataRequest{
		ConversationId: req.ConversationID,
	})
	if err != nil {
		l.Logger.Errorf("调用 GetHistoryData RPC 失败: %v", err)
		return nil, err
	}

	var result []types.HistoryData
	for _, item := range rpcResp.Items {
		// 构造 references
		var refs []types.FileReference
		for _, r := range item.References {
			refs = append(refs, types.FileReference{
				FileID:   r.FileId,
				Filename: r.Filename,
				Function: r.Function,
			})
		}

		result = append(result, types.HistoryData{
			ID:           item.MessageId,
			Documenttype: item.Documenttype,
			Information:  item.Information,
			Requests:     item.Requests,
			CreatedAt:    item.CreatedAt,
			References:   refs,
		})
	}

	return &types.GetHistoryDataResponse{
		ConversationID: rpcResp.ConversationId,
		Items:          result,
	}, nil
}
