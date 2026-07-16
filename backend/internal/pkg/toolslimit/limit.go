// Package toolslimit provides a process-wide dynamic ceiling for request tools arrays.
//
// xAI / Grok Build hard-rejects more than HardMax tools. This package also
// learns a softer ceiling from recent traffic: every Interval it samples recent
// observed tool counts and lowers the effective limit so oversized agent payloads
// fail fast instead of consuming upstream capacity.
package toolslimit

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// HardMax is the absolute upstream ceiling (xAI / Grok Build).
	HardMax = 250
	// MinFloor keeps the dynamic limit from collapsing on tiny probe requests.
	MinFloor = 32
	// Interval is how often the dynamic limit is recomputed.
	Interval = 5 * time.Minute
	// sampleCap is the ring size of recent tool-count observations.
	sampleCap = 64
	// headroom is added to the picked recent sample so slightly larger sessions still pass.
	headroom = 8
)

var (
	current atomic.Int32

	mu      sync.Mutex
	samples []int
	nextIdx int
	filled  int
)

func init() {
	current.Store(HardMax)
	samples = make([]int, sampleCap)
}

// Current returns the effective tools limit (≤ HardMax).
func Current() int {
	n := int(current.Load())
	if n < MinFloor {
		return MinFloor
	}
	if n > HardMax {
		return HardMax
	}
	return n
}

// Observe records a request's tools array length (call for any non-empty tools list).
func Observe(count int) {
	if count < 0 {
		return
	}
	mu.Lock()
	samples[nextIdx] = count
	nextIdx = (nextIdx + 1) % sampleCap
	if filled < sampleCap {
		filled++
	}
	mu.Unlock()
}

// Check returns an error when count exceeds the effective limit.
// Successful (or attempted) counts should still be Observe()'d by callers.
func Check(count int) error {
	if count <= 0 {
		return nil
	}
	limit := Current()
	if count > limit {
		return fmt.Errorf("tools 数量超过动态上限：提供了 %d 个，当前上限 %d（硬顶 %d；每 %s 根据近期请求重算）",
			count, limit, HardMax, Interval)
	}
	return nil
}

// RecomputeOnce picks one recent observation and updates the limit.
// Exposed for tests; production uses Run.
func RecomputeOnce(rng *rand.Rand) (picked, next int, ok bool) {
	mu.Lock()
	defer mu.Unlock()
	if filled == 0 {
		return 0, Current(), false
	}
	// Pick one recent sample (uniform over the ring contents).
	idx := 0
	if rng != nil {
		idx = rng.Intn(filled)
	} else {
		idx = rand.Intn(filled)
	}
	// Map into ring: oldest is nextIdx when full, else 0.
	start := 0
	if filled == sampleCap {
		start = nextIdx
	}
	picked = samples[(start+idx)%sampleCap]

	// Dynamic limit = sample + headroom, clamped to [MinFloor, HardMax].
	next = picked + headroom
	if next < MinFloor {
		next = MinFloor
	}
	if next > HardMax {
		next = HardMax
	}
	current.Store(int32(next))
	return picked, next, true
}

// Run recomputes the dynamic limit every Interval until ctx is done.
func Run(ctx context.Context, log func(string, ...any)) {
	if log == nil {
		log = func(string, ...any) {}
	}
	ticker := time.NewTicker(Interval)
	defer ticker.Stop()
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			picked, next, ok := RecomputeOnce(rng)
			if !ok {
				log("tools_limit_recompute_skipped", "reason", "no_samples", "limit", Current())
				continue
			}
			log("tools_limit_recomputed", "picked_tools", picked, "limit", next, "hard_max", HardMax)
		}
	}
}

// ResetForTest restores HardMax and clears samples (tests only).
func ResetForTest() {
	mu.Lock()
	defer mu.Unlock()
	current.Store(HardMax)
	nextIdx = 0
	filled = 0
	for i := range samples {
		samples[i] = 0
	}
}
