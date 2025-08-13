package config

import (
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	DB struct {
		DataSource string
	}
	XingChen struct {
		FlowID       string // 星辰工作流的 FlowID
		ApiURL       string // 星辰大模型 API 的 URL
		ApiResumeURL string // 星辰大模型 API 的 Resume URL
		ApiKey       string // 星辰大模型 API 的 Key
		ApiSecret    string // 星辰大模型 API 的 Secret
		UploadURL    string // 星辰大模型上传图片的 URL
		FlagCode     string // 星辰大模型判断为后端发送请求的标识码
	}
	LlmApiClient struct {
		Timeout             int
		MaxIdleConns        int
		MaxIdleConnsPerHost int
		IdleConnTimeout     int
		DisableCompression  bool
	}
	Upload struct {
		BaseDir string
	}
	Font struct {
		Path string
	}
	Download struct {
		BaseURL       string // 文件下载的基础 URL
		SignKey       string // 用于签名的密钥
		ExpireSeconds int    // 链接有效期，单位秒
	}
}
