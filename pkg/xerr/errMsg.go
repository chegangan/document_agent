package xerr

var message map[uint32]string

func init() {
	// 全局错误码
	message = make(map[uint32]string)
	message[OK] = "SUCCESS"
	message[SERVER_COMMON_ERROR] = "服务器开小差啦,稍后再来试一试"
	message[REUQEST_PARAM_ERROR] = "参数错误"
	message[TOKEN_EXPIRE_ERROR] = "token失效，请重新登陆"
	message[TOKEN_GENERATE_ERROR] = "生成token失败"
	message[DB_ERROR] = "数据库繁忙,请稍后再试"
	message[DB_UPDATE_AFFECTED_ZERO_ERROR] = "更新数据影响行数为0"
	message[InvalidParameter] = "非法参数"

	// 用户模块错误码
	message[GenerateTokenError] = "生成token失败,请稍后再试"
	message[UserAlreadyExists] = "用户已存在"
	message[UserRegisterFailed] = "用户注册失败,请稍后再试"
	message[UserNotFound] = "用户不存在"
	message[UserPasswordError] = "密码错误"
}

func MapErrMsg(errcode uint32) string {
	if msg, ok := message[errcode]; ok {
		return msg
	} else {
		return "服务器开小差啦,稍后再来试一试"
	}
}

func IsCodeErr(errcode uint32) bool {
	if _, ok := message[errcode]; ok {
		return true
	} else {
		return false
	}
}
