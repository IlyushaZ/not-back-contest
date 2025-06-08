# Human-made Flash Sale API

This service provides API for checking out and purchasing items.

## Architecture description (please read me)
The system is designed with a strong focus on data consistency and horizontal scalability. It uses PostgreSQL as the source of truth, with all critical operations—such as item reservations, checkout validation, and purchase confirmation—executed within PostgreSQL transactions to ensure strong consistency and data integrity.

Redis is used purely as a caching layer, for optimizing read-heavy operations such as checking item availability or user quotas. However, no core application logic depends on Redis, meaning that if Redis becomes unavailable or fails, the system will continue to function correctly using only the PostgreSQL backend.

The caching layer can be disabled if necessary, as it is not essential for the correctness of the system. It is primarily useful in scenarios, such as when many users attempt to check out the same item simultaneously. In these cases, caching helps reduce database load.

Redis is also used to enforce per-user purchase limits during each flash sale by tracking the number of items a user has successfully purchased. User's counter is updated after successul purchase and checked before the checkout. You can determine whether unsuccessful attempt to check limits should result in error returned to user via `--limiterFailOpen` setting (or `LIMITER_FAIL_OPEN` env variable).

Items are populated by a cron job that runs once per hour as a single instance, generating exactly 10,000 items for the current sale window (or N sales forward, which is configured by a parameter). While a more robust solution could involve distributed workers with coordination or leader election to ensure consistency and fault tolerance, I opted for the simpler approach **due to my laziness** and lack of time.

When a user performs a checkout, the selected item is **reserved exclusively for that user for a limited time** — by default, 3 minutes (configurable via settings). During this reservation window, the item can be purchased using the issued checkout code. If the reservation expires before the user completes the purchase, the item becomes available for others to check out. If a user requests the same item multiple times during the active reservation period, they will receive the same checkout code as initially issued. This ensures idempotent behavior and prevents duplicate reservations for the same item.

I suggest you to get familiar with the code because it provides many comments explaining why certain things are implemented and simplified in such way.

## API description

- `/checkout?user_id={user_id}&item_id={item_id}` returns **status 200** and **code** if user has successfully checked out the item. If sale is over or item was already checked out or sold, **status 412** is returned with corresponding error message. If user has exceeded his purchases limit, **status 429** is returned. If **status 500** is returned... 💀💀💀
- `/checkout?code={code}` returns **status 200** if user has successfully purchased the item. If code or sale has expired, **status 404** is returned which means that no such checkout or item was found.

## How to run

To run server and items generator with all their dependencies (PostgreSQL, Redis), run:
```
docker compose up -d
```
The server will be accessible on port `:8000`.

## Settings

`Server` and `item-generator` provide several parameters which can be set on start:
```
-cacheCheckouts
   	Set to cache limiter info. May be useful when single item is requested many times.
-checkoutTimeout duration
   	How long item can be reserved by user in format that can be parsed by go's time.ParseDuration. (default 3m0s)
-itemsPerSale int
   	Number of items per sale. (default 10000)
-limiterFailOpen
   	Set to make limiter allow request if failed to check limits.
-listenAddr string
   	Address in form of "[host]:port" that HTTP server should be listening on. (default ":8000")
-logLevel string
   	Set log level: DEBUG, INFO, WARNING, ERROR. (default "DEBUG")
-postgresAddr string
   	Set PostgreSQL address as host:port, where port is optional (without TLS). (default "127.0.0.1:5432")
-postgresDB string
   	Set PostgreSQL DB. (default "notbackcontest")
-postgresPassword string
   	Set PostgreSQL password. (default "develop")
-postgresUser string
   	Set PostgreSQL user. (default "develop")
-purchasesLimit int
   	Number of purchases that single user can make within one sale. (default 10)
-redisAddr string
   	Redis address in host[:port] format. (default "127.0.0.1:6379")
-redisPassword string
   	Redis password.
-redisUser string
   	Redis user.
-salesCount int
   	Number of sales to generate. (default 1)
```

## Performance tests
I've only implemented basic test on checkout due to lack of time 👉👈. But I am pretty sure my app is ⚡***BLAZINGLY FAST***⚡.

To run it, you will need to install [vegeta](https://github.com/tsenart/vegeta) and run:
```
chmod +x ./test/generate_targets.sh ./test/test_checkout.sh
./test/test_checkout.sh
```

It starts the service with dependencies via `docker compose` and runs script `test/generate_targets.sh` which generate URLs with different `item_id`'s and `user_id`'s. It generates `item_id`'s in such way that some identifiers repeat, which, IMO, looks like real usage. After that, `vegeta attack` is run with generated `targets.txt` given as `-target` argument.

## Easter egg
Press `ctrl-shift-f` in your IDE to see the things I didn't have enough time to implement.
