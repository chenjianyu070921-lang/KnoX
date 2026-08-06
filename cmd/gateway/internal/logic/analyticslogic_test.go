package logic

import "testing"

func TestNormalizeAnalyticsDays(t *testing.T) {
	cases := []struct {
		in   int
		want int
	}{
		{0, 7},
		{-5, 7},
		{7, 7},
		{90, 90},
		{91, 90},
	}
	for _, c := range cases {
		if got := normalizeAnalyticsDays(c.in); got != c.want {
			t.Fatalf("normalizeAnalyticsDays(%d) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestNormalizeSlowQueryLimit(t *testing.T) {
	cases := []struct {
		in   int
		want int
	}{
		{0, 20},
		{-1, 20},
		{20, 20},
		{100, 100},
		{101, 100},
	}
	for _, c := range cases {
		if got := normalizeSlowQueryLimit(c.in); got != c.want {
			t.Fatalf("normalizeSlowQueryLimit(%d) = %d, want %d", c.in, got, c.want)
		}
	}
}
