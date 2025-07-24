package file

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"

	"document_agent/app/llmcenter/cmd/api/internal/svc"
	"document_agent/app/llmcenter/cmd/api/internal/types"
	"document_agent/app/llmcenter/cmd/rpc/pb"

	"document_agent/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type FileUploadLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 上传文件, 用于后续对话
func NewFileUploadLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FileUploadLogic {
	return &FileUploadLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *FileUploadLogic) FileUpload(form *multipart.Form) (*types.FileUploadResponse, error) {
	files := form.File["file"]
	if len(files) == 0 {
		return nil, fmt.Errorf("未找到文件: %v", xerr.ErrFileNotFound) // 你可以自定义这个错误
	}

	fileHeader := files[0]
	fileName := fileHeader.Filename
	file, err := fileHeader.Open()
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// 调用 RPC
	stream, err := l.svcCtx.LLMCenterRpc.FileUpload(l.ctx)
	if err != nil {
		fmt.Println("apierror")
		return nil, err
	}
	fmt.Println("api")

	// 发送 FileInfo
	err = stream.Send(&pb.FileUploadRequest{
		Data: &pb.FileUploadRequest_Info{
			Info: &pb.FileInfo{
				FileName: fileName,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	// 分片发送数据块
	buf := make([]byte, 32*1024)
	for {
		n, err := file.Read(buf)
		if err != nil && err != io.EOF {
			return nil, err
		}
		if n == 0 {
			break
		}

		err = stream.Send(&pb.FileUploadRequest{
			Data: &pb.FileUploadRequest_Chunk{
				Chunk: buf[:n],
			},
		})
		if err != nil {
			return nil, err
		}
	}

	// 获取响应
	reply, err := stream.CloseAndRecv()
	if err != nil {
		return nil, err
	}

	return &types.FileUploadResponse{
		FileID:   reply.FileId,
		FileName: reply.FileName,
		URL:      reply.Url,
		Message:  reply.Message,
	}, nil
}
