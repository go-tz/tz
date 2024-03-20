package tzdata

import (
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestParse_ExtendedExample(t *testing.T) {
	var input = strings.TrimSpace(`
# Rule  NAME  FROM  TO    -  IN   ON       AT    SAVE  LETTER/S
Rule    Swiss 1941  1942  -  May  Mon>=1   1:00  1:00  S
Rule    Swiss 1941  1942  -  Oct  Mon>=1   2:00  0     -
Rule    EU    1977  1980  -  Apr  Sun>=1   1:00u 1:00  S
Rule    EU    1977  only  -  Sep  lastSun  1:00u 0     -
Rule    EU    1978  only  -  Oct   1       1:00u 0     -
Rule    EU    1979  1995  -  Sep  lastSun  1:00u 0     -
Rule    EU    1981  max   -  Mar  lastSun  1:00u 1:00  S
Rule    EU    1996  max   -  Oct  lastSun  1:00u 0     -

# Zone  NAME           STDOFF      RULES  FORMAT  [UNTIL]
Zone    Europe/Zurich  0:34:08     -      LMT     1853 Jul 16
						0:29:45.50  -      BMT     1894 Jun
						1:00        Swiss  CE%sT   1981
						1:00        EU     CE%sT

Link    Europe/Zurich  Europe/Vaduz
`)

	got, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}

	want := File{
		RuleLines: []RuleLine{
			{Name: "Swiss", From: 1941, To: 1942, In: time.May, On: Day{Form: DayFormAfter, Day: time.Monday, Num: 1}, At: Time{Duration: 1 * time.Hour, Form: WallClock}, Save: Time{Duration: 1 * time.Hour, Form: DaylightSavingTime}, Letter: "S"},
			{Name: "Swiss", From: 1941, To: 1942, In: time.October, On: Day{Form: DayFormAfter, Day: time.Monday, Num: 1}, At: Time{Duration: 2 * time.Hour, Form: WallClock}, Save: Time{Duration: 0, Form: StandardTime}, Letter: ""},
			{Name: "EU", From: 1977, To: 1980, In: time.April, On: Day{Form: DayFormAfter, Day: time.Sunday, Num: 1}, At: Time{Duration: 1 * time.Hour, Form: UniversalTime}, Save: Time{Duration: 1 * time.Hour, Form: DaylightSavingTime}, Letter: "S"},
			{Name: "EU", From: 1977, To: 1977, In: time.September, On: Day{Form: DayFormLast, Day: time.Sunday}, At: Time{Duration: 1 * time.Hour, Form: UniversalTime}, Save: Time{Duration: 0, Form: StandardTime}, Letter: ""},
			{Name: "EU", From: 1978, To: 1978, In: time.October, On: Day{Form: DayFormDayNum, Num: 1}, At: Time{Duration: 1 * time.Hour, Form: UniversalTime}, Save: Time{Duration: 0, Form: StandardTime}, Letter: ""},
			{Name: "EU", From: 1979, To: 1995, In: time.September, On: Day{Form: DayFormLast, Day: time.Sunday}, At: Time{Duration: 1 * time.Hour, Form: UniversalTime}, Save: Time{Duration: 0, Form: StandardTime}, Letter: ""},
			{Name: "EU", From: 1981, To: MaxYear, In: time.March, On: Day{Form: DayFormLast, Day: time.Sunday}, At: Time{Duration: 1 * time.Hour, Form: UniversalTime}, Save: Time{Duration: 1 * time.Hour, Form: DaylightSavingTime}, Letter: "S"},
			{Name: "EU", From: 1996, To: MaxYear, In: time.October, On: Day{Form: DayFormLast, Day: time.Sunday}, At: Time{Duration: 1 * time.Hour, Form: UniversalTime}, Save: Time{Duration: 0, Form: StandardTime}, Letter: ""},
		},
		ZoneLines: []ZoneLine{
			{Name: "Europe/Zurich", Continuation: false, Offset: 34*time.Minute + 8*time.Second, Rules: ZoneRules{Form: ZoneRulesStandard}, Format: "LMT", Until: Until{Defined: true, Year: 1853, Month: time.July, Day: Day{Form: DayFormDayNum, Num: 16}, Parts: UntilDay}},
			{Name: "", Continuation: true, Offset: 29*time.Minute + 45*time.Second + 500*time.Millisecond, Rules: ZoneRules{Form: ZoneRulesStandard}, Format: "BMT", Until: Until{Defined: true, Year: 1894, Month: time.June, Parts: UntilMonth}},
			{Name: "", Continuation: true, Offset: 1 * time.Hour, Rules: ZoneRules{Form: ZoneRulesName, Name: "Swiss"}, Format: "CE%sT", Until: Until{Defined: true, Year: 1981, Parts: UntilYear}},
			{Name: "", Continuation: true, Offset: 1 * time.Hour, Rules: ZoneRules{Form: ZoneRulesName, Name: "EU"}, Format: "CE%sT", Until: Until{Defined: false}},
		},
		LinkLines: []LinkLine{
			{From: "Europe/Zurich", To: "Europe/Vaduz"},
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Parse() mismatch (-want +got):\n%s", diff)
	}
}

func TestParse_Leap(t *testing.T) {
	var input = strings.TrimSpace(`
Leap  2016  Dec    31   23:59:60  +     S
Expires  2020  Dec    28   00:00:00
`)
	got, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}

	want := File{
		LeapLines: []LeapLine{
			{Year: 2016, Month: time.December, Day: 31, Time: HMS{23, 59, 60}, Corr: LeapAdded, Mode: StationaryLeapTime},
		},
		ExpiresLines: []ExpiresLine{
			{Year: 2020, Month: time.December, Day: 28, Time: HMS{0, 0, 0}},
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Parse() mismatch (-want +got):\n%s", diff)
	}
}
