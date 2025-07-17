package xerr

import "github.com/zeromicro/x/errors"

var (
	// 全局错误码 100xxx
	ErrServerCommon         = errors.New(100001, "服务器开小差啦,稍后再来试一试")
	ErrRequestParam         = errors.New(100002, "参数错误")
	ErrTokenExpire          = errors.New(100003, "token失效，请重新登陆")
	ErrTokenGenerate        = errors.New(100004, "生成token失败")
	ErrDbError              = errors.New(100005, "数据库繁忙,请稍后再试")
	ErrDbUpdateAffectedZero = errors.New(100006, "更新数据影响行数为0")
	ErrInvalidParameter     = errors.New(100007, "非法参数")

	// 用户模块错误码 200xxx
	ErrUserAlreadyExists  = errors.New(200101, "用户已存在")
	ErrUserRegisterFailed = errors.New(200102, "用户注册失败,请稍后再试")
	ErrUserNotFound       = errors.New(200201, "用户不存在")
	ErrUserPassword       = errors.New(200202, "密码错误")
	ErrGenerateToken      = errors.New(200301, "生成token失败,请稍后再试")
)
