package tool

import (
	"math/rand"
	"time"
)

// 定义字符串内容类型
const (
	Letters       = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	LettersDigits = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	AllChars      = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()-_=+[]{}|;:,.<>?/`~"
)

// RandomString 生成随机字符串
func RandomString(length int, charset string) string {
	if length <= 0 {
		return ""
	}

	// 初始化随机数种子
	rand.Seed(time.Now().UnixNano())

	// 构造随机字符串
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}
