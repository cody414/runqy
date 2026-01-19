package models

import (
	"crypto/tls"
	"fmt"
	"os"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
)

type RedisConns struct {
	RDB      *redis.Client
	AsynqOpt asynq.RedisClientOpt
}

// BuildRedisConns returns configured go-redis client and Asynq options.
func BuildRedisConns() (*RedisConns, error) {
	addr := fmt.Sprintf("%s:%s", os.Getenv("REDIS_HOST"), os.Getenv("REDIS_PORT"))
	password := os.Getenv("REDIS_PASSWORD")
	useTLS := os.Getenv("REDIS_TLS") == "true" // e.g., REDIS_TLS=true

	var tlsConfig *tls.Config
	if useTLS {
		tlsConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:      addr,
		Password:  password,
		DB:        0,
		TLSConfig: tlsConfig,
	})

	redisOpt := asynq.RedisClientOpt{
		Addr:      addr,
		Password:  password,
		DB:        0,
		TLSConfig: tlsConfig,
	}
	return &RedisConns{RDB: rdb, AsynqOpt: redisOpt}, nil

}
