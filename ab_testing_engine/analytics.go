package abtestingengine

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

func (a *Analytics) GetAnalytics(ctx context.Context, redisClient *redis.Client) (ResultAnalytics, error) {
	if redisClient == nil {
		return ResultAnalytics{}, errors.New("redis client is nil")
	}

	// Generate Key
	keyConversion := keyGeneratorUserConversionAnalytics(a.ExperimentID)
	keyExposure := keyGeneratorUserExposureAnalytics(a.ExperimentID)
	keyExperimentDetail := keyGeneratorExpiredTime(a.ExperimentID)
	keyExpiredTime := keyGeneratorExpiredTime(a.ExperimentID)

	// Get Conversion
	conversion, err := redisClient.HGetAll(ctx, keyConversion).Result()
	if err != nil {
		return ResultAnalytics{}, err
	}

	// Get Exposure
	exposure, err := redisClient.HGetAll(ctx, keyExposure).Result()
	if err != nil {
		return ResultAnalytics{}, err
	}

	// Get Detail and TTL
	ttl, err := redisClient.Get(ctx, keyExpiredTime).Result()
	if err != nil {
		return ResultAnalytics{}, err
	}
	detail, err := redisClient.Get(ctx, keyExperimentDetail).Result()
	if err != nil {
		return ResultAnalytics{}, err
	}

	// Parse TTL
	ttlInt, _ := strconv.Atoi(ttl)
	tUTC := time.Unix(int64(ttlInt), 0).UTC()
	tLocal := tUTC.Local()

	utcStr := tUTC.Format("2006-01-02 15:04")
	localStr := tLocal.Format("2006-01-02 15:04")

	// Parse Detail
	variants := detail.([]Variant)

	fmt.Println(conversion, exposure, ttl, detail)

	res := ResultAnalytics{
		ExperimentID: a.ExperimentID,

		ExpiredOnLocal: localStr,
		ExpiredOnUTC:   utcStr,
	}

	return res, nil
}
