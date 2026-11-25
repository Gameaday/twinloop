// Package world holds the settlement sim domain: the duration-based
// action queue. Costs are real-world minutes, never ticks (design
// invariant 4): raising ticks-per-day never changes what an action costs.
package world

// Action is one queued player action.
type Action struct {
	Kind       string
	Minutes    int  // real-world cost in minutes
	Done       int  // minutes already spent on this action
	DailyGated bool // once-per-window action; ClampDelta territory
}

// Advance consumes deltaMinutes of real time across the queue in order.
// It returns the actions that completed and the remaining queue; a
// partially finished head keeps its accumulated Done time so the next
// tick resumes it instead of restarting. Elapsed must be clamped with
// ClampDelta before reaching here.
func Advance(queue []Action, deltaMinutes int) (completed []Action, rest []Action) {
	rest = queue
	for len(rest) > 0 && deltaMinutes > 0 {
		head := rest[0]
		remaining := head.Minutes - head.Done
		if remaining <= 0 { // already-completed head: resume after a crash
			completed = append(completed, head)
			rest = rest[1:]
			continue
		}
		if deltaMinutes >= remaining {
			deltaMinutes -= remaining
			head.Done = head.Minutes
			completed = append(completed, head)
			rest = rest[1:]
		} else {
			head.Done += deltaMinutes
			deltaMinutes = 0
			rest[0] = head
		}
	}
	return completed, rest
}

// ClampDelta caps catch-up so a long outage cannot fast-forward the whole
// world in one tick (red-team R6). The chronicle narrates what the world
// "endured"; later ticks pick up any backlog. maxCatchUpMinutes <= 0
// disables the cap.
func ClampDelta(deltaMinutes, maxCatchUpMinutes int) int {
	if deltaMinutes < 0 {
		return 0
	}
	if maxCatchUpMinutes <= 0 || deltaMinutes <= maxCatchUpMinutes {
		return deltaMinutes
	}
	return maxCatchUpMinutes
}
