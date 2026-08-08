package fuzzy

import (
	"math"
	"testing"
)

func approx(a, b float64) bool { return math.Abs(a-b) < 0.05 }

func TestScoreExactAndNear(t *testing.T) {
	cases := []struct {
		name    string
		q       string
		c       string
		minWant float64 // lower bound for a sensible match
		strong  bool    // expect a strong (>=0.8) match
	}{
		{"exact", "sparkyy", "sparkyy", 1.0, true},
		{"case insensitive", "SPARKYY", "sparkyy", 1.0, true},
		{"prefix", "spark", "sparkyy", 0.85, true},
		{"substring", "park", "sparkyy", 0.7, false},
		{"scattered subseq", "spyy", "sparkyy", 0.4, false},
		{"edit distance 1", "sparkyj", "sparkyy", 0.65, false},
		{"token match", "ttv", "Sparkyy TTV LIVE", 1.0, true},
		{"token compact match", "sparkyttv", "Sparkyy TTV LIVE", 0.5, false},
		{"unrelated", "xyzqw", "sparkyy", 0, false},
		{"empty query", "", "sparkyy", 0, false},
		{"empty candidate", "spark", "", 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Score(tc.q, tc.c)
			t.Logf("q=%q c=%q -> %.3f (min %.2f, strong=%v)", tc.q, tc.c, got, tc.minWant, tc.strong)
			if tc.strong && got < 0.8 {
				t.Fatalf("expected strong match (>=0.8), got %.3f", got)
			}
			if got < tc.minWant-0.05 {
				t.Fatalf("expected >= %.2f, got %.3f", tc.minWant, got)
			}
			if tc.name == "unrelated" && got > Threshold {
				t.Fatalf("unrelated scored %.3f above threshold %.2f", got, Threshold)
			}
		})
	}
}

func TestBestScoreAcrossCandidates(t *testing.T) {
	cands := []string{"Sparkyy TTV [LIVE]", "Nea", "Dwight", "12345"}
	got := BestScore("spark", cands)
	if got < 0.85 {
		t.Fatalf("expected best score to match 'Sparkyy', got %.3f", got)
	}
	_ = approx
}

func TestThresholdReturnsMany(t *testing.T) {
	// A loose query should match several candidates above the default threshold.
	cands := []string{"sparkyy", "spqrk", "sark", "sparkle", "darkspark", "nomatch"}
	matched := 0
	for _, c := range cands {
		if Score("spark", c) >= Threshold {
			matched++
		}
	}
	if matched < 4 {
		t.Fatalf("expected loose matcher to return many results, got %d/%d", matched, len(cands))
	}
}
