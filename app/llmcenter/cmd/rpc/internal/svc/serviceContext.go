package svc

import (
	"document_agent/app/llmcenter/cmd/rpc/internal/config"
	"document_agent/app/llmcenter/model"
	"net/http"
	"time"

	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ServiceContext struct {
	Config            config.Config
	ConversationModel model.ConversationsModel
	MessageModel      model.MessagesModel
	LlmApiClient      *http.Client // <--- 新增：用于调用 LLM API 的 HTTP 客户端
	RedisClient       *redis.Redis // 2. 添加 RedisClient 字段
}

func NewServiceContext(c config.Config) *ServiceContext {

	sqlConn := sqlx.NewMysql(c.DB.DataSource)

	return &ServiceContext{
		Config:            c,
		ConversationModel: model.NewConversationsModel(sqlConn),
		MessageModel:      model.NewMessagesModel(sqlConn),
		RedisClient:       redis.MustNewRedis(c.Redis.RedisConf), // 初始化 Redis 客户端
		LlmApiClient: &http.Client{
			// 设置一个总的请求超时，防止请求永远挂起。
			// 注意：对于流式请求，这个超时需要足够长。
			Timeout: 5 * time.Minute,
			Transport: &http.Transport{
				MaxIdleConns:        100,              // 最大空闲连接数
				MaxIdleConnsPerHost: 10,               // 每个主机的最大空闲连接数
				IdleConnTimeout:     90 * time.Second, // 空闲连接超时时间
			},
		},
	}
}
