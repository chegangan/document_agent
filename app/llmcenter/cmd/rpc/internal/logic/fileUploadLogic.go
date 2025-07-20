package logic

import (
	"context"

	"document_agent/app/llmcenter/cmd/rpc/internal/svc"
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
