package models

import (
	"crypto/tls"
	"fmt"

	"github.com/Publikey/runqy/config"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
)

type RedisConns struct {
	RDB      *redis.Client
	AsynqOpt asynq.RedisClientOpt
}

// BuildRedisConns returns configured go-redis client and Asynq options.
func BuildRedisConns(cfg *config.Config) (*RedisConns, error) {
	addr := fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort)
	password := cfg.RedisPassword

	var tlsConfig *tls.Config
	if cfg.RedisTLS {
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
