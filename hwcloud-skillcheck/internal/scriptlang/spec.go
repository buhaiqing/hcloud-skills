package scriptlang

import "strings"

type Entry struct {
	Match    string `json:"match"`
	Response string `json:"response"`
	ExitCode int    `json:"exit_code"`
}

type Spec struct {
	Entries []Entry `json:"entries"`
}

// Match returns the entry whose Match is the longest prefix of cmd. If
// no entry matches, ok is false.
func (s *Spec) Match(cmd string) (Entry, bool) {
	var best Entry
	bestLen := -1
	for _, e := range s.Entries {
		if strings.HasPrefix(cmd, e.Match) && len(e.Match) > bestLen {
			best = e
			bestLen = len(e.Match)
		}
	}
	if bestLen < 0 {
		return Entry{}, false
	}
	return best, true
}
