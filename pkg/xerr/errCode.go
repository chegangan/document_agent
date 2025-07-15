package xerr

// 成功返回
const OK uint32 = 200

/**(前3位代表业务,后三位代表具体功能)**/

// 全局错误码
const SERVER_COMMON_ERROR uint32 = 100001
const REUQEST_PARAM_ERROR uint32 = 100002
const TOKEN_EXPIRE_ERROR uint32 = 100003
const TOKEN_GENERATE_ERROR uint32 = 100004
const DB_ERROR uint32 = 100005
const DB_UPDATE_AFFECTED_ZERO_ERROR uint32 = 100006
const InvalidParameter uint32 = 100007 // 非法参数

// 用户模块错误码 (200xxx)
const (
	// 用户注册相关 (2001xx)
	UserAlreadyExists  uint32 = 200101 // 用户已存在
	UserRegisterFailed uint32 = 200102 // 用户注册失败

	// 用户登录相关 (2002xx)
	UserNotFound      uint32 = 200201 // 用户不存在
	UserPasswordError uint32 = 200202 // 密码错误

	// Token相关 (2003xx)
	GenerateTokenError uint32 = 200301 // 生成token失败
)
