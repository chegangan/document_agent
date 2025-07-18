package logic

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"document_agent/app/llmcenter/cmd/rpc/internal/svc"
	"document_agent/app/llmcenter/cmd/rpc/llmcenter"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
)

const (
	// 定义一个安全的文件上传目录
	uploadDir = "./uploads"
)

type FileUploadLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewFileUploadLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FileUploadLogic {
	return &FileUploadLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// FileUpload 处理客户端流式文件上传
func (l *FileUploadLogic) FileUpload(stream llmcenter.LlmCenter_FileUploadServer) error {
	// 1. 接收第一个请求，它必须包含文件的元信息 (FileInfo)
	req, err := stream.Recv()
	if err != nil {
		l.Logger.Errorf("Failed to receive first stream message: %v", err)
		return err
	}

	// 从请求中获取文件元信息
	fileInfo := req.GetInfo()
	if fileInfo == nil {
		return errors.New("expected first message to be FileInfo, but it was not")
	}
	l.Logger.Infof("Receiving file upload for: %s", fileInfo.FileName)

	// 2. 准备在服务端创建文件
	// 生成一个唯一的文件ID，防止文件名冲突
	fileID := uuid.New().String()
	// 拼接一个安全的文件名，例如使用 fileID 和原始文件的扩展名
	newFileName := fmt.Sprintf("%s%s", fileID, filepath.Ext(fileInfo.FileName))

	// 确保上传目录存在
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		l.Logger.Errorf("Failed to create upload directory: %v", err)
		return err
	}

	// 创建文件路径并打开文件准备写入
	filePath := filepath.Join(uploadDir, newFileName)
	file, err := os.Create(filePath)
	if err != nil {
		l.Logger.Errorf("Failed to create file on server: %v", err)
		return err
	}
	defer file.Close() // 确保函数结束时关闭文件

	// 3. 循环接收文件数据块 (chunk) 并写入文件
	for {
		// 持续从流中接收消息
		req, err := stream.Recv()

		// 如果错误是 io.EOF，表示客户端已经发送完所有数据，这是正常结束的信号
		if err == io.EOF {
			l.Logger.Infof("File upload finished for %s, saved to %s", fileInfo.FileName, filePath)
			break // 跳出循环
		}
		// 如果是其他错误，记录日志并返回
		if err != nil {
			l.Logger.Errorf("Error while receiving file chunk: %v", err)
			// 如果接收出错，最好删除已创建的不完整文件
			os.Remove(filePath)
			return err
		}

		// 获取数据块
		chunk := req.GetChunk()
		if chunk == nil {
			// 正常情况下，除了第一个消息，其他都应该是 chunk
			// 如果不是，则中断并清理
			os.Remove(filePath)
			return errors.New("expected subsequent messages to be file chunks")
		}

		// 将数据块写入文件
		if _, err := file.Write(chunk); err != nil {
			l.Logger.Errorf("Failed to write chunk to file: %v", err)
			os.Remove(filePath)
			return err
		}
	}

	// 4. 文件接收完毕，向客户端发送最终响应
	// 这里的 URL 可以是您后续提供的文件服务 URL
	fileURL := fmt.Sprintf("/files/%s", newFileName)

	// 调用 stream.SendAndClose() 发送响应并关闭流
	err = stream.SendAndClose(&llmcenter.FileUploadResponse{
		FileId:   fileID,
		FileName: fileInfo.FileName,
		Url:      fileURL,
		Message:  "文件上传成功",
	})
	if err != nil {
		l.Logger.Errorf("Failed to send response and close stream: %v", err)
		return err
	}

	// 在这里，您可以添加将文件信息（fileID, filePath, etc.）存入数据库的逻辑
	// l.svcCtx.YourModel.Create(...)

	return nil
}

// ### 核心逻辑步骤分解：

// 1.  **接收元信息**：
//     * 第一次调用 `stream.Recv()` 来获取客户端发送的第一个消息。
//     * 根据我们设计的协议，这个消息必须是 `FileInfo` 类型。我们通过 `req.GetInfo()` 来获取它。如果不是，就返回错误。

// 2.  **创建本地文件**：
//     * 为了避免文件名冲突和安全问题，我们使用 `uuid` 生成一个唯一的文件 ID。
//     * 使用这个 ID 和原始文件的扩展名，构造一个新的、安全的文件名。
//     * 确保服务端的上传目录存在 (`os.MkdirAll`)。
//     * 创建并打开这个文件 (`os.Create`)，准备接收数据。`defer file.Close()` 是一个好习惯，确保文件句柄在任何情况下都会被释放。

// 3.  **循环接收数据块**：
//     * 在一个 `for` 循环中，持续调用 `stream.Recv()`。
//     * **处理流结束**：当 `stream.Recv()` 返回 `io.EOF` 错误时，这并不是一个真正的错误。它是一个信号，表示客户端已经成功发送完所有数据并关闭了它的发送流。此时我们应该跳出循环。
//     * **处理数据块**：对于每个成功接收到的消息，我们通过 `req.GetChunk()` 获取字节数据，并使用 `file.Write()` 将其写入我们之前创建的文件中。
//     * **错误处理**：如果在接收过程中发生除 `io.EOF` 之外的任何错误，或者接收到的消息不是预期的 `chunk` 类型，都应该中断上传，并删除在服务端创建的不完整文件 (`os.Remove`)，防止磁盘被垃圾文件占满。

// 4.  **发送最终响应**：
//     * 当循环正常结束后，说明文件已完整接收。
//     * 调用 `stream.SendAndClose()`，这个方法会做两件事：
//         1.  将 `FileUploadResponse` 发送给客户端。
//         2.  正式关闭服务端的流。
//     * 此后，您可以执行将文件元信息持久化到数据库等收尾
