package main

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	abtestingengine "github.com/bimonugraraga/lagiacrus-ab-testing-engine/ab_testing_engine"
	"github.com/redis/go-redis/v9"
)

func main() {
	exp := abtestingengine.Experiment{
		ID: "123",
		Variants: []abtestingengine.Variant{
			abtestingengine.Variant{
				ID:     "1",
				Weight: 20,
			},
			abtestingengine.Variant{
				ID:     "2",
				Weight: 70,
			},
			abtestingengine.Variant{
				ID:     "3",
				Weight: 10,
			},
		},
	}

	ctx := context.Background()
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	_, _ = exp.GetAnalytics(ctx, rdb)

	// if err := rdb.Ping(ctx).Err(); err != nil {
	// 	log.Fatal("redis not running:", err)
	// }

	// Assign(ctx, rdb, exp)
}

func Assign(ctx context.Context, rdb *redis.Client, exp abtestingengine.Experiment) {
	temp := make(map[string]int)
	tempUser := []int64{}
	for i := 0; i < 1000; i++ {
		n := int64(rand.IntN(1_000_000))
		tempUser = append(tempUser, n)
		variant, err := exp.AssignUser(ctx, fmt.Sprintf("%d", n), rdb, time.Now().UTC().AddDate(0, 0, 10))
		if err != nil {
			temp["0"]++
			continue
		}
		// // fmt.Println(n, variant.ID)
		u := abtestingengine.UserConversion{
			UserID:       n,
			ExperimentID: 123,
			VariantID:    variant.ID,
		}
		u.ExposedUser(ctx, rdb)
		if randomBool() {
			u.ConversionUser(ctx, rdb)
		}
		temp[variant.ID]++
	}

	fmt.Println(temp)
	fmt.Println("======================")
}

func randomBool() bool {
	return rand.IntN(2) == 1
}
