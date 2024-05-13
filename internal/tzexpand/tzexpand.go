package tzexpand

import (
	"fmt"
	"github.com/ngrash/go-tz/internal/unixtime"
	"github.com/ngrash/go-tz/tzdata"
	"sort"
	"time"
)

func Earliest(u tzdata.Until) int64 {
	e := earliest(u)

	hours := int(time.Duration(e.Time.TimeOfDay) / time.Hour)
	minutes := int(time.Duration(e.Time.TimeOfDay) / time.Minute)
	seconds := int(time.Duration(e.Time.TimeOfDay) / time.Second)

	return unixtime.FromDateTime(e.Year, int(e.Month), e.Day.Num, hours, minutes, seconds)
}

func earliest(u tzdata.Until) tzdata.Until {
	// If the UNTIL column is not defined, return the zero value.
	if !u.Defined {
		return u
	}

	// If a part is not defined, set it to the earliest possible value.
	if !u.Parts.Has(tzdata.UntilMonthOnly) {
		u.Month = time.January
		u.Parts = u.Parts.Set(tzdata.UntilMonthOnly)
	}

	// If the day is defined, set it to the earliest possible value for the month.
	if u.Parts.Has(tzdata.UntilDayOnly) {
		if u.Day.Form != tzdata.DayFormNum {
			// Calculate the real day of the month.
			var num int
			u.Year, u.Month, num = DayOfMonth(u.Year, u.Month, u.Day)
			u.Day = tzdata.Day{Form: tzdata.DayFormNum, Num: num}
		}
	} else {
		u.Day = tzdata.Day{Form: tzdata.DayFormNum, Num: 1}
		u.Parts = u.Parts.Set(tzdata.UntilDayOnly)
	}

	if !u.Parts.Has(tzdata.UntilTimeOnly) {
		u.Time = tzdata.Time{TimeOfDay: 0, Form: tzdata.WallClock}
		u.Parts = u.Parts.Set(tzdata.UntilTimeOnly)
	}

	return u
}

func DayOfMonth(year int, month time.Month, d tzdata.Day) (y int, m time.Month, day int) {
	switch d.Form {
	case tzdata.DayFormNum:
		return year, month, d.Num
	case tzdata.DayFormLast:
		num := lastWeekdayOfMonth(year, int(month), int(d.Day))
		return year, month, num
	case tzdata.DayFormAfter:
		y, m, d := nextWeekday(year, int(month), d.Num, int(d.Day))
		return y, time.Month(m), d
	case tzdata.DayFormBefore:
		y, m, d := lastWeekday(year, int(month), d.Num, int(d.Day))
		return y, time.Month(m), d
	}
	panic(fmt.Errorf("invalid DayForm: %q", d.Form))
}

var (
	EpochMin = Moment{Year: 1902, Month: time.January, Day: 1, Time: tzdata.NewWallClock(0 * time.Hour)} // UNIX epoch min for 32-bit integers
	Epoch0   = Moment{Year: 1970, Month: time.January, Day: 1, Time: tzdata.NewWallClock(0 * time.Hour)}
	EpochMax = Moment{Year: 2038, Month: time.January, Day: 19, Time: tzdata.NewWallClock(3*time.Hour + 14*time.Minute + 7*time.Second)} // UNIX epoch max for 32-bit integers
)

// Moment is a limit line with the year, month, and day expanded.
type Moment struct {
	Year  int
	Month time.Month
	Day   int
	Time  tzdata.Time
}

func ExpandRules(min, max Moment, r []tzdata.RuleLine) []tzdata.RuleLine {
	var tr []tzdata.RuleLine
	for _, rule := range r {
		tr = append(tr, expandRule(min, max, rule)...)
	}

	// Sort the rules by year, month, and day.
	sort.Slice(tr, func(i, j int) bool {
		if tr[i].From != tr[j].From {
			return tr[i].From < tr[j].From
		}
		if tr[i].In != tr[j].In {
			return tr[i].In < tr[j].In
		}
		return tr[i].On.Num < tr[j].On.Num
	})

	return tr
}

func expandRule(min, max Moment, rl tzdata.RuleLine) []tzdata.RuleLine {
	if rl.From == tzdata.MinYear {
		rl.From = tzdata.Year(min.Year)
	}
	if rl.To == tzdata.MaxYear {
		rl.To = tzdata.Year(max.Year)
	}

	var tr []tzdata.RuleLine
	for year := rl.From; year <= rl.To; year++ {
		y, m, d := DayOfMonth(int(year), rl.In, rl.On)
		r := tzdata.RuleLine{
			Name:   rl.Name,
			From:   tzdata.Year(y),
			To:     tzdata.Year(y),
			In:     m,
			On:     tzdata.NewDayNum(d),
			At:     rl.At,
			Save:   rl.Save,
			Letter: rl.Letter,
		}

		if int(r.From) < min.Year || int(r.From) > max.Year {
			continue
		}
		if int(r.From) == max.Year && r.In > max.Month {
			continue
		}
		if int(r.From) == min.Year && r.In < min.Month {
			continue
		}
		if int(r.From) == max.Year && r.In == max.Month && r.On.Num > max.Day {
			continue
		}
		if int(r.From) == min.Year && r.In == min.Month && r.On.Num < min.Day {
			continue
		}
		// TODO: Normalize time and check it also.
		tr = append(tr, r)
	}
	return tr
}
