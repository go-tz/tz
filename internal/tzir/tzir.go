package tzir

import (
	"fmt"
	"github.com/ngrash/go-tz/internal/tzexpand"
	"github.com/ngrash/go-tz/internal/unixtime"
	"github.com/ngrash/go-tz/tzdata"
	"sort"
	"strings"
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

func processZone(f tzdata.File, z tzdata.ZoneLine, activeOffset int64) (Zone, error) {

	// TODO: I remember reading something along "if after applying offsets, the effective rule would change ..."
	// TODO: Maybe worth to check this out and apply rues for this edge case here.

	if z.Rules.Form == tzdata.ZoneRulesStandard || z.Rules.Form == tzdata.ZoneRulesTime {
		// Zone has no rules, so it is always in standard time.
		return Zone{
			Line:        z,
			Transitions: nil, // there are no transitions in this zone
			FirstStdTransition: Transition{
				UTOccY: 0,
				UTOcc:  0,
				Occ:    0,
				Off:    z.Rules.Time.Seconds() + z.Offset.Seconds(),
				Dst:    z.Rules.Form == tzdata.ZoneRulesTime,
				Desig:  designation(z.Format, ""),
			},
			HasStdTransition: true,
		}, nil
	}

	if z.Rules.Form != tzdata.ZoneRulesName {
		return Zone{}, fmt.Errorf("cant handle zone rules form %s", z.Rules.Form)
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

		// Remember first transition to standard time. It is used for timestamps prior to the first transition.
		// This is relevant when there are gaps between zones and for the initial transition.
		if !irz.HasStdTransition {
			for _, t := range transitions {
				if !t.Dst {
					irz.FirstStdTransition = t
					irz.HasStdTransition = true
					break
				}
			}
		}

		if validForever(z, ars) {
			// all active transitions are valid forever.
			// TODO: Check if there are more rules that might become effective in later years.
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
	var all []Transition
	for _, r := range ars {
		utocc := ruleOccurrenceIn(r, y)
		all = append(all, Transition{
			UTOccY: y,
			UTOcc:  utocc,
			Rule:   r,
			Off:    ruleOffset(z, r),
			Dst:    r.Save.Form == tzdata.DaylightSavingTime,
			Desig:  designation(z.Format, r.Letter),
		})
	}

	// Sort transitions by their occurrence that year. This allows us to calculate the real occurrence dates by applying
	// the offset of the active rule.
	sort.Slice(all, func(i, j int) bool {
		return all[i].UTOcc < all[j].UTOcc
	})

	// actual transitions that happen during the validity of the zone.
	var actual []Transition

	// Loop through transitions for this year in the order they occur in.
	for _, t := range all {
		// Adjust transition occurrences based on the offset of the previous rule.
		// TODO: This could wrap to past year. Do we need to handle that case separately?
		t.Occ = t.UTOcc - offset

		// Update offset for next iteration.
		// TODO: after checking for expiry?
		offset = t.Off

		// Check if Zone expires when this rule is applied.
		if z.Until.Defined {
			until := tzexpand.Earliest(z.Until)
			// TODO: the offset calculation works for my current example but I doubt it is correct
			until = until - offset + z.Offset.Seconds()
			if expired := t.Occ > until; expired {
				// Zone expired. Return all transitions that happened before this one.
				return actual, offset, true, until
			}
		}

		// Zone did not expire before this transition, so is actually happens.
		actual = append(actual, t)
	}

	return actual, offset, false, 0
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

	FirstStdTransition Transition
	HasStdTransition   bool
}

type Transition struct {
	Rule   tzdata.RuleLine
	UTOccY int
	UTOcc  int64
	Occ    int64
	Off    int64
	Dst    bool
	Desig  string
}

func designation(format, letter string) string {
	desig := format
	if strings.Contains(format, "%s") {
		desig = strings.ReplaceAll(format, "%s", letter)
	}
	return desig
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
