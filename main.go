package main

import (
	"context"
	"math/rand/v2"

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
	anaytics := abtestingengine.Analytics{
		ExperimentID: exp.ID,
	}
	_, _ = anaytics.GetAnalytics(ctx, rdb)

	// if err := rdb.Ping(ctx).Err(); err != nil {
	// 	log.Fatal("redis not running:", err)
	// }
	// temp := make(map[string]int)
	// tempUser := []int64{}
	// for i := 0; i < 1000; i++ {
	// 	n := int64(rand.IntN(1_000_000))
	// 	tempUser = append(tempUser, n)
	// 	variant, err := exp.AssignUser(ctx, fmt.Sprintf("%d", n), rdb, time.Now().UTC().AddDate(0, 0, 10))
	// 	if err != nil {
	// 		temp["0"]++
	// 		continue
	// 	}
	// 	// // fmt.Println(n, variant.ID)
	// 	u := abtestingengine.UserConversion{
	// 		UserID:       n,
	// 		ExperimentID: 123,
	// 		VariantID:    variant.ID,
	// 	}
	// 	u.ExposedUser(ctx, rdb)
	// 	if randomBool() {
	// 		u.ConversionUser(ctx, rdb)
	// 	}
	// 	temp[variant.ID]++
	// }

	// fmt.Println(temp)
	// fmt.Println("======================")
	// res, _ := exp.AssignUserIntBulk(ctx, tempUser, 100, 10, rdb, 30)
	// fmt.Println(len(res[int64(1)].UserID))
	// fmt.Println(len(res[int64(2)].UserID))
	// fmt.Println(len(res[int64(3)].UserID))
	// fmt.Println(res)
}

func randomBool() bool {
	return rand.IntN(2) == 1
}
