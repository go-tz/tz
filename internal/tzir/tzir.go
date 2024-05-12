package tzir

import (
	"fmt"
	"github.com/ngrash/go-tz/internal/tzexpand"
	"github.com/ngrash/go-tz/internal/unixtime"
	"github.com/ngrash/go-tz/tzdata"
	"github.com/ngrash/go-tz/tzif"
	"sort"
	"time"
)

func Process(f tzdata.File, zs []tzdata.ZoneLine) (tzif.Data, error) {
	var (
		zones []zone
		// final transitions to rule lines that never expire.
		final []transition
		// activeOffset is the offset to UT that is applied by the current active zone.
		// Defaults to 0 prior to the first rule.
		activeOffset int64
	)
	for _, z := range zs {
		if z.Rules.Form != tzdata.ZoneRulesName {
			continue // TODO
		}

		rs, err := findRules(f.RuleLines, z.Rules.Name)
		if err != nil {
			return tzif.Data{}, err
		}

		var irz zone
		y := firstYear(rs)
		for {
			ars := activeRules(rs, y)

			// In the first pass, find occurrences of rules in the current year without any local offsets (the universal time occurrence, utocc).
			// To find the real occurrence, we need to take into account the rule that is in effect when the transition happens.
			var transitions []transition
			for _, r := range ars {
				utocc := ruleOccurrenceIn(r, y)
				transitions = append(transitions, transition{
					utoccy: y,
					utocc:  utocc,
					r:      r,
					off:    ruleOffset(z, r),
				})
			}

			// Sort transitions by their occurrence that year. This allows us to calculate the real occurrence dates by applying
			// the offset of the active rule.
			sort.Slice(transitions, func(i, j int) bool {
				return transitions[i].utocc < transitions[j].utocc
			})

			// Loop through transitions for this year in the order they occur in.
			// Adjust their occurrences based on the offset of the effective rule.
			var done bool
			for i, t := range transitions {
				// TODO: This could wrap to past year. Do we need to handle that case separately?
				t.occ = t.utocc - activeOffset
				transitions[i] = t

				activeOffset = t.off

				// Remember zone's first transition to standard time.
				if t.r.Save.Form == tzdata.StandardTime && !irz.definesStdTime {
					irz.firstStdTime = t
					irz.definesStdTime = true
				}

				// Check if Zone expires when this rule is applied.
				if z.Until.Defined {
					until := tzexpand.Earliest(z.Until)
					// TODO: the offset calculation works for my current example but I doubt it is correct
					until = until - activeOffset + z.Offset.Seconds()
					if expired := t.occ > until; expired {
						// TODO: Add transition to next zone. There can be no gaps.
						fmt.Printf("Zone expired: %d (transition needed)\n", until)
						done = true
						break
					}
				}
				fmt.Printf("%+v\n", t)
			}
			if done {
				fmt.Println("done")
				break
			}

			// TODO: I remember reading something along "if after applying offsets, the effective rule would change ..."
			// TODO: Maybe worth to check this out and apply rues for this edge case here.

			if validForever(z, ars) {
				fmt.Printf("valid forever: %d\n", len(ars))
				final = transitions
				break // done
			}
			y++

			if y == 2030 {
				return tzif.Data{}, fmt.Errorf("error: y = 3000")
			}
		}
		zones = append(zones, irz)
	}

	// We have rules which are valid indefinitely.
	if len(final) > 0 {
		fmt.Println("final")
		if len(final) > 2 {
			return tzif.Data{}, fmt.Errorf("cannot handle more than two rules that never expire")
		}

		for _, t := range final {
			fmt.Printf("%+v\n", t)
		}
	}

	return tzif.Data{}, nil
}

type zone struct {
	z tzdata.ZoneLine

	expires   bool
	expiresAt int64

	// * A  zone or continuation line L with a named rule set starts with standard time by
	//		//   default: that is, any of L's timestamps preceding L's earliest rule use	the
	//		//   rule in effect after L's first transition into standard time.
	firstStdTime   transition
	definesStdTime bool

	// transitions keeps track of all transitions that occur within this zone.
	// transitions that never expired are tracked separately as final.
	transitions []transition

	// final transitions of the zone that will never expire.
	final []transition
}

type transition struct {
	r      tzdata.RuleLine
	utoccy int
	utocc  int64
	occ    int64
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
