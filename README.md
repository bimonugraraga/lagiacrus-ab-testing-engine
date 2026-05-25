# lagiacrus-ab-testing-engine

Go package for deterministic A/B variant assignment with Redis-backed exposure + conversion analytics.

## Features

- Deterministic user → variant assignment (weight-based, 0–99 bucket)
- Optional bulk assignment (batched + concurrent)
- Exposure tracking with de-duplication (Bloom/bitmap in Redis)
- Conversion tracking with exposure prerequisite + de-duplication
- Simple analytics readout: exposure, conversion, conversion rate, expiry time

## Requirements

- Go 1.22+
- Redis (used for experiment metadata, exposure/conversion locks, and analytics)

## Install

```bash
go get github.com/bimonugraraga/lagiacrus-ab-testing-engine@latest
```

Import path (from `go.mod`):

```go
import abtestingengine "github.com/bimonugraraga/lagiacrus-ab-testing-engine"
```

## Quickstart

```go
package main

import (
	"context"
	"fmt"
	"time"

	abtestingengine "github.com/bimonugraraga/lagiacrus-ab-testing-engine"
	"github.com/redis/go-redis/v9"
)

func main() {
	var ctx context.Context = context.Background()

	var rdb *redis.Client = redis.NewClient(&redis.Options{Addr: "localhost:6379"})

	var exp abtestingengine.Experiment = abtestingengine.Experiment{
		ID: "exp_checkout_button",
		Variants: []abtestingengine.Variant{
			{ID: "control", Name: "Control", Weight: 50},
			{ID: "treatment", Name: "Treatment", Weight: 50},
		},
	}

	var userID string = "user_123"
	var expiredAt time.Time = time.Now().UTC().AddDate(0, 0, 30)

	var variant abtestingengine.Variant
	var err error
	variant, err = exp.AssignUser(ctx, userID, rdb, expiredAt)
	if err != nil {
		panic(err)
	}
	fmt.Println("assigned variant:", variant.ID)

	var event abtestingengine.UserConversion = abtestingengine.UserConversion{
		UserID:       userID,
		ExperimentID: exp.ID,
		VariantID:    variant.ID,
	}

	if err := event.ExposedUser(ctx, rdb); err != nil {
		panic(err)
	}

	if err := event.ConversionUser(ctx, rdb); err != nil {
		panic(err)
	}

	analytics, err := exp.GetAnalytics(ctx, rdb)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", analytics)
}
```

## Concepts

### Assignment

- `(*Experiment).AssignUser(...)` assigns a user into one of 100 buckets using a deterministic hash of `userID` and `experimentID`.
- Variant selection is based on cumulative weights; weights must sum to 100.
- If a Redis client is provided, the experiment metadata and expiry timestamp are stored once using `SETNX`.

When to assign without Redis (variant lookup only):

- Use this when you only need to know which variant a user maps to (deterministic bucketing) and you do not want to store anything in Redis.
- Pass `redisClient = nil`. In this mode, `expiredTime` is not used because no Redis TTL/metadata is written.
- You cannot use `ExposedUser`, `ConversionUser`, or `GetAnalytics` without Redis metadata, because they require the experiment expiry timestamp to be stored in Redis.

Signature:

```go
func (e *Experiment) AssignUser(
	ctx context.Context,
	userID string,
	redisClient *redis.Client,
	expiredTime time.Time,
) (Variant, error)
```

Parameter types:

- `ctx`: `context.Context`
- `userID`: `string`
- `redisClient`: `*redis.Client` (pass `nil` to skip Redis writes)
- `expiredTime`: `time.Time` (UTC recommended; if zero, defaults to now+30 days when writing to Redis)

```go
var ctx context.Context = context.Background()
var rdb *redis.Client = redis.NewClient(&redis.Options{Addr: "localhost:6379"})

var exp abtestingengine.Experiment = abtestingengine.Experiment{
	ID: "exp_checkout_button",
	Variants: []abtestingengine.Variant{
		{ID: "control", Name: "Control", Weight: 50},
		{ID: "treatment", Name: "Treatment", Weight: 50},
	},
}

var userID string = "user_123"
var expiredAt time.Time = time.Now().UTC().AddDate(0, 0, 30)

variant, err := exp.AssignUser(ctx, userID, rdb, expiredAt)
if err != nil {
	panic(err)
}
fmt.Println("assigned:", variant.ID)
```

Variant lookup only (no Redis writes):

```go
var ctx context.Context = context.Background()

var exp abtestingengine.Experiment = abtestingengine.Experiment{
	ID: "exp_checkout_button",
	Variants: []abtestingengine.Variant{
		{ID: "control", Name: "Control", Weight: 50},
		{ID: "treatment", Name: "Treatment", Weight: 50},
	},
}

var userID string = "user_123"

variant, err := exp.AssignUser(ctx, userID, nil, time.Time{})
if err != nil {
	panic(err)
}
fmt.Println("assigned:", variant.ID)
```

### Exposure

- `(*UserConversion).ExposedUser(...)`:
  - Requires the experiment expiry timestamp to exist in Redis (set via `AssignUser`/`AssignUserBulk` with Redis).
  - Uses a bitmap-based Bloom-style lock to de-duplicate exposures per (experiment, user).
  - Increments a Redis hash counter per variant only on first exposure.

Signature:

```go
func (u *UserConversion) ExposedUser(ctx context.Context, redisClient *redis.Client) error
```

Parameter types:

- `ctx`: `context.Context`
- `redisClient`: `*redis.Client`
- `UserConversion.UserID`: `any` (commonly `string` or `int64`)
- `UserConversion.ExperimentID`: `any` (commonly `string`)
- `UserConversion.VariantID`: `any` (commonly `string`)

```go
var ctx context.Context = context.Background()
var rdb *redis.Client = redis.NewClient(&redis.Options{Addr: "localhost:6379"})

var expID string = "exp_checkout_button"
var variantID string = "control"
var userID string = "user_123"

var evt abtestingengine.UserConversion = abtestingengine.UserConversion{
	UserID:       userID,
	ExperimentID: expID,
	VariantID:    variantID,
}
if err := evt.ExposedUser(ctx, rdb); err != nil {
	panic(err)
}
```

### Conversion

- `(*UserConversion).ConversionUser(...)`:
  - Requires the user to have been exposed (checks exposure bitmap).
  - De-duplicates conversions per (experiment, user).
  - Increments a Redis hash counter per variant only on first conversion.

Signature:

```go
func (u *UserConversion) ConversionUser(ctx context.Context, redisClient *redis.Client) error
```

Parameter types:

- `ctx`: `context.Context`
- `redisClient`: `*redis.Client`
- `UserConversion.UserID`: `any` (commonly `string` or `int64`)
- `UserConversion.ExperimentID`: `any` (commonly `string`)
- `UserConversion.VariantID`: `any` (commonly `string`)

```go
var ctx context.Context = context.Background()
var rdb *redis.Client = redis.NewClient(&redis.Options{Addr: "localhost:6379"})

var expID string = "exp_checkout_button"
var variantID string = "control"
var userID string = "user_123"

var evt abtestingengine.UserConversion = abtestingengine.UserConversion{
	UserID:       userID,
	ExperimentID: expID,
	VariantID:    variantID,
}
if err := evt.ConversionUser(ctx, rdb); err != nil {
	panic(err)
}
```

### Analytics

- `(*Experiment).GetAnalytics(...)` returns:
  - exposure counts by variant
  - conversion counts by variant
  - conversion rate by variant
  - expiry time (local + UTC)

Signature:

```go
func (e *Experiment) GetAnalytics(ctx context.Context, redisClient *redis.Client) (ResultAnalytics, error)
```

Parameter types:

- `ctx`: `context.Context`
- `redisClient`: `*redis.Client`
- Return: `ResultAnalytics` (plus `error`)

```go
var ctx context.Context = context.Background()
var rdb *redis.Client = redis.NewClient(&redis.Options{Addr: "localhost:6379"})

var exp abtestingengine.Experiment = abtestingengine.Experiment{ID: "exp_checkout_button"}

var analytics abtestingengine.ResultAnalytics
var err error

analytics, err = exp.GetAnalytics(ctx, rdb)
if err != nil {
	panic(err)
}
fmt.Printf("%+v\n", analytics)
```

## Bulk Assignment

`(*Experiment).AssignUserBulk(...)` assigns many users concurrently and groups user IDs by variant:

Signature:

```go
func (e *Experiment) AssignUserBulk(
	ctx context.Context,
	userIDs []string,
	batchSize int,
	numWorkers int,
	redisClient *redis.Client,
	expiredTime time.Time,
) (ResultAssignBulk, error)
```

Parameter types:

- `ctx`: `context.Context`
- `userIDs`: `[]string`
- `batchSize`: `int`
- `numWorkers`: `int`
- `redisClient`: `*redis.Client` (pass `nil` to skip Redis writes)
- `expiredTime`: `time.Time` (UTC recommended; if writing to Redis, must be in the future)

```go
var ctx context.Context = context.Background()
var rdb *redis.Client = redis.NewClient(&redis.Options{Addr: "localhost:6379"})

var exp abtestingengine.Experiment = abtestingengine.Experiment{
	ID: "exp_checkout_button",
	Variants: []abtestingengine.Variant{
		{ID: "control", Name: "Control", Weight: 50},
		{ID: "treatment", Name: "Treatment", Weight: 50},
	},
}

var userIDs []string = []string{"u1", "u2", "u3", "u4", "u5"}
var batchSize int = 200
var numWorkers int = 8
var expiredAt time.Time = time.Now().UTC().AddDate(0, 0, 30)

var res abtestingengine.ResultAssignBulk
var err error

res, err = exp.AssignUserBulk(ctx, userIDs, batchSize, numWorkers, rdb, expiredAt)
if err != nil {
	panic(err)
}
fmt.Println(len(res.ResultsBulk["control"].UserID))
```

Notes:

- `batchSize` controls how many user IDs one worker handles.
- `numWorkers` controls how many batches are processed concurrently.
- Passing a Redis client stores experiment metadata + expiry once (it does not write per-user assignments to Redis).

Example with inputs:

```go
var ctx context.Context = context.Background()
var rdb *redis.Client = redis.NewClient(&redis.Options{Addr: "localhost:6379"})

var exp abtestingengine.Experiment = abtestingengine.Experiment{
	ID: "exp_checkout_button",
	Variants: []abtestingengine.Variant{
		{ID: "control", Name: "Control", Weight: 50},
		{ID: "treatment", Name: "Treatment", Weight: 50},
	},
}

var userIDs []string = []string{"u1", "u2", "u3", "u4", "u5"}
var expiredAt time.Time = time.Now().UTC().AddDate(0, 0, 30)

var res abtestingengine.ResultAssignBulk
var err error

res, err = exp.AssignUserBulk(ctx, userIDs, 2, 4, rdb, expiredAt)
if err != nil {
	panic(err)
}

for variantID, group := range res.ResultsBulk {
	fmt.Println("variant:", variantID, "users:", len(group.UserID))
}
```

## Redis Data Model

Keys created (prefix `lagiacrus:experiment`):

- `lagiacrus:experiment:{<experimentID>}`: JSON experiment detail (stored once)
- `lagiacrus:experiment:{<experimentID>}:expired_time`: UNIX expiry seconds (stored once)
- `lagiacrus:experiment:{<experimentID>}:exposure_lock`: bitmap for exposure de-duplication (TTL matches expiry)
- `lagiacrus:experiment:{<experimentID>}:exposure`: hash counter variant → exposure count (TTL matches expiry)
- `lagiacrus:experiment:{<experimentID>}:conversion_lock`: bitmap for conversion de-duplication (TTL matches expiry)
- `lagiacrus:experiment:{<experimentID>}:conversion`: hash counter variant → conversion count (TTL matches expiry)

## Bloom/Bitmap Sizing

The bitmap size is controlled by `BloomIndexK` and `BloomIndexM` in the package constants. `BloomIndexM` is in bits; Redis memory usage will depend on how the bitmap expands and Redis’ internal encoding, but the upper bound is roughly `BloomIndexM / 8` bytes.
