package abtestingengine

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

var convertUserLua = redis.NewScript(`
-- KEYS:
-- 1 = exposure bitmap
-- 2 = conversion bitmap
-- 3 = conversion analytics hash

-- ARGV:
-- bloom indexes..., variant, ttl

local k = #ARGV - 2
local variant = ARGV[#ARGV-1]
local ttl = ARGV[#ARGV]

------------------------------------------------
-- 1. check exposure exists
------------------------------------------------
for i=1,k do
    local offset = tonumber(ARGV[i])
    if redis.call("GETBIT", KEYS[1], offset) == 0 then
        return -1   -- not exposed
    end
end

------------------------------------------------
-- 2. check already converted (dedup)
------------------------------------------------
local exists = 1
for i=1,k do
    local offset = tonumber(ARGV[i])
    if redis.call("GETBIT", KEYS[2], offset) == 0 then
        exists = 0
        break
    end
end

if exists == 1 then
    return 0 -- already converted
end

------------------------------------------------
-- 3. mark conversion
------------------------------------------------
for i=1,k do
    local offset = tonumber(ARGV[i])
    redis.call("SETBIT", KEYS[2], offset, 1)
end

------------------------------------------------
-- 4. increment analytics
------------------------------------------------
redis.call("HINCRBY", KEYS[3], variant, 1)

------------------------------------------------
-- 5. lazy TTL
------------------------------------------------
if redis.call("TTL", KEYS[2]) == -1 then
    redis.call("EXPIRE", KEYS[2], ttl)
end

if redis.call("TTL", KEYS[3]) == -1 then
    redis.call("EXPIRE", KEYS[3], ttl)
end

return 1
`)

func (u *UserConversion) ConversionUser(ctx context.Context, redisClient *redis.Client) error {
	if redisClient == nil {
		return errors.New("redisClient is nil")
	}

	// Fetch Expire
	expiredTime, err := redisClient.Get(ctx, keyGeneratorExpiredTime(u.ExperimentID)).Result()
	if err != nil && err != redis.Nil {
		return err
	}
	if err == redis.Nil {
		return errors.New("experiment expiredTime is not set")
	}

	unixInt, err := strconv.ParseInt(expiredTime, 10, 64)
	if err != nil {
		return err
	}

	ttl := int(unixInt - time.Now().UTC().Unix())
	if ttl <= 0 {
		return errors.New("experiment expiredTime is expired")
	}

	// Set conversion lock
	keyConversionLock := keyGeneratorUserConversionLock(u.ExperimentID)
	keyConversionAnalytics := keyGeneratorUserConversionAnalytics(u.ExperimentID)
	keyExposureLock := keyGeneratorUserExposureLock(u.ExperimentID)
	indexes := bloomIndexes(u.ExperimentID, u.UserID, BloomIndexK, BloomIndexM)

	args := make([]interface{}, 0, len(indexes)+2)

	for _, idx := range indexes {
		args = append(args, idx)
	}

	args = append(args, u.VariantID)
	args = append(args, ttl)

	res, err := convertUserLua.Run(
		ctx,
		redisClient,
		[]string{
			keyExposureLock,
			keyConversionLock,
			keyConversionAnalytics,
		},
		args...,
	).Int()
	if err != nil {
		return err
	}

	switch res {
	case -1:
		return errors.New("user not exposed")
	case 0:
		return nil // already converted (idempotent)
	case 1:
		return nil // success
	default:
		return errors.New("unknown conversion result")
	}
}
