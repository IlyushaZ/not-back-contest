package cache

import (
	"strings"

	"github.com/redis/go-redis/v9"
)

func NewRedis(addr, user, password string) (*redis.Client, func() error, error) {
	if !strings.Contains(addr, ":") {
		addr = addr + ":6379"
	}

	opts := &redis.Options{
		Addr:     addr,
		Username: user,
		Password: password,
	}

	r := redis.NewClient(opts)

	return r, r.Close, nil
}
