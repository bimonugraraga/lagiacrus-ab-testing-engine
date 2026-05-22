package abtestingengine

import (
	"fmt"
	"hash/fnv"
)

func bloomIndexes(experimentID, userID any, k int, m uint64) []uint64 {
	base := hash64(fmt.Sprintf("%v:%v", experimentID, userID))
	fmt.Println(">>>>>>>>>>>", base)

	indexes := make([]uint64, k)

	for i := 0; i < k; i++ {
		h := base + uint64(i)*0x9e3779b97f4a7c15 // golden ratio mixing
		indexes[i] = h % m
	}

	return indexes
}

func hash64(s string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(s))
	return h.Sum64()
}
