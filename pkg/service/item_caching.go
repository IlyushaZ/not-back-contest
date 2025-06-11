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
	// the more number of instances we have - the more useless this cache becomes,
	// but it does not give much overhead (i guess so)
	localCache []checkoutCacheVal
	// TODO: we can get rid of mutex
	// or cut down the time spent on locking by sharding local cache into multiple segments.
	mu sync.RWMutex
}

func NewItemCaching(i Item, redis *redis.Client, checkoutTimeout time.Duration, itemsPerSale int) *ItemCaching {
	ic := &ItemCaching{
		Item:            i,
		redis:           redis,
		checkoutTimeout: checkoutTimeout,
		localCache:      make([]checkoutCacheVal, itemsPerSale),
	}

	return ic
}

type checkoutCacheVal struct {
	until  time.Time
	itemID int // only for local cache
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

	ccv, err := ic.getCheckoutCacheVal(ctx, itemID, now)
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
		}
	}

	// slower path - try to checkout in DB
	code, err = ic.Item.Checkout(ctx, userID, itemID)
	if err != nil {
		return
	}

	ccv = checkoutCacheVal{now.Add(ic.checkoutTimeout), itemID, userID, code}

	ic.mu.Lock()
	ic.localCache[itemID%len(ic.localCache)] = ccv
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

func (ic *ItemCaching) getCheckoutCacheVal(ctx context.Context, itemID int, now time.Time) (checkoutCacheVal, error) {
	var (
		localIdx = itemID % len(ic.localCache)
		ccv      checkoutCacheVal
		err      error
	)

	ic.mu.RLock()
	ccv = ic.localCache[localIdx]
	ic.mu.RUnlock()

	// Check the date as well because when it's more than one instance running,
	// someone could have already checked out the item through the other instance.
	// In this case we should go to redis anyway
	if ccv.itemID == itemID && now.Before(ccv.until) {
		slog.Debug("found value in local cache", slog.Int("item_id", itemID))
		return ccv, nil
	}

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

		ccv.itemID = itemID

		// populate local cache
		ic.mu.Lock()
		ic.localCache[localIdx] = ccv
		ic.mu.Unlock()

		return ccv, nil
	}
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
