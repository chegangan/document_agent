// filepath: pkg/interceptor/rpcserver/streamLoggerInterceptor.go
package rpcserver

import (
	"errors"

	"github.com/zeromicro/go-zero/core/logx"
	xerror "github.com/zeromicro/x/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// StreamLoggerInterceptor 是一个 gRPC 流式服务器拦截器，用于记录日志和处理错误
func StreamLoggerInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	// 调用原始的 handler
	err := handler(srv, ss)

	// 在流结束后处理错误
	if err != nil {
		var codeErr *xerror.CodeMsg
		// 检查错误链中是否包含自定义错误类型
		if errors.As(err, &codeErr) {
			// 记录完整的自定义错误信息
			logx.WithContext(ss.Context()).Errorf("【RPC-STREAM-ERR】 %+v", err)
			// 将自定义错误转换为 gRPC 的 status error
			err = status.Error(codes.Code(codeErr.Code), codeErr.Msg)
		} else {
			// 记录其他未知错误
			logx.WithContext(ss.Context()).Errorf("【RPC-STREAM-ERR】 %+v", err)
		}
	}

	return err
}
