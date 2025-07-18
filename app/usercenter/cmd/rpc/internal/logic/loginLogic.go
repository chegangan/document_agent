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

type LoginLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *LoginLogic) loginByMobile(mobile, password string) (int64, error) {

	user, err := l.svcCtx.UserModel.FindOneByMobile(l.ctx, mobile)
	if err != nil && err != model.ErrNotFound {
		return 0, fmt.Errorf("根据手机号查询用户信息失败，mobile:%s, err:%v: %w", mobile, err, xerr.ErrDbError)
	}

	if user == nil {
		return 0, fmt.Errorf("mobile: %s: %w", mobile, xerr.ErrUserNotFound)
	}

	if !(tool.Md5ByString(password) == user.Password) {
		return 0, fmt.Errorf("密码匹配出错, mobile: %s: %w", mobile, xerr.ErrUserPassword)
	}

	return user.Id, nil
}

func (l *LoginLogic) Login(in *usercenter.LoginReq) (*usercenter.LoginResp, error) {

	var userId int64
	var err error
	userId, err = l.loginByMobile(in.Mobile, in.Password)
	if err != nil {
		return nil, err
	}

	//2、Generate the token, so that the service doesn't call rpc internally
	generateTokenLogic := NewGenerateTokenLogic(l.ctx, l.svcCtx)
	tokenResp, err := generateTokenLogic.GenerateToken(&usercenter.GenerateTokenReq{
		UserId: userId,
	})
	if err != nil {
		return nil, fmt.Errorf("GenerateToken userId : %d: %w", userId, xerr.ErrGenerateToken)
	}

	return &usercenter.LoginResp{
		AccessToken:  tokenResp.AccessToken,
		AccessExpire: tokenResp.AccessExpire,
		RefreshAfter: tokenResp.RefreshAfter,
	}, nil
}
