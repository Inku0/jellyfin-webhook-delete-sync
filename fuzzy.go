package main

/*
Source: https://github.com/sahilm/fuzzy
modified for:
1. laxer equivalence in matching
	- added runesMatch(candidate, pattern) instead of strict equalFold(...)
		- runesMatch returns true when both runes are separators (space, punctuation, symbols), so : and space can be considered compatible

2. loop/index safety
	- guard against patternIndex >= len(runes) before indexing runes[patternIndex]
		- broke the scan once the full pattern was matched to avoid out-of-range behavior and missed completions

3. handle unequal separator counts
	- added separator skipping logic so extra separators on one side don’t force failure:
		- skip separator runes in the pattern when candidate isn’t a separator
		- optionally skip extra separator runes in the candidate when pattern isn’t on a separator
			- this is what made "2001: A Space Odyssey" match "2001 A Space Odyssey" reliably

*/

import (
	"sort"
	"unicode"
	"unicode/utf8"
)

// Match represents a matched string.
type Match struct {
	// The matched string.
	Str string
	// The index of the matched string in the supplied slice.
	Index int
	// The indexes of matched characters. Useful for highlighting matches.
	MatchedIndexes []int
	// Score used to rank matches
	Score int
}

const (
	firstCharMatchBonus            = 10
	matchFollowingSeparatorBonus   = 20
	camelCaseMatchBonus            = 20
	adjacentMatchBonus             = 5
	unmatchedLeadingCharPenalty    = -5
	maxUnmatchedLeadingCharPenalty = -15
)

// Matches is a slice of Match structs
type Matches []Match

func (a Matches) Len() int           { return len(a) }
func (a Matches) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a Matches) Less(i, j int) bool { return a[i].Score >= a[j].Score }

// Source represents an abstract source of a list of strings. Source must be iterable type such as a slice.
// The source will be iterated over till Len() with String(i) being called for each element where i is the
// index of the element. You can find a working example in the README.
type Source interface {
	// The string to be matched at position i.
	String(i int) string
	// The length of the source. Typically is the length of the slice of things that you want to match.
	Len() int
}

type stringSource []string

func (ss stringSource) String(i int) string {
	return ss[i]
}

func (ss stringSource) Len() int { return len(ss) }

/*
Find looks up pattern in data and returns matches
in descending order of match quality. Match quality
is determined by a set of bonus and penalty rules.

The following types of matches apply a bonus:

* The first character in the pattern matches the first character in the match string.

* The matched character is camel cased.

* The matched character follows a separator such as an underscore character.

* The matched character is adjacent to a previous match.

Penalties are applied for every character in the search string that wasn't matched and all leading
characters upto the first match.

Results are sorted by best match.
*/
func Find(pattern string, data []string) Matches {
	return FindFrom(pattern, stringSource(data))
}

/*
FindNoSort is an alternative Find implementation that does not sort
the results in the end.
*/
func FindNoSort(pattern string, data []string) Matches {
	return FindFromNoSort(pattern, stringSource(data))
}

/*
FindFrom is an alternative implementation of Find using a Source
instead of a list of strings.
*/
func FindFrom(pattern string, data Source) Matches {
	matches := FindFromNoSort(pattern, data)
	sort.Stable(matches)
	return matches
}

/*
FindFromNoSort is an alternative FindFrom implementation that does
not sort results in the end.
*/
func FindFromNoSort(pattern string, data Source) Matches {
	if len(pattern) == 0 {
		return nil
	}

	pat := []rune(pattern)
	var matches Matches
	var matchedIndexes []int

	for i := 0; i < data.Len(); i++ {
		s := data.String(i)

		var match Match
		match.Str = s
		match.Index = i
		if matchedIndexes != nil {
			match.MatchedIndexes = matchedIndexes
		} else {
			match.MatchedIndexes = make([]int, 0, len(pat))
		}

		pi := 0
		var last rune
		lastIndex := -1
		currAdjacent := 0

		for j := 0; j < len(s); {
			if pi >= len(pat) {
				break
			}

			c, size := utf8.DecodeRuneInString(s[j:])

			// Skip extra separators in pattern (e.g. ':' in "2001: A...")
			for pi < len(pat) && isSeparator(pat[pi]) && !isSeparator(c) {
				pi++
			}
			if pi >= len(pat) {
				break
			}

			// Skip extra separators in candidate (optional but robust)
			if isSeparator(c) && !isSeparator(pat[pi]) {
				lastIndex = j
				last = c
				j += size
				continue
			}

			if runesMatch(c, pat[pi]) {
				score := 0
				if j == 0 {
					score += firstCharMatchBonus
				}
				if unicode.IsLower(last) && unicode.IsUpper(c) {
					score += camelCaseMatchBonus
				}
				if j != 0 && isSeparator(last) {
					score += matchFollowingSeparatorBonus
				}
				if len(match.MatchedIndexes) > 0 {
					prev := match.MatchedIndexes[len(match.MatchedIndexes)-1]
					b := adjacentCharBonus(lastIndex, prev, currAdjacent)
					score += b
					currAdjacent += b
				}
				if len(match.MatchedIndexes) == 0 {
					pen := j * unmatchedLeadingCharPenalty
					score += max(pen, maxUnmatchedLeadingCharPenalty)
				}
				match.Score += score
				match.MatchedIndexes = append(match.MatchedIndexes, j)
				pi++
			}

			lastIndex = j
			last = c
			j += size
		}

		// Allow trailing separators in pattern
		for pi < len(pat) && isSeparator(pat[pi]) {
			pi++
		}

		// penalty for unmatched chars in candidate
		match.Score += len(match.MatchedIndexes) - len(s)

		if pi == len(pat) {
			matches = append(matches, match)
			matchedIndexes = nil
		} else {
			matchedIndexes = match.MatchedIndexes[:0]
		}
	}

	return matches
}

// Relaxed version of equalFold
func runesMatch(candidate, pattern rune) bool {
	// exact/case-insensitive first
	if equalFold(candidate, pattern) {
		return true
	}
	// treat all separators as equivalent (space, colon, dash, slash, etc.)
	return isSeparator(candidate) && isSeparator(pattern)
}

// Taken from strings.EqualFold
func equalFold(tr, sr rune) bool {
	if tr == sr {
		return true
	}
	if tr < sr {
		tr, sr = sr, tr
	}
	// Fast check for ASCII.
	if tr < utf8.RuneSelf {
		// ASCII, and sr is upper case.  tr must be lower case.
		if 'A' <= sr && sr <= 'Z' && tr == sr+'a'-'A' {
			return true
		}
		return false
	}

	// General case. SimpleFold(x) returns the next equivalent rune > x
	// or wraps around to smaller values.
	r := unicode.SimpleFold(sr)
	for r != sr && r < tr {
		r = unicode.SimpleFold(r)
	}
	return r == tr
}

func adjacentCharBonus(i int, lastMatch int, currentBonus int) int {
	if lastMatch == i {
		return currentBonus*2 + adjacentMatchBonus
	}
	return 0
}

func isSeparator(r rune) bool {
	if unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsSymbol(r) {
		return true
	}
	switch r {
	case '/', '-', '_', '.', '\\':
		return true
	}
	return false
}
