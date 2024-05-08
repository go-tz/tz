package tzexpand

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/ngrash/go-tz/tzdata"
)

func TestDayOfMonth(t *testing.T) {
	type in struct {
		Year  int
		Month time.Month
		Day   tzdata.Day
	}
	type want struct {
		Year  int
		Month time.Month
		Day   int
	}
	cases := []struct {
		in   in
		want want
	}{
		{in{2021, time.March, tzdata.NewDayNum(23)}, want{2021, time.March, 23}},
		{in{2021, time.March, tzdata.NewDayLast(time.Sunday)}, want{2021, time.March, 28}},

		// Leap day
		{in{2020, time.February, tzdata.NewDayAfter(28, time.Saturday)}, want{2020, time.February, 29}},
		{in{2020, time.February, tzdata.NewDayLast(time.Saturday)}, want{2020, time.February, 29}},
		// Day Leap day in a non-leap year
		{in{2021, time.February, tzdata.NewDayAfter(28, time.Saturday)}, want{2021, time.March, 6}},

		// Day of week is on the exact day of month
		{in{2021, time.March, tzdata.NewDayAfter(28, time.Sunday)}, want{2021, time.March, 28}},
		// Day of week is later in the same month
		{in{2021, time.March, tzdata.NewDayAfter(15, time.Sunday)}, want{2021, time.March, 21}},
		// Day of week is next month
		{in{2021, time.March, tzdata.NewDayAfter(30, time.Sunday)}, want{2021, time.April, 4}},
		// Day of week is next year
		{in{2021, time.December, tzdata.NewDayAfter(30, time.Sunday)}, want{2022, time.January, 2}},

		// Day of week is on the exact day of month
		{in{2021, time.March, tzdata.NewDayBefore(28, time.Sunday)}, want{2021, time.March, 28}},
		// Day of week is earlier in the same month
		{in{2021, time.March, tzdata.NewDayBefore(15, time.Sunday)}, want{2021, time.March, 14}},
		// Day of week is last month
		{in{2021, time.March, tzdata.NewDayBefore(5, time.Sunday)}, want{2021, time.February, 28}},
		// Day of week is last year
		{in{2021, time.January, tzdata.NewDayBefore(2, time.Sunday)}, want{2020, time.December, 27}},
	}

	for _, c := range cases {
		y, m, d := DayOfMonth(c.in.Year, c.in.Month, c.in.Day)
		got := want{y, m, d}
		if diff := cmp.Diff(c.want, got); diff != "" {
			t.Errorf("Day.Resolve(%+v) mismatch (-want +got):\n%s", c.in, diff)
		}
	}
}

func TestExpandRule(t *testing.T) {
	cases := []struct {
		name string
		from Moment
		to   Moment
		in   tzdata.RuleLine
		want []tzdata.RuleLine
	}{
		{
			name: "Single year, not limited by from and to",
			from: Moment{Year: 1980},
			to:   Moment{Year: 1990},
			in: tzdata.RuleLine{
				Name:   "EU",
				From:   1981,
				To:     1981,
				In:     time.March,
				On:     tzdata.NewDayLast(time.Sunday),
				At:     tzdata.NewWallClock(11 * time.Hour),
				Save:   tzdata.NewDaylightSavingTime(1 * time.Hour),
				Letter: "S",
			},
			want: []tzdata.RuleLine{
				{
					From:   1981,
					To:     1981,
					In:     time.March,
					On:     tzdata.NewDayNum(29),
					At:     tzdata.NewWallClock(11 * time.Hour),
					Save:   tzdata.NewDaylightSavingTime(1 * time.Hour),
					Letter: "S",
				},
			},
		},

		{
			name: "Multiple years, not limited by from and to",
			from: Moment{Year: 1980},
			to:   Moment{Year: 1990},
			in: tzdata.RuleLine{
				Name:   "EU",
				From:   1981,
				To:     1983,
				In:     time.March,
				On:     tzdata.NewDayLast(time.Sunday),
				At:     tzdata.NewWallClock(11 * time.Hour),
				Save:   tzdata.NewDaylightSavingTime(1 * time.Hour),
				Letter: "S",
			},
			want: []tzdata.RuleLine{
				{
					From:   1981,
					To:     1981,
					In:     time.March,
					On:     tzdata.NewDayNum(29),
					At:     tzdata.NewWallClock(11 * time.Hour),
					Save:   tzdata.NewDaylightSavingTime(1 * time.Hour),
					Letter: "S",
				},
				{
					From:   1982,
					To:     1982,
					In:     time.March,
					On:     tzdata.NewDayNum(28),
					At:     tzdata.NewWallClock(11 * time.Hour),
					Save:   tzdata.NewDaylightSavingTime(1 * time.Hour),
					Letter: "S",
				},
				{
					From:   1983,
					To:     1983,
					In:     time.March,
					On:     tzdata.NewDayNum(27),
					At:     tzdata.NewWallClock(11 * time.Hour),
					Save:   tzdata.NewDaylightSavingTime(1 * time.Hour),
					Letter: "S",
				},
			},
		},

		{
			name: "Single year, excluded by from and to",
			from: Moment{Year: 1990},
			to:   Moment{Year: 1993},
			in: tzdata.RuleLine{
				Name:   "EU",
				From:   1981,
				To:     1981,
				In:     time.March,
				On:     tzdata.NewDayLast(time.Sunday),
				At:     tzdata.NewWallClock(11 * time.Hour),
				Save:   tzdata.NewDaylightSavingTime(1 * time.Hour),
				Letter: "S",
			},
			want: nil,
		},

		{
			name: "Multiple years, limited by from and to",
			from: Moment{Year: 1982},
			to:   Moment{Year: 1982, Month: time.March, Day: 28},
			in: tzdata.RuleLine{
				Name:   "EU",
				From:   1981,
				To:     1983,
				In:     time.March,
				On:     tzdata.NewDayLast(time.Sunday),
				At:     tzdata.NewWallClock(11 * time.Hour),
				Save:   tzdata.NewDaylightSavingTime(1 * time.Hour),
				Letter: "S",
			},
			want: []tzdata.RuleLine{
				{
					From:   1982,
					To:     1982,
					In:     time.March,
					On:     tzdata.NewDayNum(28),
					At:     tzdata.NewWallClock(11 * time.Hour),
					Save:   tzdata.NewDaylightSavingTime(1 * time.Hour),
					Letter: "S",
				},
			},
		},

		{
			name: "Starts at MinYear",
			from: Moment{Year: 1982},
			to:   Moment{Year: 1983, Month: time.May},
			in: tzdata.RuleLine{
				Name:   "EU",
				From:   tzdata.MinYear,
				To:     1983,
				In:     time.March,
				On:     tzdata.NewDayLast(time.Sunday),
				At:     tzdata.NewWallClock(11 * time.Hour),
				Save:   tzdata.NewDaylightSavingTime(1 * time.Hour),
				Letter: "S",
			},
			want: []tzdata.RuleLine{
				{
					From:   1982,
					To:     1982,
					In:     time.March,
					On:     tzdata.NewDayNum(28),
					At:     tzdata.NewWallClock(11 * time.Hour),
					Save:   tzdata.NewDaylightSavingTime(1 * time.Hour),
					Letter: "S",
				},
				{
					From:   1983,
					To:     1983,
					In:     time.March,
					On:     tzdata.NewDayNum(27),
					At:     tzdata.NewWallClock(11 * time.Hour),
					Save:   tzdata.NewDaylightSavingTime(1 * time.Hour),
					Letter: "S",
				},
			},
		},

		{
			name: "Ends at MaxYear",
			from: Moment{Year: 1982},
			to:   Moment{Year: 1983, Month: time.May},
			in: tzdata.RuleLine{
				Name:   "EU",
				From:   1982,
				To:     tzdata.MaxYear,
				In:     time.March,
				On:     tzdata.NewDayLast(time.Sunday),
				At:     tzdata.NewWallClock(11 * time.Hour),
				Save:   tzdata.NewDaylightSavingTime(1 * time.Hour),
				Letter: "S",
			},
			want: []tzdata.RuleLine{
				{
					From:   1982,
					To:     1982,
					In:     time.March,
					On:     tzdata.NewDayNum(28),
					At:     tzdata.NewWallClock(11 * time.Hour),
					Save:   tzdata.NewDaylightSavingTime(1 * time.Hour),
					Letter: "S",
				},
				{
					From:   1983,
					To:     1983,
					In:     time.March,
					On:     tzdata.NewDayNum(27),
					At:     tzdata.NewWallClock(11 * time.Hour),
					Save:   tzdata.NewDaylightSavingTime(1 * time.Hour),
					Letter: "S",
				},
			},
		},

		{
			name: "Starts at MinYear, ends at MaxYear",
			from: Moment{Year: 1982},
			to:   Moment{Year: 1983, Month: time.May},
			in: tzdata.RuleLine{
				Name:   "EU",
				From:   tzdata.MinYear,
				To:     tzdata.MaxYear,
				In:     time.March,
				On:     tzdata.NewDayLast(time.Sunday),
				At:     tzdata.NewWallClock(11 * time.Hour),
				Save:   tzdata.NewDaylightSavingTime(1 * time.Hour),
				Letter: "S",
			},
			want: []tzdata.RuleLine{
				{
					From:   1982,
					To:     1982,
					In:     time.March,
					On:     tzdata.NewDayNum(28),
					At:     tzdata.NewWallClock(11 * time.Hour),
					Save:   tzdata.NewDaylightSavingTime(1 * time.Hour),
					Letter: "S",
				},
				{
					From:   1983,
					To:     1983,
					In:     time.March,
					On:     tzdata.NewDayNum(27),
					At:     tzdata.NewWallClock(11 * time.Hour),
					Save:   tzdata.NewDaylightSavingTime(1 * time.Hour),
					Letter: "S",
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := expandRule(c.from, c.to, c.in)
			if diff := cmp.Diff(c.want, got); diff != "" {
				t.Errorf("expandRule(%v) mismatch (-want +got):\n%s", c.in, diff)
			}
		})
	}
}
