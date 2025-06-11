package limiter

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const cacheKeyPrefix = "limiter:"

const redisTimeout = 300 * time.Millisecond

type Limiter struct {
	Redis *redis.Client
	Limit int
}

func (l *Limiter) Increment(ctx context.Context, userID int) (int, error) {
	key := userCounterKey(userID)

	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, redisTimeout)
	defer cancel()

	val, err := l.Redis.Incr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("can't increment user's counter: %w", err)
	}

	if val == 1 {
		if err := l.Redis.Expire(ctx, key, time.Hour).Err(); err != nil {
			return 0, fmt.Errorf("can't set counter expiration: %w", err)
		}
	}

	return int(val), nil
}

func (l *Limiter) LimitExceeded(ctx context.Context, userID int) (bool, error) {
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, redisTimeout)
	defer cancel()

	c, err := l.Redis.Get(ctx, userCounterKey(userID)).Int()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, nil
		}

		return false, err
	}

	return c > l.Limit, nil
}

// userCounterKey builds key which is used to store count of user's purchases per sale.
// It consists of user's ID concatenated to current timestamp rounded down to current hour,
// which is the start of the sale.
func userCounterKey(userID int) string {
	now := time.Now().Truncate(time.Hour).Unix()
	return cacheKeyPrefix + strconv.Itoa(userID) + ":" + strconv.FormatInt(now, 10)
}
