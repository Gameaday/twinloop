// Package delve holds the dungeon roguelike domain. Generation and all
// rolls live server-side; the daily seed is an HMAC over a server-only
// secret and never reaches a client (red-team R2: a deterministic
// seed in the open plus a discoverable client seed equals free perfect
// bots, so the seed never ships).
package delve

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"time"
)

// DailySeed derives the day's 32-byte seed from the server-only secret.
func DailySeed(secret []byte, dayIndex int64) []byte {
	mac := hmac.New(sha256.New, secret)
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], uint64(dayIndex))
	mac.Write(b[:])
	return mac.Sum(nil)
}

// DayIndex counts whole UTC days since epoch (inclusive, first day = 0).
// UTC ticks avoid DST ambiguity and match the daily.historylabs.dev
// epoch convention on the fleet.
func DayIndex(now time.Time, epoch time.Time) int64 {
	now, epoch = now.UTC(), epoch.UTC()
	nowMidnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	epochMidnight := time.Date(epoch.Year(), epoch.Month(), epoch.Day(), 0, 0, 0, 0, time.UTC)
	days := nowMidnight.Sub(epochMidnight).Hours() / 24
	if days < 0 {
		return 0
	}
	return int64(days)
}
