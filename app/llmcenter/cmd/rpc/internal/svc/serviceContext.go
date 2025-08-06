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
	FilesModel        model.FilesModel
	DocumentsModel    model.DocumentsModel
	HistoryDatasModel model.HistorydatasModel
	LlmApiClient      *http.Client // <--- 新增：用于调用 LLM API 的 HTTP 客户端
	RedisClient       *redis.Redis // 2. 添加 RedisClient 字段
}

func NewServiceContext(c config.Config) *ServiceContext {

	sqlConn := sqlx.NewMysql(c.DB.DataSource)

	return &ServiceContext{
		Config:            c,
		ConversationModel: model.NewConversationsModel(sqlConn),
		MessageModel:      model.NewMessagesModel(sqlConn),
		FilesModel:        model.NewFilesModel(sqlConn),
		DocumentsModel:    model.NewDocumentsModel(sqlConn),
		HistoryDatasModel: model.NewHistorydatasModel(sqlConn),
		RedisClient:       redis.MustNewRedis(c.Redis.RedisConf), // 初始化 Redis 客户端
		LlmApiClient: &http.Client{
			// 设置一个总的请求超时，防止请求永远挂起。
			// 注意：对于流式请求，这个超时需要足够长。
			Timeout: time.Duration(c.LlmApiClient.Timeout) * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        c.LlmApiClient.MaxIdleConns,                                 // 最大空闲连接数
				MaxIdleConnsPerHost: c.LlmApiClient.MaxIdleConnsPerHost,                          // 每个主机的最大空闲连接数
				IdleConnTimeout:     time.Duration(c.LlmApiClient.IdleConnTimeout) * time.Second, // 空闲连接超时时间
				DisableCompression:  c.LlmApiClient.DisableCompression,
			},
		},
	}
}
