package logic

import (
	"context"
	"fmt"

	"document_agent/app/usercenter/cmd/rpc/internal/svc"
	"document_agent/app/usercenter/cmd/rpc/pb"
	"document_agent/app/usercenter/cmd/rpc/usercenter"
	"document_agent/app/usercenter/model"
	"document_agent/pkg/xerr"

	"github.com/jinzhu/copier"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetUserInfoLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetUserInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserInfoLogic {
	return &GetUserInfoLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetUserInfoLogic) GetUserInfo(in *pb.GetUserInfoReq) (*pb.GetUserInfoResp, error) {

	user, err := l.svcCtx.UserModel.FindOne(l.ctx, in.Id)
	if err != nil && err != model.ErrNotFound {
		return nil, fmt.Errorf("GetUserInfo find user db err, id:%d, err:%v: %w", in.Id, err, xerr.ErrDbError)
	}
	if user == nil {
		return nil, fmt.Errorf("id:%d: %w", in.Id, xerr.ErrUserNotFound)
	}
	var respUser usercenter.User
	_ = copier.Copy(&respUser, user)

	return &usercenter.GetUserInfoResp{
		User: &respUser,
	}, nil

}
