package scanners

import (
	"math"
	"testing"
)

func TestUncompressedSizeWouldExceed(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name       string
		total      int64
		size       uint64
		max        int64
		wantExceed bool
	}{
		{"within limit", 100, 50, 200, false},
		{"exactly at limit", 100, 100, 200, false},
		{"one over remaining", 100, 101, 200, true},
		{"size alone exceeds max", 0, 300, 200, true},
		{"max int64 overflow size", 0, uint64(math.MaxInt64) + 1, math.MaxInt64, true},
		{"already at max total", 200, 1, 200, true},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := uncompressedSizeWouldExceed(tc.total, tc.size, tc.max)
			if got != tc.wantExceed {
				t.Fatalf("uncompressedSizeWouldExceed(%d, %d, %d) = %v, want %v",
					tc.total, tc.size, tc.max, got, tc.wantExceed)
			}
		})
	}
}
