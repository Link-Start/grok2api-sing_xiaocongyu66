package toolslimit

import (
	"math/rand"
	"testing"
)

func TestCheckRespectsDynamicLimit(t *testing.T) {
	ResetForTest()
	defer ResetForTest()

	if err := Check(100); err != nil {
		t.Fatalf("under hard max: %v", err)
	}
	if err := Check(251); err == nil {
		t.Fatal("expected hard rejection above 250")
	}

	// Feed small samples and recompute → limit drops.
	for i := 0; i < 10; i++ {
		Observe(40)
	}
	rng := rand.New(rand.NewSource(1))
	picked, next, ok := RecomputeOnce(rng)
	if !ok || picked != 40 {
		t.Fatalf("picked=%d next=%d ok=%v", picked, next, ok)
	}
	want := 40 + headroom
	if next != want {
		t.Fatalf("next=%d want %d", next, want)
	}
	if err := Check(want); err != nil {
		t.Fatalf("at dynamic limit: %v", err)
	}
	if err := Check(want + 1); err == nil {
		t.Fatal("expected rejection above dynamic limit")
	}
}

func TestRecomputeClampsToHardMax(t *testing.T) {
	ResetForTest()
	defer ResetForTest()
	Observe(250)
	_, next, ok := RecomputeOnce(rand.New(rand.NewSource(2)))
	if !ok || next != HardMax {
		t.Fatalf("next=%d ok=%v", next, ok)
	}
}

func TestRecomputeFloor(t *testing.T) {
	ResetForTest()
	defer ResetForTest()
	Observe(1)
	_, next, ok := RecomputeOnce(rand.New(rand.NewSource(3)))
	if !ok || next != MinFloor {
		t.Fatalf("next=%d want floor %d", next, MinFloor)
	}
}
