package world

import "testing"

func TestDigestCountsByKind(t *testing.T) {
	entries := []LedgerEntry{
		{CharacterID: 1, Tick: 1, Kind: "harvest"},
		{CharacterID: 1, Tick: 1, Kind: "harvest"},
		{CharacterID: 1, Tick: 2, Kind: "wolf_pack"},
	}
	digest := Digest(entries)
	if digest["harvest"] != 2 || digest["wolf_pack"] != 1 || len(digest) != 2 {
		t.Fatalf("digest counts wrong: %+v", digest)
	}
}

func TestDigestTextNarratives(t *testing.T) {
	if got := DigestText(nil); got != "The world kept turning, and you missed nothing." {
		t.Fatalf("empty digest narrative wrong: %q", got)
	}
	entries := []LedgerEntry{
		{Kind: "timber"},
		{Kind: "timber"},
		{Kind: "timber"},
		{Kind: "wolf_pack"},
	}
	got := DigestText(entries)
	want := "While you were away: 3 timbers, 1 wolf_pack."
	if got != want {
		t.Fatalf("digest text = %q, want %q", got, want)
	}
}
