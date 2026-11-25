package world

import "sort"

// LedgerEntry is one append-only world event attached to a character.
// Entries are never edited or deleted; digests read them in order so a
// returning player gets a faithful "what happened since you were away"
// summary (the catch-up digest).
type LedgerEntry struct {
	CharacterID int64
	Tick        int64
	Kind        string
}

// Digest counts events by kind. The UI layer renders the map as a short
// narrative ("4 tasks completed, 30 timber harvested, 1 wolf pack seen").
func Digest(entries []LedgerEntry) map[string]int {
	counts := make(map[string]int, len(entries))
	for _, e := range entries {
		counts[e.Kind]++
	}
	return counts
}

// DigestText renders a digest in the canonical order: everything that
// happened, most frequent first, ties alphabetical. Pure text, no art.
func DigestText(entries []LedgerEntry) string {
	counts := Digest(entries)
	if len(counts) == 0 {
		return "The world kept turning, and you missed nothing."
	}
	kinds := make([]string, 0, len(counts))
	for kind := range counts {
		kinds = append(kinds, kind)
	}
	sort.Slice(kinds, func(i, j int) bool {
		if counts[kinds[i]] != counts[kinds[j]] {
			return counts[kinds[i]] > counts[kinds[j]]
		}
		return kinds[i] < kinds[j]
	})
	out := "While you were away: "
	for i, kind := range kinds {
		if i > 0 {
			out += ", "
		}
		out += humanizeCounts(counts[kind], kind)
	}
	out += "."
	return out
}

func humanizeCounts(n int, kind string) string {
	if n == 1 {
		return "1 " + kind
	}
	return itoa(n) + " " + kind + "s"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		return "-" + string(b)
	}
	return string(b)
}
