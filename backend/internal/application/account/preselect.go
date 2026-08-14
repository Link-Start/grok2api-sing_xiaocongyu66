package account

// DefaultPreselectValidateCount is the minimum sample size for preselected account probes.
// When fewer enabled accounts remain, all of them are tested.
const DefaultPreselectValidateCount = 5

// samplePreselectIDs returns the first limit IDs (priority order). If the pool is
// smaller than limit, all IDs are returned.
func samplePreselectIDs(ids []uint64, limit int) []uint64 {
	if limit <= 0 {
		limit = DefaultPreselectValidateCount
	}
	if len(ids) == 0 {
		return nil
	}
	if len(ids) < limit {
		limit = len(ids)
	}
	return append([]uint64(nil), ids[:limit]...)
}
