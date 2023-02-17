package scrm

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/minio/minio-go/v7"
	"github.com/sirupsen/logrus"
	"go-admin/common/log"
	"gorm.io/gorm"
)

type ctxKey string

var (
	GormDB         *gorm.DB
	RedisClient    redis.UniversalClient
	CtiRedisClient redis.UniversalClient
	MinIOClient    *minio.Client
)

var (
	GinContextKey ctxKey = "gin"
)

func Logger() *logrus.Entry {
	return log.Logger().WithFields(logrus.Fields{
		"service": "scrm",
		"module":  "scrm",
	})
}

func GinContext(ctx context.Context) *gin.Context {
	if c, ok := ctx.(*gin.Context); ok {
		return c
	}
	if c, ok := ctx.Value(GinContextKey).(*gin.Context); ok {
		return c
	}
	return nil
}
