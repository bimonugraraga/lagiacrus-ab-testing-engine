package abtestingengine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

/*
AssignUser assigns user to variant based on weight of each variant.
Expired time is the time when the assignment will be expired (UTC) (2026-04-06 04:45:00 +0000 UTC).
Default is 30 days.
*/
func (e *Experiment) AssignUser(ctx context.Context, userID string, redisClient *redis.Client, expiredTime time.Time) (Variant, error) {
	if len(e.Variants) == 0 {
		return Variant{}, errors.New("No Variant Exist")
	}

	total := 0
	for _, val := range e.Variants {
		total += val.Weight
	}
	if total != 100 {
		return Variant{}, errors.New("Weight total must 100")
	}

	hash := hash(userID, e.ID)
	bucket := int(hash % 100)

	cumulative := 0
	for _, v := range e.Variants {
		cumulative += v.Weight

		if bucket < cumulative {
			// Set Experiment Detail to Redis
			if redisClient != nil {
				err := e.setRedisExperiment(ctx, redisClient, expiredTime)
				if err != nil {
					return Variant{}, err
				}
			}
			return v, nil
		}
	}

	// fallback safety
	return e.Variants[len(e.Variants)-1], nil
}

/*
AssignUserBulk assigns user to variant in batch.
batchSize will determine how many user that get handled by 1 go routine.
numWorkers will determine the number of go routine that get handled at the same time.
*/
func (e *Experiment) AssignUserBulk(ctx context.Context, userIDs []string, batchSize int, numWorkers int, redisClient *redis.Client, expiredTime time.Time) (ResultAssignBulk, error) {
	// Validation
	if len(userIDs) == 0 {
		return ResultAssignBulk{}, errors.New("UserIDs is empty")
	}
	if batchSize <= 0 {
		return ResultAssignBulk{}, errors.New("BatchSize must be greater than 0")
	}
	if numWorkers <= 0 {
		return ResultAssignBulk{}, errors.New("NumWorkers must be greater than 0")
	}
	total := 0
	for _, val := range e.Variants {
		total += val.Weight
	}
	if total != 100 {
		return ResultAssignBulk{}, errors.New("Weight total must 100")
	}

	// Init Result
	type Result struct {
		Results map[string]ResultBulk
		mu      sync.Mutex
	}
	var (
		currentWorkers = 0
		results        Result
		wg             sync.WaitGroup
	)

	results.Results = make(map[string]ResultBulk)
	for _, val := range e.Variants {
		results.Results[val.ID] = ResultBulk{
			Variant: val,
		}
	}

	// Assign User to variant per batch
	isLastBatch := false
	for i := 0; i < len(userIDs); i += batchSize {
		end := i + batchSize
		if end >= len(userIDs) {
			end = len(userIDs)
			isLastBatch = true
		}

		batch := userIDs[i:end]
		wg.Add(1)

		go func(ctx context.Context, batch []string, results *Result) {
			for _, userID := range batch {
				variant, err := e.AssignUser(ctx, userID, nil, time.Time{})
				if err != nil {
					continue
				}

				results.mu.Lock()

				tmp := results.Results[variant.ID]
				tmp.UserID = append(tmp.UserID, userID)
				results.Results[variant.ID] = tmp

				results.mu.Unlock()
			}

			wg.Done()
		}(ctx, batch, &results)
		currentWorkers++

		if currentWorkers == numWorkers || isLastBatch {
			wg.Wait()
			currentWorkers = 0
		}

	}

	if redisClient != nil {
		err := e.setRedisExperiment(ctx, redisClient, expiredTime)
		if err != nil {
			return ResultAssignBulk{}, err
		}
	}
	return ResultAssignBulk{
		ResultsBulk: results.Results,
	}, nil
}

func hash(userID, experimentID string) uint32 {
	h := fnv.New32()
	key := fmt.Sprintf("%s:%s", userID, experimentID)
	h.Write([]byte(key))

	return h.Sum32()
}

func (e *Experiment) setRedisExperiment(ctx context.Context, redisClient *redis.Client, expiredTime time.Time) error {
	if redisClient == nil {
		return errors.New("Redis client is nil")
	}
	if expiredTime.IsZero() {
		expiredTime = time.Now().UTC().AddDate(0, 0, 30)
	}
	exp := expiredTime.Unix()
	now := time.Now().UTC().Unix()
	ttl := exp - now
	if ttl <= 0 {
		return errors.New("Expired time must be greater than current time")
	}

	// Set Expired Time
	keyExpiredTime := keyGeneratorExpiredTime(e.ID)
	err := redisClient.SetNX(
		ctx,
		keyExpiredTime,
		expiredTime.Unix(),
		time.Duration(ttl)*time.Second,
	).Err()
	if err != nil {
		return err
	}

	// Set Experiment Detail
	keyExperimentDetail := keyGeneratorExperimentDetail(e.ID)
	loadByte, _ := json.Marshal(e)
	err = redisClient.SetNX(
		ctx,
		keyExperimentDetail,
		loadByte,
		time.Duration(ttl)*time.Second,
	).Err()
	if err != nil {
		return err
	}

	return nil
}
