package service

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/IlyushaZ/not-back-contest/pkg/model"
	"github.com/redis/go-redis/v9"
)

const (
	checkoutsKeyPrefix = "checkouts:"
)

// ItemCaching is a caching layer which is intended to be called before ItemGeneric.
// It may be helpful if we see that single item is tried to be checked out many times.
type ItemCaching struct {
	Item

	Redis           *redis.Client
	CheckoutTimeout time.Duration
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
	key := checkoutCacheKey(itemID)

	redisCtx, cancel := context.WithTimeout(ctx, time.Millisecond*300)
	defer cancel()

	val, err := ic.Redis.Get(redisCtx, key).Result()
	switch {
	case err == redis.Nil:
		// do nothing
	case err != nil:
		slog.Error("can't get checkout info from redis", slog.Any("error", err))

	default:
		ccv, err := parseCheckoutCacheVal(val)
		if err != nil {
			slog.Error("can't parse checkout cache value", slog.String("val", val), slog.Any("error", err))
			break
		}

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

	ccv := checkoutCacheVal{now.Add(ic.CheckoutTimeout), userID, code}

	go func() {
		redisCtx, cancel := context.WithTimeout(context.TODO(), time.Second)
		defer cancel()

		// i guess we can not really concern about atomicity here,
		// because only one user buying this item will reach this code section at a time
		if err = ic.Redis.Set(redisCtx, key, ccv.String(), ic.CheckoutTimeout).Err(); err != nil {
			slog.Error("can't set checkout info in redis", slog.Any("error", err))
		}
	}()

	return
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
