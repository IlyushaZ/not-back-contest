package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/IlyushaZ/not-back-contest/pkg/model"
	"github.com/redis/go-redis/v9"
)

const (
	checkoutsKeyPrefix = "checkouts:"
)

var (
	errCacheMiss = errors.New("cache miss")
)

// ItemCaching is a caching layer which is intended to be called before ItemGeneric.
// It may be helpful if we see that single item is tried to be checked out many times.
type ItemCaching struct {
	Item

	redis           *redis.Client
	checkoutTimeout time.Duration
	localCache      map[int]checkoutCacheVal
	mu              sync.RWMutex
}

func NewItemCaching(i Item, redis *redis.Client, checkoutTimeout time.Duration) *ItemCaching {
	ic := &ItemCaching{
		Item:            i,
		redis:           redis,
		checkoutTimeout: checkoutTimeout,
		localCache:      make(map[int]checkoutCacheVal, 10000),
	}

	go func() {
		t := time.NewTicker(5 * time.Second)
		defer t.Stop()

		slog.Info("local cache cleaner started")

		for {
			select {
			case <-t.C:
				ic.cleanupLocalCache()
			}
		}
	}()

	return ic
}

type checkoutCacheVal struct {
	until  time.Time
	userID int
	code   string
}

func (c checkoutCacheVal) String() string {
	return strconv.Itoa(c.userID) + "|" + strconv.FormatInt(c.until.Unix(), 10) + "|" + c.code
}

// Checkout calls to Item.Checkout, but checks whether the item is checked out using cache.
// It may help in cases when we have many checkouts with single item.
// If redis has no info about item's checkout or checkout has already expired, we use slower path (go to DB).
// Errors occurring when calling redis are not returned.
func (ic *ItemCaching) Checkout(ctx context.Context, userID, itemID int) (code string, err error) {
	now := time.Now()

	ccv, err := ic.getCheckoutCacheVal(ctx, itemID)
	switch {
	case errors.Is(err, errCacheMiss):
		// do nothing

	case err != nil:
		slog.Error("can't get checkout info from cache", slog.Any("error", err))

	default:
		if now.Before(ccv.until) {
			if ccv.userID == userID { // it's us who had checked out the item before
				return ccv.code, nil
			}

			slog.Debug("someone cooked here")

			return "", model.ErrItemUnavailable
		} else {
			ic.mu.Lock()
			delete(ic.localCache, itemID)
			ic.mu.Unlock()
		}
	}

	// slower path - try to checkout in DB
	code, err = ic.Item.Checkout(ctx, userID, itemID)
	if err != nil {
		return
	}

	ccv = checkoutCacheVal{now.Add(ic.checkoutTimeout), userID, code}

	ic.mu.Lock()
	ic.localCache[itemID] = ccv
	ic.mu.Unlock()

	go func() {
		redisCtx, cancel := context.WithTimeout(context.TODO(), time.Second)
		defer cancel()

		key := checkoutCacheKey(itemID)

		// i guess we can not really concern about atomicity here,
		// because only one user buying this item will reach this code section at a time
		if err = ic.redis.Set(redisCtx, key, ccv.String(), ic.checkoutTimeout).Err(); err != nil {
			slog.Error("can't set checkout info in redis", slog.Any("error", err))
		}
	}()

	return
}

func (ic *ItemCaching) getCheckoutCacheVal(ctx context.Context, itemID int) (checkoutCacheVal, error) {
	var (
		ccv checkoutCacheVal
		err error
	)

	ic.mu.RLock()
	ccv, ok := ic.localCache[itemID]
	if ok {
		ic.mu.RUnlock()
		return ccv, nil
	}

	ic.mu.RUnlock()

	key := checkoutCacheKey(itemID)

	redisCtx, cancel := context.WithTimeout(ctx, time.Millisecond*300)
	defer cancel()

	val, err := ic.redis.Get(redisCtx, key).Result()
	switch {
	case err == redis.Nil:
		return ccv, errCacheMiss
	case err != nil:
		return ccv, fmt.Errorf("can't get checkout info from redis: %w", err)

	default:
		ccv, err := parseCheckoutCacheVal(val)
		if err != nil {
			return ccv, fmt.Errorf("can't parse checkout cache val: %w", err)
		}

		return ccv, nil
	}
}

func (ic *ItemCaching) cleanupLocalCache() {
	expired := []int{}
	now := time.Now()

	ic.mu.RLock()
	for k, v := range ic.localCache {
		if v.until.Before(now) {
			expired = append(expired, k)
		}
	}
	ic.mu.RUnlock()

	for _, id := range expired {
		ic.mu.Lock()
		delete(ic.localCache, id)
		ic.mu.Unlock()
	}

	slog.Debug("local cache cleaned up", slog.Int("items_deleted", len(expired)))
}

func checkoutCacheKey(itemID int) string {
	return checkoutsKeyPrefix + strconv.Itoa(itemID)
}

func parseCheckoutCacheVal(val string) (checkoutCacheVal, error) {
	split := strings.Split(val, "|")
	if len(split) != 3 {
		return checkoutCacheVal{}, fmt.Errorf("expected val to consist of 3 parts, got %d", len(split))
	}

	var (
		ccv checkoutCacheVal
		err error
	)

	ccv.userID, err = strconv.Atoi(split[0])
	if err != nil {
		return checkoutCacheVal{}, fmt.Errorf("can't parse userID: %w", err)
	}

	ts, err := strconv.ParseInt(split[1], 10, 64)
	if err != nil {
		return checkoutCacheVal{}, fmt.Errorf("can't parse timestamp: %w", err)
	}

	ccv.until = time.Unix(ts, 0)
	ccv.code = split[2]

	return ccv, nil
}
