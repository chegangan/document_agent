package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"document_agent/app/llmcenter/model"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

// Redis 键常量，便于管理
const (
	// 存储文档信息。键格式: document:info:{messageId}
	docCacheKeyPrefix = "document:info:"
	// 缓存过期时间（秒），设置为 24 小时
	docCacheExpire = 3600 * 24
)

// getDocCacheKey 生成文档对应的 Redis 键。
func getDocCacheKey(messageId string) string {
	return fmt.Sprintf("%s%s", docCacheKeyPrefix, messageId)
}

// DocumentRepository 负责文档的数据访问逻辑，包括缓存处理。
type DocumentRepository struct {
	documentsModel model.DocumentsModel
	redisClient    *redis.Redis
}

// NewDocumentRepository 创建仓库的新实例。
func NewDocumentRepository(dm model.DocumentsModel, r *redis.Redis) *DocumentRepository {
	return &DocumentRepository{
		documentsModel: dm,
		redisClient:    r,
	}
}

// FindDocument 获取文档，使用 Redis 作为 Cache-Aside 缓存层。
func (r *DocumentRepository) FindDocument(ctx context.Context, messageId string) (*model.Documents, error) {
	docCacheKey := getDocCacheKey(messageId)
	logger := logx.WithContext(ctx) // 从 context 获取 logger

	// 1. 先从 Redis 获取
	docJson, err := r.redisClient.Get(docCacheKey)
	if err == nil && docJson != "" {
		// 缓存命中
		var doc model.Documents
		if json.Unmarshal([]byte(docJson), &doc) == nil {
			logger.Infof("cache hit for document: %s", messageId)
			return &doc, nil
		}
		// JSON 反序列化失败，视为缓存未命中
		logger.Errorf("failed to unmarshal document from cache: %v", err)
	}

	logger.Infof("cache miss for document: %s", messageId)
	// 2. 缓存未命中，从数据库查询
	dbDoc, err := r.documentsModel.FindOne(ctx, messageId)
	if err != nil {
		// err 可能是 model.ErrNotFound，这是合法情况。
		return nil, err
	}

	// 3. 将结果写回 Redis 以便后续请求命中缓存
	docBytes, marshalErr := json.Marshal(dbDoc)
	if marshalErr == nil {
		// 使用 Setex 设置带过期时间的键。缓存写入失败不应导致整个操作失败。
		_ = r.redisClient.Setex(docCacheKey, string(docBytes), docCacheExpire)
	} else {
		logger.Errorf("failed to marshal document for cache: %v", marshalErr)
	}

	return dbDoc, nil
}

// UpdateDocumentContent 在数据库更新文档内容并使缓存失效。
func (r *DocumentRepository) UpdateDocumentContent(ctx context.Context, messageId, content string) error {
	logger := logx.WithContext(ctx) // 从 context 获取 logger

	// 1. 更新主数据源（数据库）
	err := r.documentsModel.UpdateContent(ctx, messageId, content)
	if err != nil {
		return err
	}

	// 2. 通过删除缓存键使缓存失效
	docCacheKey := getDocCacheKey(messageId)
	_, delErr := r.redisClient.Del(docCacheKey)
	if delErr != nil {
		// 删除缓存是次要操作，应记录错误但不应使请求失败。
		logger.Errorf("failed to delete cache key %s: %v", docCacheKey, delErr)
	} else {
		logger.Infof("cache invalidated for document: %s", messageId)
	}

	return nil
}
