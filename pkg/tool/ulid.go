package tool

import (
	"math/rand"
	"time"

	"github.com/oklog/ulid/v2"
)

// 生成一个唯一且按时间递增排序的 ULID 字符串
// GenerateULID 生成一个 ULID 作为唯一标识符
func GenerateULID() string {
	t := time.Now().UTC()
	entropy := ulid.Monotonic(rand.New(rand.NewSource(t.UnixNano())), 0)
	return ulid.MustNew(ulid.Timestamp(t), entropy).String()
}
