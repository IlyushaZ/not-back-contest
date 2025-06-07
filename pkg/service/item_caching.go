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

// timestamp|userID|code|(status?)
type checkoutCacheValue struct {
	until  time.Time
	userID int
	code   string
}

// checkouts:item_id -> timestamp|userID|code|(status?)
func (c checkoutCacheValue) String() string {
	return strconv.Itoa(c.userID) + "|" + strconv.FormatInt(c.until.Unix(), 10) + "|" + c.code
}

// Checkout calls to Item.Checkout, but checks whether the item is checked out using cache.
// It may help in cases when we have many checkouts with single item.
// If redis has no info about item's checkout or checkout has already expired, we use slower path (go to DB).
// Errors occurring when calling redis are not returned.
func (ic *ItemCaching) Checkout(ctx context.Context, userID, itemID int) (code string, err error) {
	now := time.Now()
	key := checkoutCacheKey(itemID)

	val, err := ic.Redis.Get(ctx, key).Result()
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

			return "", model.ErrItemUnavailable
		}
	}

	// slower path - try to checkout in DB
	code, err = ic.Item.Checkout(ctx, userID, itemID)
	if err != nil {
		return
	}

	// TODO: what if we move this to separate goroutine?
	ccv := checkoutCacheValue{now.Add(ic.CheckoutTimeout), userID, code}

	if err = ic.Redis.Set(ctx, key, ccv.String(), ic.CheckoutTimeout).Err(); err != nil {
		slog.Error("can't set checkout info in redis", slog.Any("error", err))
	}

	return
}

func checkoutCacheKey(itemID int) string {
	return checkoutsKeyPrefix + strconv.Itoa(itemID)
}

func parseCheckoutCacheVal(val string) (checkoutCacheValue, error) {
	split := strings.Split(val, "|")
	if len(split) != 3 {
		return checkoutCacheValue{}, fmt.Errorf("expected val to consist of 3 parts, got %d", len(split))
	}

	var (
		ccv checkoutCacheValue
		err error
	)

	ccv.userID, err = strconv.Atoi(split[0])
	if err != nil {
		return checkoutCacheValue{}, fmt.Errorf("can't parse userID: %w", err)
	}

	ts, err := strconv.ParseInt(split[1], 10, 64)
	if err != nil {
		return checkoutCacheValue{}, fmt.Errorf("can't parse timestamp: %w", err)
	}

	ccv.until = time.Unix(ts, 0)
	ccv.code = split[2]

	return ccv, nil
}
