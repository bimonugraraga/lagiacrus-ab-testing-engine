package abtestingengine

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

var exposeUserLua = redis.NewScript(`
-- check bloom bits (dedup)
local exists = 1
for i=1,#ARGV-2 do
    if redis.call("GETBIT", KEYS[1], ARGV[i]) == 0 then
        exists = 0
        break
    end
end

-- already exposed
if exists == 1 then
    return 0
end

-- lock exposure (set bloom bits)
for i=1,#ARGV-2 do
    redis.call("SETBIT", KEYS[1], ARGV[i], 1)
end

local variant = ARGV[#ARGV-1]
local ttl = ARGV[#ARGV]

-- increment analytics
redis.call("HINCRBY", KEYS[2], variant, 1)

-- lazy TTL
if redis.call("TTL", KEYS[1]) == -1 then
    redis.call("EXPIRE", KEYS[1], ttl)
end

if redis.call("TTL", KEYS[2]) == -1 then
    redis.call("EXPIRE", KEYS[2], ttl)
end

return 1
`)

func (u *UserConversion) ExposedUser(ctx context.Context, redisClient *redis.Client) error {
	if redisClient == nil {
		return errors.New("redisClient is nil")
	}

	// FetchExpiredTime
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

	// ExposeUser
	keyUserExposureLock := keyGeneratorUserExposureLock(u.ExperimentID)
	keyUserExposureAnalytics := keyGeneratorUserExposureAnalytics(u.ExperimentID)

	indexes := bloomIndexes(u.ExperimentID, u.UserID, BloomIndexK, BloomIndexM)
	fmt.Println(indexes)
	args := make([]interface{}, 0, len(indexes)+2)

	for _, idx := range indexes {
		args = append(args, idx)
	}

	args = append(args, u.VariantID)
	args = append(args, ttl)
	_, err = exposeUserLua.Run(
		ctx,
		redisClient,
		[]string{
			keyUserExposureLock,
			keyUserExposureAnalytics,
		},
		args...,
	).Result()
	if err != nil {
		fmt.Println(">>>>>>>>>>>", err)
		return err
	}
	return nil
}
