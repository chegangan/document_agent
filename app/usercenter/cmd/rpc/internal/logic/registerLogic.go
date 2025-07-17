package logic

import (
	"context"
	"fmt"

	"document_agent/app/usercenter/cmd/rpc/internal/svc"
	"document_agent/app/usercenter/cmd/rpc/usercenter"
	"document_agent/app/usercenter/model"
	"document_agent/pkg/tool"
	"document_agent/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type RegisterLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterLogic {
	return &RegisterLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *RegisterLogic) Register(in *usercenter.RegisterReq) (*usercenter.RegisterResp, error) {

	if len(in.Mobile) != 11 {
		return nil, fmt.Errorf("Register mobile:%s is not a valid mobile number: %w", in.Mobile, xerr.ErrInvalidParameter)
	}

	user1, err := l.svcCtx.UserModel.FindOneByMobile(l.ctx, in.Mobile)
	if err != nil && err != model.ErrNotFound {
		return nil, fmt.Errorf("mobile:%s, err:%v: %w", in.Mobile, err, xerr.ErrDbError)
	}
	if user1 != nil {
		return nil, fmt.Errorf("Register user exists mobile:%s: %w", in.Mobile, xerr.ErrUserAlreadyExists)
	}

	user := new(model.User)
	user.Mobile = in.Mobile
	if len(in.Nickname) == 0 {
		user.Nickname = "用户" + tool.RandomString(4, tool.Letters)
	}
	if len(in.Password) > 0 {
		user.Password = tool.Md5ByString(in.Password)
	}
	insertResult, err := l.svcCtx.UserModel.Insert(l.ctx, user)
	if err != nil {
		return nil, fmt.Errorf("Register db user Insert err:%v, user:%+v: %w", err, user, xerr.ErrDbError)
	}
	userId, err := insertResult.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("Register db user insertResult.LastInsertId err:%v, user:%+v: %w", err, user, xerr.ErrDbError)
	}

	//2、Generate the token, so that the service doesn't call rpc internally
	generateTokenLogic := NewGenerateTokenLogic(l.ctx, l.svcCtx)
	tokenResp, err := generateTokenLogic.GenerateToken(&usercenter.GenerateTokenReq{
		UserId: userId,
	})
	if err != nil {
		return nil, fmt.Errorf("GenerateToken userId: %d: %w", userId, xerr.ErrGenerateToken)
	}

	return &usercenter.RegisterResp{
		AccessToken:  tokenResp.AccessToken,
		AccessExpire: tokenResp.AccessExpire,
		RefreshAfter: tokenResp.RefreshAfter,
	}, nil
}
