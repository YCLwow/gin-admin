package config

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// Redis配置
const (
	RedisAddr     = "localhost:6379"
	RedisPassword = ""
	RedisDB       = 0
	TokenExpiry   = 24 * time.Hour
)

var (
	// RedisClient 全局Redis客户端
	RedisClient *redis.Client
)

// InitRedis 初始化Redis连接
func InitRedis() {
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     RedisAddr,
		Password: RedisPassword,
		DB:       RedisDB,
	})

	// 测试连接
	ctx := context.Background()
	if _, err := RedisClient.Ping(ctx).Result(); err != nil {
		log.Fatalf("Redis连接失败: %v", err)
	}

	log.Println("成功连接到Redis")
}

// SaveToken 将token保存到Redis
func SaveToken(ctx context.Context, userID int, username, token string) error {
	// 使用用户ID作为键，保存token
	key := "user:token:" + username
	return RedisClient.Set(ctx, key, token, TokenExpiry).Err()
}

// GetToken 从Redis获取token
func GetToken(ctx context.Context, username string) (string, error) {
	key := "user:token:" + username
	return RedisClient.Get(ctx, key).Result()
}

// InvalidateToken 使token失效
func InvalidateToken(ctx context.Context, username string) error {
	key := "user:token:" + username
	return RedisClient.Del(ctx, key).Err()
}
