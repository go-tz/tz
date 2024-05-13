package tzir

import (
	"fmt"
	"github.com/ngrash/go-tz/internal/tzexpand"
	"github.com/ngrash/go-tz/internal/unixtime"
	"github.com/ngrash/go-tz/tzdata"
	"sort"
	"time"
)

func Process(f tzdata.File, zs []tzdata.ZoneLine) ([]Zone, error) {
	var (
		zones []Zone
		// activeOffset is the offset to UT that is applied by the current active zone.
		// Defaults to 0 prior to the first rule.
		activeOffset int64
	)
	for _, z := range zs {
		irz, err := processZone(f, z, activeOffset)
		if err != nil {
			return zones, err
		}
		zones = append(zones, irz)
	}

	return zones, nil
}

//   - A  zone or continuation line L with a named rule set starts with standard time by
//     default: that is, any of L's timestamps preceding L's earliest rule use	the
//     rule in effect after L's first transition into standard time.
func firstStdTransition(ts []Transition) (Transition, bool) {
	for _, t := range ts {
		if t.Rule.Save.Form == tzdata.StandardTime {
			return t, true
		}
	}
	return Transition{}, false
}

func processZone(f tzdata.File, z tzdata.ZoneLine, activeOffset int64) (Zone, error) {

	// TODO: I remember reading something along "if after applying offsets, the effective rule would change ..."
	// TODO: Maybe worth to check this out and apply rues for this edge case here.

	if z.Rules.Form != tzdata.ZoneRulesName {
		return Zone{}, fmt.Errorf("can only handle zones with named rules")
	}

	rs, err := findRules(f.RuleLines, z.Rules.Name)
	if err != nil {
		return Zone{}, err
	}

	var irz = Zone{Line: z}
	y := firstYear(rs)
	for {
		ars := activeRules(rs, y)

		transitions, newOffset, zoneExpired, zoneExpiredAt := processZoneYear(z, y, ars, activeOffset)
		// Update offset for next iteration.
		activeOffset = newOffset

		irz.Transitions = append(irz.Transitions, transitions...)

		if validForever(z, ars) {
			// all active transitions are valid forever.
			// TODO: Check if there are more rules that might become effective in later years.
			irz.Final = transitions
			return irz, nil // reached final ruleset
		} else {
			if zoneExpired {
				irz.Expires = true
				irz.ExpiresAt = zoneExpiredAt
				return irz, nil // zone expired
			}
		}

		y++
		if y == 9999 {
			return Zone{}, fmt.Errorf("error: y = 9999") // something is wrong
		}
	}
}

func processZoneYear(z tzdata.ZoneLine, y int, ars []tzdata.RuleLine, offset int64) ([]Transition, int64, bool, int64) {
	// In the first pass, find occurrences of rules in the current year without any local offsets (the universal time occurrence, utocc).
	// To find the real occurrence, we need to take into account the rule that is in effect when the transition happens.
	var transitions []Transition
	for _, r := range ars {
		utocc := ruleOccurrenceIn(r, y)
		transitions = append(transitions, Transition{
			utoccy: y,
			utocc:  utocc,
			Rule:   r,
			off:    ruleOffset(z, r),
		})
	}

	// Sort transitions by their occurrence that year. This allows us to calculate the real occurrence dates by applying
	// the offset of the active rule.
	sort.Slice(transitions, func(i, j int) bool {
		return transitions[i].utocc < transitions[j].utocc
	})

	// Loop through transitions for this year in the order they occur in.
	for i, t := range transitions {
		// Adjust transition occurrences based on the offset of the previous rule.
		// TODO: This could wrap to past year. Do we need to handle that case separately?
		t.Occ = t.utocc - offset
		transitions[i] = t

		// Update offset for next iteration.
		// TODO: after checking for expiry?
		offset = t.off

		// Check if Zone expires when this rule is applied.
		// TODO: Make sure later transitions this year don't make it in the final set.
		if z.Until.Defined {
			until := tzexpand.Earliest(z.Until)
			// TODO: the offset calculation works for my current example but I doubt it is correct
			until = until - offset + z.Offset.Seconds()
			if expired := t.Occ > until; expired {
				// TODO: Add transition to next zone. There can be no gaps.
				return transitions, offset, true, t.Occ
			}
		}
	}

	return transitions, offset, false, 0
}

type Zone struct {
	Line tzdata.ZoneLine

	// transitions keeps track of all transitions that occur within this zone.
	// transitions that never expired are tracked separately as final.
	Transitions []Transition

	Expires   bool
	ExpiresAt int64

	// final transitions of the zone that will never expire.
	Final []Transition
}

type Transition struct {
	Rule   tzdata.RuleLine
	utoccy int
	utocc  int64
	Occ    int64
	off    int64
}

func ruleOffset(z tzdata.ZoneLine, r tzdata.RuleLine) int64 {
	zoff := z.Offset.Seconds()
	roff := r.Save.TimeOfDay.Seconds()
	return zoff + roff
}

func validForever(z tzdata.ZoneLine, rs []tzdata.RuleLine) bool {
	if z.Until.Defined {
		return false // finite zone
	}

	for _, r := range rs {
		if r.To != tzdata.MaxYear {
			return false // finite rule
		}
	}

	return true
}

func firstYear(rs []tzdata.RuleLine) int {
	if len(rs) == 0 {
		return 0 // TODO: I think we need to handle earlier if there are no rule lines.
	}
	y := int(rs[0].From)
	for _, r := range rs {
		y = min(y, int(r.From))
	}
	return y
}

func activeRules(rs []tzdata.RuleLine, year int) []tzdata.RuleLine {
	var active []tzdata.RuleLine
	for _, r := range rs {
		if int(r.From) <= year && int(r.To) >= year {
			active = append(active, r)
		}
	}
	return active
}

func ruleOccurrenceIn(r tzdata.RuleLine, year int) int64 {
	// TODO: add offset - respect different time forms, e.g. wallclock, utc
	y, m, d := tzexpand.DayOfMonth(year, r.In, r.On)
	hours, minutes, seconds := splitTime(time.Duration(r.At.TimeOfDay))
	return unixtime.FromDateTime(y, int(m), d, hours, minutes, seconds)
}

func zoneExpiration(z tzdata.ZoneLine) int64 {
	return tzexpand.Earliest(z.Until) // TODO: add offset
}

func splitTime(t time.Duration) (int, int, int) {
	h := int(t / time.Hour)
	m := int(t / time.Minute)
	s := int(t / time.Second)
	return h, m, s
}

func findRules(l []tzdata.RuleLine, name string) ([]tzdata.RuleLine, error) {
	var rules []tzdata.RuleLine
	for _, r := range l {
		if r.Name == name {
			rules = append(rules, r)
		}
	}
	if len(rules) == 0 {
		return nil, fmt.Errorf("no rules found for name %s", name)
	}
	return rules, nil
}
