// Package fuzzy implements an intentionally loose, composite string matcher
// used to rank stream samples by how well their OCR'd survivor names match a
// free-form query. It blends three signals so that exact and near-exact matches
// score ~1.0 while only loosely related strings still surface above the
// threshold — the matcher is tuned to "return many results":
//
//   - trigram (3-gram) Jaccard similarity: resilient to letter swaps and small
//     OCR recognition errors.
//   - normalized Levenshtein ratio (1 - dist/maxLen): rewards overall shape
//     similarity.
//   - subsequence / substring bonus: a strong, length-invariant signal that
//     fires even when the query is scattered through the candidate (e.g. a
//     caller typing initials or a fragment of a long username).
//
// Multi-word survivor strings ("Sparkyy TTV [LIVE]") are scored against the
// whole, the space-stripped compact form, and each whitespace token; the best
// per-candidate score wins. BestScore takes the max across candidates.
package fuzzy

import (
	"strings"
	"unicode"
)

// Threshold is the default cutoff above which a candidate is considered a
// (loose) match. Tuned to reject random noise while still passing genuine
// OCR near-misses (subsequence matches start at 0.35).
const Threshold = 0.60

// Score returns a 0..1 fuzzy similarity of query against candidate. An empty
// query or candidate scores 0.
func Score(query, candidate string) float64 {
	qn := norm(query)
	if qn == "" {
		return 0
	}
	cn := norm(candidate)
	if cn == "" {
		return 0
	}
	qc := compact(qn)
	cc := compact(cn)

	best := 0.0
	// Compare the query (both spaced and compact forms) against the candidate's
	// whole, compact and per-token forms; keep the strongest signal.
	targets := targetsFor(cn, cc)
	for _, q := range []string{qn, qc} {
		for _, t := range targets {
			if s := signal(q, t); s > best {
				best = s
			}
		}
	}
	return best
}

// BestScore returns the highest Score(query, c) over the candidates. Empty list
// scores 0.
func BestScore(query string, candidates []string) float64 {
	best := 0.0
	for _, c := range candidates {
		if s := Score(query, c); s > best {
			best = s
		}
	}
	return best
}

// signal combines the match-type signal and the shape-similarity signals for a
// single (already normalized) query/target pair into a 0..1 value. The match
// type drives ranking (exact > prefix > substring > scattered subseq > shape
// only); shape similarity (trigram + Levenshtein) refines within each band and
// rescues partial matches that have no subsequence relationship at all.
func signal(q, t string) float64 {
	if q == "" || t == "" {
		return 0
	}
	lev := levenshteinRatio(q, t)
	tri := trigramSimilarity(q, t)
	shape := 0.5*tri + 0.5*lev
	qr := []rune(q)
	tr := []rune(t)

	switch {
	case q == t:
		return 1.0
	case strings.HasPrefix(t, q):
		// Prefix: very strong; longer query fraction => closer to 1.
		return clamp(0.6 + 0.4*float64(len(qr))/float64(len(tr)))
	case strings.Contains(t, q):
		// Contiguous substring: strong.
		return clamp(0.55 + 0.35*float64(len(qr))/float64(len(tr)))
	case isSubsequence(q, t):
		// Scattered subsequence: moderate, lifted by overall shape similarity.
		return clamp(0.35 + 0.4*shape)
	default:
		// No subsequence relation: rely on shape only. Unrelated strings fall
		// below the threshold here; genuine near-misses still score well.
		return shape
	}
}

func clamp(v float64) float64 {
	if v > 1 {
		return 1
	}
	if v < 0 {
		return 0
	}
	return v
}

// isSubsequence reports whether q's runes appear in t in order (not necessarily
// contiguously).
func isSubsequence(q, t string) bool {
	rq := []rune(q)
	rt := []rune(t)
	i := 0
	for j := 0; i < len(rq) && j < len(rt); j++ {
		if rq[i] == rt[j] {
			i++
		}
	}
	return i == len(rq)
}

// levenshteinRatio is 1 - editDistance/maxLen, clamped to [0,1].
func levenshteinRatio(a, b string) float64 {
	ra := []rune(a)
	rb := []rune(b)
	if len(ra) == 0 && len(rb) == 0 {
		return 1
	}
	d := editDistance(ra, rb)
	max := len(ra)
	if len(rb) > max {
		max = len(rb)
	}
	if max == 0 {
		return 0
	}
	r := 1 - float64(d)/float64(max)
	if r < 0 {
		r = 0
	}
	return r
}

func editDistance(a, b []rune) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}
	prev := make([]int, len(b)+1)
	cur := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			del := prev[j] + 1
			ins := cur[j-1] + 1
			sub := prev[j-1] + cost
			m := del
			if ins < m {
				m = ins
			}
			if sub < m {
				m = sub
			}
			cur[j] = m
		}
		prev, cur = cur, prev
	}
	return prev[len(b)]
}

// trigramSimilarity is the Jaccard similarity of the two strings' padded
// 3-gram sets. For strings shorter than 3 runes it falls back to a simple
// character-overlap ratio so very short usernames still score sensibly.
func trigramSimilarity(a, b string) float64 {
	ra := []rune(a)
	rb := []rune(b)
	if len(ra) < 3 || len(rb) < 3 {
		return charOverlap(ra, rb)
	}
	sa := trigramSet(a)
	sb := trigramSet(b)
	if len(sa) == 0 || len(sb) == 0 {
		return 0
	}
	inter := 0
	for g := range sa {
		if sb[g] {
			inter++
		}
	}
	union := len(sa) + len(sb) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

// trigramSet builds the set of length-3 substrings of a space-padded string.
func trigramSet(s string) map[string]bool {
	padded := " " + s + " "
	r := []rune(padded)
	out := make(map[string]bool, len(r))
	for i := 0; i+3 <= len(r); i++ {
		out[string(r[i:i+3])] = true
	}
	return out
}

// charOverlap is a fallback similarity for very short strings: 2*|∩| / (|a|+|b|).
func charOverlap(a, b []rune) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 1
	}
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	sb := make(map[rune]bool, len(b))
	for _, c := range b {
		sb[c] = true
	}
	inter := 0
	for _, c := range a {
		if sb[c] {
			inter++
			sb[c] = false
		}
	}
	return 2 * float64(inter) / float64(len(a)+len(b))
}

// targetsFor returns the set of normalized comparison targets derived from a
// candidate: its compact (space-stripped) form, its full spaced form, and each
// of its whitespace tokens. Deduplicated.
func targetsFor(cn, cc string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, 4)
	add := func(s string) {
		if s == "" {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	add(cc)
	add(cn)
	for _, t := range strings.Fields(cn) {
		add(t)
		add(compact(t))
	}
	return out
}

// norm lowercases s and collapses every run of non-alphanumeric runes into a
// single space. Letters and digits are preserved (Latin + Cyrillic + others).
func norm(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	inSpace := true
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(unicode.ToLower(r))
			inSpace = false
		} else if !inSpace {
			b.WriteByte(' ')
			inSpace = true
		}
	}
	return strings.TrimSpace(b.String())
}

// compact is norm with all spaces removed.
func compact(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return b.String()
}
