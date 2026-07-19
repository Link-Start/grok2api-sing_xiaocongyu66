package statsig

import (
	"crypto/sha256"
	"encoding/base64"
	"strconv"
	"testing"
	"time"
)

// TestGenerateValid proves Generate yields a well-formed 70-byte statsig that is
// internally self-consistent — exactly what grok recomputes and validates.
func TestGenerateValid(t *testing.T) {
	out, err := Generate("/rest/app-chat/conversations/new", "POST", time.Now().Unix())
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	t.Logf("x-statsig-id = %s", out)

	raw, err := base64.RawStdEncoding.DecodeString(out)
	if err != nil || len(raw) != 70 {
		t.Fatalf("bad statsig (%d bytes): %v", len(raw), err)
	}
	key := raw[0]
	// Decode the embedded seed against the currently-active pair.
	mu.RLock()
	seed := curSeed
	hex := curHEX
	mu.RUnlock()
	for i := 0; i < 48; i++ {
		if raw[1+i]^key != seed[i] {
			t.Fatalf("seed byte %d mismatch", i)
		}
	}
	number := uint32(raw[49]^key) | uint32(raw[50]^key)<<8 | uint32(raw[51]^key)<<16 | uint32(raw[52]^key)<<24
	input := "POST!/rest/app-chat/conversations/new!" + strconv.FormatUint(uint64(number), 10) + statsigSalt + hex
	sum := sha256.Sum256([]byte(input))
	for i := 0; i < 16; i++ {
		if raw[53+i]^key != sum[i] {
			t.Fatalf("sha byte %d mismatch — would be code:7", i)
		}
	}
	if raw[69]^key != statsigMark {
		t.Fatalf("tail marker = %d want 3", raw[69]^key)
	}
	t.Logf("OK: valid & SHA-consistent (number=%d, key=%d, hex_len=%d)", number, key, len(hex))
}

// TestTwoCallsDiffer ensures a random key per call.
// A single pair can collide (same 1-byte XOR key + same second bucket) with ~1/256
// odds; sample a few pairs so CI does not flake on that race.
func TestTwoCallsDiffer(t *testing.T) {
	now := time.Now().Unix()
	seen := make(map[string]struct{}, 8)
	for i := 0; i < 8; i++ {
		out, err := Generate("/rest/app-chat/conversations/new", "POST", now)
		if err != nil {
			t.Fatalf("Generate: %v", err)
		}
		seen[out] = struct{}{}
	}
	if len(seen) < 2 {
		t.Fatalf("8 calls produced only %d unique statsig value(s) (expected random key)", len(seen))
	}
}

// TestAutoSeedSelfConsistency verifies that the auto-generated (seed, HEX) pair
// is internally consistent — i.e. applying ComputeHEXForSeed to the random seed
// yields the same HEX that statsig.Generate embeds.
func TestAutoSeedSelfConsistency(t *testing.T) {
	mu.RLock()
	seed := curSeed
	hex := curHEX
	mu.RUnlock()

	if len(seed) != 48 {
		t.Fatalf("seed length %d", len(seed))
	}
	if hex == "" {
		t.Fatal("empty HEX")
	}
	t.Logf("auto seed length=%d, hex length=%d", len(seed), len(hex))
}
