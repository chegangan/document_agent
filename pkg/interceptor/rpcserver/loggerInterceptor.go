package rpcserver

import (
	"context"
	"errors"
	"github.com/zeromicro/go-zero/core/logx"
	xerror "github.com/zeromicro/x/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func LoggerInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	resp, err = handler(ctx, req)
	if err != nil {
		var codeErr *xerror.CodeMsg
		// 使用 errors.As 检查错误链中是否包含 *xerror.CodeMsg 类型的错误
		if errors.As(err, &codeErr) {
			// 如果是自定义错误，记录完整的原始错误信息（error包含堆栈）
			logx.WithContext(ctx).Errorf("【RPC-SRV-ERR】 %+v", err)

			// 将自定义错误转换为 gRPC 的 status error
			err = status.Error(codes.Code(codeErr.Code), codeErr.Msg)
		} else {
			// 如果不是自定义错误，只记录错误日志
			logx.WithContext(ctx).Errorf("【RPC-SRV-ERR】 %+v", err)
		}
	}

	return resp, err
}
