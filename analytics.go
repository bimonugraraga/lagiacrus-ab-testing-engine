package abtestingengine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

func (e *Experiment) GetAnalytics(ctx context.Context, redisClient *redis.Client) (ResultAnalytics, error) {
	if redisClient == nil {
		return ResultAnalytics{}, errors.New("redis client is nil")
	}

	// Generate Key
	keyConversion := keyGeneratorUserConversionAnalytics(e.ID)
	keyExposure := keyGeneratorUserExposureAnalytics(e.ID)
	keyExperimentDetail := keyGeneratorExperimentDetail(e.ID)
	keyExpiredTime := keyGeneratorExpiredTime(e.ID)

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
	err = json.Unmarshal([]byte(detail), e)
	if err != nil {
		return ResultAnalytics{}, err
	}

	// Conversion
	conversionResult := make([]map[any]any, len(e.Variants))
	for i, variant := range e.Variants {
		conversionResult[i] = make(map[any]any)
		conversionResult[i]["id"] = variant.ID
		conversionResult[i]["conversion"], _ = strconv.Atoi(conversion[variant.ID])
	}

	// Exposure
	exposureResult := make([]map[any]any, len(e.Variants))
	for i, variant := range e.Variants {
		exposureResult[i] = make(map[any]any)
		exposureResult[i]["id"] = variant.ID
		exposureResult[i]["exposure"], _ = strconv.Atoi(exposure[variant.ID])
	}

	// Conversion Rate
	conversionRate := make([]map[any]any, len(e.Variants))
	for i, variant := range e.Variants {
		conversionRate[i] = make(map[any]any)
		conversionRate[i]["id"] = variant.ID
		conv := float64(conversionResult[i]["conversion"].(int))
		expo := float64(exposureResult[i]["exposure"].(int))
		if expo == 0 {
			conversionRate[i]["conversion_rate"] = "0.0000"
			continue
		}
		conversionRate[i]["conversion_rate"] = fmt.Sprintf("%.4f", conv/expo)
	}

	res := ResultAnalytics{
		ExperimentID:   e.ID,
		Variants:       e.Variants,
		Conversion:     conversionResult,
		Exposure:       exposureResult,
		ConversionRate: conversionRate,
		ExpiredOnLocal: localStr,
		ExpiredOnUTC:   utcStr,
	}

	return res, nil
}
