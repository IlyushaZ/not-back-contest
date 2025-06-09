package model

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultCheckoutTimeout = 3 * time.Minute
	CheckoutCodeLen        = 8
)

type Checkout struct {
	Base
	UserID int
	ItemID int
	Code   string
	Error  string
}

type CheckoutCode struct {
	UserID int
	ItemID int
	Rand   string
}

func (c *CheckoutCode) GenerateRand() {
	c.Rand = randAlphaNum(CheckoutCodeLen)
}

func (c *CheckoutCode) String() string {
	return strconv.Itoa(c.UserID) + ":" + strconv.Itoa(c.ItemID) + ":" + c.Rand
}

func (c *CheckoutCode) FromString(s string) error {
	split := strings.Split(s, ":")
	if len(split) != 3 {
		return fmt.Errorf("expected code to have 3 parts, got %d", len(split))
	}

	userID, err := strconv.Atoi(split[0])
	if err != nil {
		return fmt.Errorf("can't parse user_id: %w", err)
	}

	itemID, err := strconv.Atoi(split[1])
	if err != nil {
		return fmt.Errorf("can't parse item_id: %w", err)
	}

	c.UserID = userID
	c.ItemID = itemID
	c.Rand = split[2]

	return nil
}

func randAlphaNum(n int) string {
	const alphaNum = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"

	b := make([]byte, n)
	for i := range b {
		b[i] = alphaNum[rand.Intn(len(alphaNum))] // nolint:gosec
	}

	return string(b)
}
