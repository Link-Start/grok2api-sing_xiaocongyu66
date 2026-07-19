package account

import (
	"testing"
	"time"

	accountdomain "github.com/chenyme/grok2api/backend/internal/domain/account"
)

func TestPreserveActiveQuotaWindowsUntilReset(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(time.Hour)
	past := now.Add(-time.Second)
	incoming := []accountdomain.QuotaWindow{{Mode: "console", Remaining: 20, Total: 20}}

	active := preserveActiveQuotaWindows([]accountdomain.QuotaWindow{{Mode: "console", Remaining: 7, Total: 20, ResetAt: &future}}, incoming, now)
	if len(active) != 1 || active[0].Remaining != 7 {
		t.Fatalf("active window = %#v", active)
	}

	// Delayed rotation: partially consumed console windows keep local remaining even
	// before the recovery timer starts (ResetAt is still nil).
	partial := preserveActiveQuotaWindows([]accountdomain.QuotaWindow{{Mode: "console", Remaining: 15, Total: 20}}, incoming, now)
	if len(partial) != 1 || partial[0].Remaining != 15 || partial[0].ResetAt != nil {
		t.Fatalf("partial delayed window = %#v", partial)
	}

	expired := preserveActiveQuotaWindows([]accountdomain.QuotaWindow{{Mode: "console", Remaining: 0, Total: 20, ResetAt: &past}}, incoming, now)
	if len(expired) != 1 || expired[0].Remaining != 20 {
		t.Fatalf("expired window = %#v", expired)
	}
}
