package config

import (
	"flag"
	"time"

	"github.com/IlyushaZ/not-back-contest/pkg/model"
)

type Config struct {
	LogLevel   string
	ListenAddr string

	PostgresAddr     string // Postgres address in host[:port] format
	PostgresDB       string
	PostgresUser     string
	PostgresPassword string

	RedisAddr     string // Redis address in host[:port] format
	RedisUser     string // Redis user
	RedisPassword string // Redis password

	LimiterFailOpen bool
	CacheCheckouts  bool // whether to save and check checkout info to redis
	PurchasesLimit  int
	CheckoutTimeout time.Duration

	CheckoutsBatchSize     int
	CheckoutsFlushInterval time.Duration

	// Items generator params
	SalesCount   int
	ItemsPerSale int
}

func New() *Config {
	c := &Config{}

	flag.StringVar(&c.LogLevel, "logLevel", LookupEnvString("LOG_LEVEL", "DEBUG"), "Set log level: DEBUG, INFO, WARNING, ERROR.")
	flag.StringVar(&c.ListenAddr, "listenAddr", LookupEnvString("LISTEN_ADDR", ":8000"), `Address in form of "[host]:port" that HTTP server should be listening on.`)

	flag.StringVar(&c.PostgresAddr, "postgresAddr", LookupEnvString("POSTGRES_ADDR", "127.0.0.1:5432"), "Set PostgreSQL address as host:port, where port is optional (without TLS).")
	flag.StringVar(&c.PostgresDB, "postgresDB", LookupEnvString("POSTGRES_DB", "notbackcontest"), "Set PostgreSQL DB.")
	flag.StringVar(&c.PostgresUser, "postgresUser", LookupEnvString("POSTGRES_USER", "develop"), "Set PostgreSQL user.")
	flag.StringVar(&c.PostgresPassword, "postgresPassword", LookupEnvString("POSTGRES_PASSWORD", "develop"), "Set PostgreSQL password.")

	flag.StringVar(&c.RedisAddr, "redisAddr", LookupEnvString("REDIS_ADDR", "127.0.0.1:6379"), "Redis address in host[:port] format.")
	flag.StringVar(&c.RedisUser, "redisUser", LookupEnvString("REDIS_USER", ""), "Redis user.")
	flag.StringVar(&c.RedisPassword, "redisPassword", LookupEnvString("REDIS_PASSWORD", ""), "Redis password.")

	flag.BoolVar(&c.LimiterFailOpen, "limiterFailOpen", LookupEnvBool("LIMITER_FAIL_OPEN", false), "Set to make limiter allow request if failed to check limits.")
	flag.BoolVar(&c.CacheCheckouts, "cacheCheckouts", LookupEnvBool("CACHE_CHECKOUTS", false), "Set to cache limiter info. May be useful when single item is requested many times.")
	flag.IntVar(&c.PurchasesLimit, "purchasesLimit", LookupEnvInt("PURCHASES_LIMIT", 10), "Number of purchases that single user can make within one sale.")
	flag.DurationVar(&c.CheckoutTimeout, "checkoutTimeout", LookupEnvDuration("CHECKOKUT_TIMEOUT", model.DefaultCheckoutTimeout), "How long item can be reserved by user in format that can be parsed by go's time.ParseDuration.")

	flag.IntVar(&c.CheckoutsBatchSize, "checkoutsBatchSize", LookupEnvInt("CHECKOUTS_BATCH_SIZE", 500), "Number of checkout attempts to be stored in buffer before being flushed.")
	flag.DurationVar(&c.CheckoutsFlushInterval, "checkoutsFlushInterval", LookupEnvDuration("CHECKOUTS_FLUSH_INTERVAL", 10*time.Second), "How ofter checkouts buffer should be flushed.")

	flag.IntVar(&c.SalesCount, "salesCount", LookupEnvInt("SALES_COUNT", 1), "Number of sales to generate (only for items-generator).")
	flag.IntVar(&c.ItemsPerSale, "itemsPerSale", LookupEnvInt("ITEMS_PER_SALE", model.ItemsPerSale), "Number of items per sale.")

	flag.Parse()

	return c
}
