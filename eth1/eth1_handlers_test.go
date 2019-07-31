package eth1

import (
	"testing"
	"time"
)

func TestConstructBlocksByNumber(t *testing.T) {
	num := uint64(5)
	blockTime := time.Second
	res := ConstructBlocksByNumber(num, blockTime)
	numKeys := uint64(0)
	currBlock := res[num]
	for k, v := range res {
		// We ensure block times increase monotonically from
		// block number 0 to 5.
		if v.Time > currBlock.Time {
			t.Error("Expected block times to be increasing")
		}
		numKeys++
	}
	if numKeys != num {
		t.Errorf("Expected %d keys, received %d", num, numKeys)
	}
}
