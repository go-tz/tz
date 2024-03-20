// Package tzdata provides a parser for the tzdata and leapsecond files provided by IANA
// at https://www.iana.org/time-zones.
package tzdata

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"time"
)

// File represents the result of parsing a tzdata or leapsecond file.
// It contains the parsed zone lines, rule lines, and link lines, each in the order they appear in the file.
// It also contains the parsed leap lines and expires lines, if the file is a leapsecond file.
// The data structure is shared between the two file types, but the leap lines and expires lines are only
// present in leapsecond files while the zone lines, rule lines, and link lines are only present in tzdata files.
type File struct {
	ZoneLines    []ZoneLine
	RuleLines    []RuleLine
	LinkLines    []LinkLine
	LeapLines    []LeapLine
	ExpiresLines []ExpiresLine
}

// parseError is an error that occurred during parsing.
// It contains the line number and the line where the error occurred.
type parseError struct {
	lineNumber int
	line       string
	err        error
}

// Error returns a string representation of the parse error, implementing the error interface.
func (e *parseError) Error() string {
	return fmt.Sprintf("line %d: %q: %v", e.lineNumber, e.line, e.err)
}

// zoneContinuationParseError returns a parse error for a zone continuation line.
func zoneContinuationParseError(lineNumber int, line string, err error) error {
	return &parseError{lineNumber, line, fmt.Errorf("parse zone continuation: %w", err)}
}

// zoneParseError returns a parse error for a zone line.
func zoneParseError(lineNumber int, line string, err error) error {
	return &parseError{lineNumber, line, fmt.Errorf("parse zone: %w", err)}
}

// ruleParseError returns a parse error for a rule line.
func ruleParseError(lineNumber int, line string, err error) error {
	return &parseError{lineNumber, line, fmt.Errorf("parse rule: %w", err)}
}

// linkParseError returns a parse error for a link line.
func linkParseError(lineNumber int, line string, err error) error {
	return &parseError{lineNumber, line, fmt.Errorf("parse link: %w", err)}
}

// leapParseError returns a parse error for a leap line.
func leapParseError(lineNumber int, line string, err error) error {
	return &parseError{lineNumber, line, fmt.Errorf("parse leap: %w", err)}
}

// expiresParseError returns a parse error for an expires line.
func expiresParseError(lineNumber int, line string, err error) error {
	return &parseError{lineNumber, line, fmt.Errorf("parse expires: %w", err)}
}

// Parse parses the content of tzdata file and returns a File struct containing the parsed lines.
func Parse(r io.Reader) (File, error) {
	var result File
	scanner := bufio.NewScanner(r)

	var (
		lineNumber           int
		continuationExpected bool
	)
	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		fields, err := splitLine(line)
		if err != nil {
			return result, err
		}
		if fields == nil {
			continue // skip comment or empty line
		}
		if strings.HasPrefix(line, "Zone") || continuationExpected {
			var zone ZoneLine
			if continuationExpected {
				zone, err = parseZoneContinuationLine(fields)
				if err != nil {
					return result, zoneContinuationParseError(lineNumber, line, err)
				}
			} else {
				zone, err = parseZoneLine(fields)
				if err != nil {
					return result, zoneParseError(lineNumber, line, err)
				}
			}
			result.ZoneLines = append(result.ZoneLines, zone)
			// If the UNTIL column is defined, we expect a continuation line to follow.
			continuationExpected = zone.Until.Defined
		} else if strings.HasPrefix(line, "Rule") {
			rule, err := parseRuleLine(fields)
			if err != nil {
				return result, ruleParseError(lineNumber, line, err)
			}
			result.RuleLines = append(result.RuleLines, rule)
		} else if strings.HasPrefix(line, "Link") {
			link, err := parseLinkLine(fields)
			if err != nil {
				return result, linkParseError(lineNumber, line, err)
			}
			result.LinkLines = append(result.LinkLines, link)
		} else if strings.HasPrefix(line, "Leap") {
			leap, err := parseLeapLine(fields)
			if err != nil {
				return result, leapParseError(lineNumber, line, err)
			}
			result.LeapLines = append(result.LeapLines, leap)
		} else if strings.HasPrefix(line, "Expires") {
			expires, err := parseExpiresLine(fields)
			if err != nil {
				return result, expiresParseError(lineNumber, line, err)
			}
			result.ExpiresLines = append(result.ExpiresLines, expires)
		} else {
			return result, &parseError{lineNumber, line, fmt.Errorf("unexpected line")}
		}
	}

	if err := scanner.Err(); err != nil {
		return result, fmt.Errorf("scanner: %w", err)
	}
	return result, nil
}

// LeapCorr represents the correction direction of a leap second.
type LeapCorr string

const (
	// LeapAdded means a second was added.
	LeapAdded LeapCorr = "+"
	// LeapSkipped means a second was skipped.
	LeapSkipped LeapCorr = "-"
)

// LeapLine represents a leap line.
type LeapLine struct {
	Year  int          // YEAR column
	Month time.Month   // MONTH column
	Day   int          // DAY column
	Time  HMS          // HH:MM:SS column
	Corr  LeapCorr     // CORR column
	Mode  LeapTimeMode // R/S column
}

// LeapTimeMode represents the mode of a leap second.
type LeapTimeMode int

const (
	// StationaryLeapTime means the leap second time given
	// by the other fields should be interpreted as UTC.
	StationaryLeapTime LeapTimeMode = iota

	// RollingLeapTime means the leap second time given by the
	// other fields should be interpreted as local (wall clock) time.
	//
	// The spec says:
	//
	//  Rolling leap seconds were implemented back when it was not clear
	//  whether common practice was rolling or stationary, with concerns
	//  that one would see Times Square ball drops where there'd be a
	//  “3... 2... 1... leap... Happy New Year” countdown, placing the
	//  leap second at midnight New York time rather than midnight UTC.
	//  However, this countdown style does not seem to have caught on,
	//  which means rolling leap seconds are not used in practice
	RollingLeapTime
)

// HMS represents the time that is shown on a watch.
type HMS struct {
	Hours   int
	Minutes int
	Seconds int
}

func parseLeapLine(fields []string) (LeapLine, error) {
	if len(fields) != 7 {
		return LeapLine{}, fmt.Errorf("expected 7 fields, got %d", len(fields))
	}
	if fields[0] != "Leap" {
		return LeapLine{}, fmt.Errorf("expected 'Leap', got %q", fields[0])
	}
	var (
		leap LeapLine
		errs error
		err  error
	)
	if leap.Year, err = parseLeapYEAR(fields[1]); err != nil {
		errs = errors.Join(errs, fmt.Errorf("YEAR %q: %w", fields[1], err))
	}
	if leap.Month, err = parseLeapMONTH(fields[2]); err != nil {
		errs = errors.Join(errs, fmt.Errorf("MONTH %q: %w", fields[2], err))
	}
	if leap.Day, err = parseLeapDAY(fields[3]); err != nil {
		errs = errors.Join(errs, fmt.Errorf("DAY %q: %w", fields[3], err))
	}
	if leap.Time, err = parseLeapHHMMSS(fields[4]); err != nil {
		errs = errors.Join(errs, fmt.Errorf("HH:MM:SS %q: %w", fields[4], err))
	}
	if leap.Corr, err = parseLeapCORR(fields[5]); err != nil {
		errs = errors.Join(errs, fmt.Errorf("CORR %q: %w", fields[5], err))
	}
	if leap.Mode, err = parseLeapRS(fields[6]); err != nil {
		errs = errors.Join(errs, fmt.Errorf("R/S %q: %w", fields[6], err))
	}
	return leap, errs
}

func parseLeapCORR(s string) (LeapCorr, error) {
	switch s {
	case "+":
		return LeapAdded, nil
	case "-":
		return LeapSkipped, nil
	default:
		return "", fmt.Errorf("invalid leap correction: %q", s)
	}
}

func parseLeapRS(s string) (LeapTimeMode, error) {
	l := strings.ToLower(s)
	if isAbbrev(l, "rolling", "r") {
		return RollingLeapTime, nil
	}
	if isAbbrev(l, "stationary", "s") {
		return StationaryLeapTime, nil
	}
	return 0, fmt.Errorf("invalid leap mode: %q", s)
}

// parseLeapYEAR parses the YEAR column of a leap line.
func parseLeapYEAR(s string) (int, error) {
	return strconv.Atoi(s)
}

// parseLeapMONTH parses the MONTH column of a leap line.
func parseLeapMONTH(s string) (time.Month, error) {
	return parseMonth(s)
}

// parseLeapDAY parses the DAY column of a leap line.
func parseLeapDAY(s string) (int, error) {
	return strconv.Atoi(s)
}

// parseLeapHHMMSS parses the HH:MM:SS column of a leap line.
func parseLeapHHMMSS(s string) (HMS, error) {
	return parseHMS(s)
}

// parseHMS parses a time in HH:MM:SS format.
func parseHMS(s string) (HMS, error) {
	parts := strings.Split(s, ":")
	if len(parts) != 3 {
		return HMS{}, fmt.Errorf("expected 3 parts, got %d", len(parts))
	}
	hh, err := strconv.Atoi(parts[0])
	if err != nil {
		return HMS{}, fmt.Errorf("hours: %v", err)
	}
	mm, err := strconv.Atoi(parts[1])
	if err != nil {
		return HMS{}, fmt.Errorf("minutes: %v", err)
	}
	ss, err := strconv.Atoi(parts[2])
	if err != nil {
		return HMS{}, fmt.Errorf("seconds: %v", err)
	}
	return HMS{Hours: hh, Minutes: mm, Seconds: ss}, nil
}

// ExpiresLine represents an expires line.
type ExpiresLine struct {
	Year  int
	Month time.Month
	Day   int
	Time  HMS
}

// parseExpiresLine parses an expires line.
func parseExpiresLine(fields []string) (ExpiresLine, error) {
	if len(fields) != 5 {
		return ExpiresLine{}, fmt.Errorf("expected 5 fields, got %d", len(fields))
	}
	if fields[0] != "Expires" {
		return ExpiresLine{}, fmt.Errorf("expected 'Expires', got %q", fields[0])
	}
	var (
		expires ExpiresLine
		errs    error
		err     error
	)
	if expires.Year, err = parseExpiresYEAR(fields[1]); err != nil {
		errs = errors.Join(errs, fmt.Errorf("YEAR %q: %w", fields[1], err))
	}
	if expires.Month, err = parseExpiresMONTH(fields[2]); err != nil {
		errs = errors.Join(errs, fmt.Errorf("MONTH %q: %w", fields[2], err))
	}
	if expires.Day, err = parseExpiresDAY(fields[3]); err != nil {
		errs = errors.Join(errs, fmt.Errorf("DAY %q: %w", fields[3], err))
	}
	if expires.Time, err = parseExpiresHHMMSS(fields[4]); err != nil {
		errs = errors.Join(errs, fmt.Errorf("HH:MM:SS %q: %w", fields[4], err))
	}
	return expires, errs
}

// parseExpiresYEAR parses the YEAR column of an expires line.
func parseExpiresYEAR(s string) (int, error) {
	return strconv.Atoi(s)
}

// parseExpiresMONTH parses the MONTH column of an expires line.
func parseExpiresMONTH(s string) (time.Month, error) {
	return parseMonth(s)
}

// parseExpiresDAY parses the DAY column of an expires line.
func parseExpiresDAY(s string) (int, error) {
	return strconv.Atoi(s)
}

// parseExpiresHHMMSS parses the HH:MM:SS column of an expires line.
func parseExpiresHHMMSS(s string) (HMS, error) {
	return parseHMS(s)
}

// LinkLine represents a link line.
type LinkLine struct {
	From string
	To   string
}

// parseLinkLine parses a link line.
//
// The spec says:
//
//	 A link line has the form
//
//			    Link  TARGET           LINK-NAME
//
//	 For example:
//
//			    Link  Europe/Istanbul  Asia/Istanbul
//
//	 The TARGET field should appear as the NAME field in some zone
//	 line or as the LINK-NAME field in some link line.  The LINK-NAME
//	 field is used as an alternative name for that zone; it has the
//	 same syntax as a zone line's NAME field.  Links can chain
//	 together, although the behavior is unspecified if a chain of one
//	 or more links does not terminate in a Zone name.  A link line can
//	 appear before the line that defines the link target.  For
//	 example:
func parseLinkLine(parts []string) (LinkLine, error) {
	if len(parts) != 3 {
		return LinkLine{}, fmt.Errorf("expected 3 fields, got %d", len(parts))
	}
	if parts[0] != "Link" {
		return LinkLine{}, fmt.Errorf("expected 'Link', got %q", parts[0])
	}
	return LinkLine{From: parts[1], To: parts[2]}, nil
}

// Year represents a year in the proleptic Gregorian calendar.
type Year int

func (y Year) String() string {
	if y == MinYear {
		return "<indefinite past>"
	}
	if y == MaxYear {
		return "<indefinite future>"
	}
	return strconv.Itoa(int(y))
}

const (
	// MinYear means the indefinite past.
	MinYear = math.MinInt
	// MaxYear means the indefinite future.
	MaxYear = math.MaxInt
)

// TimeForm represents the form of a time instance usually represented by a time.Duration.
type TimeForm int

func (f TimeForm) String() string {
	switch f {
	case WallClock:
		return "WallClock"
	case StandardTime:
		return "StandardTime"
	case DaylightSavingTime:
		return "DaylightSavingTime"
	case UniversalTime:
		return "UniversalTime"
	default:
		return "<UNDEFINED>"
	}
}

const (
	WallClock TimeForm = iota
	StandardTime
	DaylightSavingTime
	UniversalTime
)

// DayForm represents the form of a day in a rule or zone line.
type DayForm int

func (f DayForm) String() string {
	switch f {
	case DayFormDayNum:
		return "DayNum"
	case DayFormLast:
		return "Last"
	case DayFormAfter:
		return "After"
	case DayFormBefore:
		return "Before"
	default:
		return "<UNDEFINED>"
	}
}

const (
	DayFormDayNum DayForm = iota
	DayFormLast
	DayFormAfter
	DayFormBefore
)

// Time represents a time instance by the duration since 00:00, the start of a calendar day.
type Time struct {
	time.Duration
	Form TimeForm
}

// Day represents a day in a rule or zone line.
type Day struct {
	Form DayForm
	Num  int
	Day  time.Weekday
}

// RuleLine represents a rule line.
type RuleLine struct {
	Name   string     // The NAME field of the rule line.
	From   Year       // The FROM field of the rule line.
	To     Year       // The TO field of the rule line.
	In     time.Month // The IN field of the rule line.
	On     Day        // The ON field of the rule line.
	At     Time       // The AT field of the rule line.
	Save   Time       // The SAVE field of the rule line.
	Letter string     // The LETTER/S field of the rule line.
}

// ZoneLine represents a zone line or a continuation line.
type ZoneLine struct {
	Continuation bool          // Continuation is true if the line is a continuation line.
	Name         string        // The NAME field of the zone line. Is empty for continuation lines.
	Offset       time.Duration // The STDOFF field of the zone line.
	Rules        ZoneRules     // The RULES field of the zone line.
	Format       string        // The FORMAT field of the zone line.
	Until        Until         // The UNTIL field of the zone line.
}

// parseZoneNAME parses the NAME column of a zone line.
// It returns an error if the column is invalid according to spec.
//
// The spec says:
//
//	The name of the timezone.  This is the name used in
//	creating the time conversion information file for the
//	timezone.  It should not contain a file name component “.”
//	or “..”; a file name component is a maximal substring that
//	does not contain “/”.
func parseZoneNAME(s string) (string, error) {
	if len(s) == 0 {
		return "", fmt.Errorf("empty name")
	}
	if strings.Contains(s, ".") {
		return "", fmt.Errorf("name contains a dot: %q", s)
	}
	return s, nil
}

// parseZoneSTDOFF parses the STDOFF column of a zone line.
// It returns an error if the column is invalid according to spec.
//
// The spec says:
//
//	The amount of time to add to UT to get standard time,
//	without any adjustment for daylight saving.  This field
//	has the same format as the AT and SAVE fields of rule
//	lines, except without suffix letters; begin the field with
//	a minus sign if time must be subtracted from UT.
func parseZoneSTDOFF(s string) (time.Duration, error) {
	return parseTimeOfDay(s)
}

// ZoneRulesForm represents the type of the RULES column of a zone line.
type ZoneRulesForm int

func (f ZoneRulesForm) String() string {
	switch f {
	case ZoneRulesName:
		return "Name"
	case ZoneRulesTime:
		return "Time"
	case ZoneRulesStandard:
		return "Standard"
	default:
		return "<UNDEFINED>"
	}
}

const (
	// ZoneRulesStandard means standard time always applies because the RULES column is "-".
	ZoneRulesStandard ZoneRulesForm = iota
	// ZoneRulesName means the RULES column references rule lines by name.
	ZoneRulesName
	// ZoneRulesTime means the RULES column contains a time in rule-line SAVE column format.
	ZoneRulesTime
)

// ZoneRules represents the RULES column of a zone line.
type ZoneRules struct {
	// Form is the form of the RULES column.
	Form ZoneRulesForm
	// Name contains the name if Form is ZoneRulesName.
	Name string
	// Time contains the time if Form is ZoneRulesTime.
	Time Time
}

// parseZoneRULES parses the RULES column of a zone line.
// It returns an error if the column is invalid according to spec.
//
// The spec says:
//
//	The name of the rules that apply in the timezone or,
//	alternatively, a field in the same format as a rule-line
//	SAVE column, giving the amount of time to be added to
//	local standard time and whether the resulting time is
//	standard or daylight saving.  If this field is - then
//	standard time always applies.  When an amount of time is
//	given, only the sum of standard time and this amount
//	matters.
func parseZoneRULES(s string) (ZoneRules, error) {
	if s == "-" {
		return ZoneRules{Form: ZoneRulesStandard}, nil
	}
	if d, err := parseRuleSAVE(s); err == nil {
		return ZoneRules{Form: ZoneRulesTime, Time: d}, nil
	}
	// Assume it's a name if it's neither "-" nor a time.
	// At this point, we don't know if the name is valid,
	// because we don't have any context. Later code should
	// ensure there is a rule line with the given name.
	return ZoneRules{Form: ZoneRulesName, Name: s}, nil
}

// parseZoneFORMAT parses the FORMAT column of a zone line.
// It returns an error if the column is invalid according to spec.
//
// The spec says:
//
//	The format for time zone abbreviations.  The pair of
//	characters %s is used to show where the “variable part” of
//	the time zone abbreviation goes.  Alternatively, a format
//	can use the pair of characters %z to stand for the UT
//	offset in the form ±hh, ±hhmm, or ±hhmmss, using the
//	shortest form that does not lose information, where hh,
//	mm, and ss are the hours, minutes, and seconds east (+) or
//	west (-) of UT.  Alternatively, a slash (/) separates
//	standard and daylight abbreviations.  To conform to POSIX,
//	a time zone abbreviation should contain only alphanumeric
//	ASCII characters, “+” and “-”.  By convention, the time
//	zone abbreviation “-00” is a placeholder that means local
//	time is unspecified.
func parseZoneFORMAT(s string) (string, error) {
	if len(s) == 0 {
		return "", fmt.Errorf("empty format")
	}
	unquoted, _ := unquote(s)
	return unquoted, nil
}

// UntilPartsMask is a bitmask of the parts that are defined in the UNTIL column of a zone line.
// It is used to track which fields of the Until struct are defined and which should "default to
// the earliest possible value for the missing fields" as per spec.
type UntilPartsMask uint8

// Has returns true if all the parts in the mask are set.
func (p UntilPartsMask) Has(parts UntilPartsMask) bool {
	return p&parts != 0
}

// Set sets the parts in the mask.
func (p UntilPartsMask) Set(parts UntilPartsMask) UntilPartsMask {
	return p | parts
}

const (
	// UntilUndefined is the zero value of UntilPartsMask.
	UntilUndefined = UntilPartsMask(0)

	// If a part is set, all parts to the right are also set.
	// For example, if UntilTime is set, UntilDay, UntilMonth, and UntilYear are also set.
	// This is because the spec says that trailing fields can be omitted, and default to the earliest possible value for the missing fields.
	// The *Only parts are used internally to set the correct parts when parsing the UNTIL column.
	untilYearOnly UntilPartsMask = 1 << iota
	untilMonthOnly
	untilDayOnly
	untilTimeOnly

	// UntilYear indicates that Until.Year is defined. This is always set if Until.Defined is true.
	UntilYear = untilYearOnly
	// UntilMonth indicates that Until.Month is defined.
	UntilMonth = untilYearOnly | untilMonthOnly
	// UntilDay indicates that Until.Day is defined.
	UntilDay = untilYearOnly | untilMonthOnly | untilDayOnly
	// UntilTime indicates that Until.Time is defined.
	UntilTime = untilYearOnly | untilMonthOnly | untilDayOnly | untilTimeOnly
)

// Until represents the UNTIL column of a zone line.
// The empty value is a zero value of the struct, which means the UNTIL column is not defined.
type Until struct {
	// Set to true if the UNTIL column is defined.
	Defined bool
	// Parts is a bitmask of the parts that are defined.
	Parts UntilPartsMask
	// Year is the year in the UNTIL column.
	// It is always defined if Defined is true.
	Year int
	// Month is the month in the UNTIL column.
	// It is defined if Parts.Has(UntilMonth) is true.
	Month time.Month
	// Day is the day in the UNTIL column.
	// It is defined if Parts.Has(UntilDay) is true.
	Day Day
	// Time is the time in the UNTIL column.
	// It is defined if Parts.Has(UntilTime) is true.
	Time Time
}

// parseZoneUNTIL parses the UNTIL column of a zone line.
// It returns an error if the column is invalid according to spec.
//
// The spec says:
//
//	The time at which the UT offset or the rule(s) change for
//	a location.  It takes the form of one to four fields YEAR
//	[MONTH [DAY [TIME]]].  If this is specified, the time zone
//	information is generated from the given UT offset and rule
//	change until the time specified, which is interpreted
//	using the rules in effect just before the transition.  The
//	month, day, and time of day have the same format as the
//	IN, ON, and AT fields of a rule; trailing fields can be
//	omitted, and default to the earliest possible value for
//	the missing fields.
func parseZoneUNTIL(s string) (Until, error) {
	if len(s) == 0 {
		// UNTIL column is optional.
		return Until{}, nil
	}

	var u Until
	parts := strings.Fields(s)
	if len(parts) > 4 {
		return u, fmt.Errorf("too many fields: %d", len(parts))
	}

	// Parse year.
	year, err := strconv.Atoi(parts[0])
	if err != nil {
		return u, fmt.Errorf("year: %v", err)
	}
	u.Year = year
	u.Parts = u.Parts.Set(untilYearOnly)

	// Parse month, if present.
	if len(parts) > 1 {
		month, err := parseRuleIN(parts[1])
		if err != nil {
			return u, fmt.Errorf("month: %v", err)
		}
		u.Month = month
		u.Parts = u.Parts.Set(untilMonthOnly)
	}

	// Parse day, if present.
	if len(parts) > 2 {
		day, err := parseRuleON(parts[2])
		if err != nil {
			return u, fmt.Errorf("day: %v", err)
		}
		u.Day = day
		u.Parts = u.Parts.Set(untilDayOnly)
	}

	// Parse time, if present.
	if len(parts) > 3 {
		t, err := parseRuleAT(parts[3])
		if err != nil {
			return u, fmt.Errorf("time: %v", err)
		}
		u.Time = t
		u.Parts = u.Parts.Set(untilTimeOnly)
	}

	// If we did not return early, the UNTIL column is defined.
	u.Defined = true
	return u, nil
}

// parseZoneLine parses a zone line.
//
// The spec says:
//
//	 A zone line has the form
//
//			    Zone  NAME        STDOFF  RULES   FORMAT  [UNTIL]
//
//	 For example:
//
//			    Zone  Asia/Amman  2:00    Jordan  EE%sT   2017 Oct 27 01:00
func parseZoneLine(fields []string) (ZoneLine, error) {
	if len(fields) < 5 {
		return ZoneLine{}, fmt.Errorf("expected at least 5 fields, got %d", len(fields))
	}
	if len(fields) > 9 {
		return ZoneLine{}, fmt.Errorf("expected at most 9 fields, got %d", len(fields))
	}
	if fields[0] != "Zone" {
		return ZoneLine{}, fmt.Errorf("expected 'Zone', got %q", fields[0])
	}
	var (
		z    ZoneLine
		errs error
		err  error
	)
	if z.Name, err = parseZoneNAME(fields[1]); err != nil {
		errs = errors.Join(errs, fmt.Errorf("NAME %q: %w", fields[1], err))
	}
	if z.Offset, err = parseZoneSTDOFF(fields[2]); err != nil {
		errs = errors.Join(errs, fmt.Errorf("STDOFF %q: %w", fields[2], err))
	}
	if z.Rules, err = parseZoneRULES(fields[3]); err != nil {
		errs = errors.Join(errs, fmt.Errorf("RULES %q: %w", fields[3], err))
	}
	if z.Format, err = parseZoneFORMAT(fields[4]); err != nil {
		errs = errors.Join(errs, fmt.Errorf("FORMAT %q: %w", fields[4], err))
	}
	if len(fields) > 5 {
		until := strings.Join(fields[5:], " ")
		if z.Until, err = parseZoneUNTIL(until); err != nil {
			errs = errors.Join(errs, fmt.Errorf("UNTIL %q: %w", fields[5], err))
		}
	}
	return z, errs
}

// parseZoneContinuationLine parses a zone continuation line.
//
// The spec says:
//
//	[It] has the
//	same form as a zone line except that the string “Zone” and
//	the name are omitted, as the continuation line will place
//	information starting at the time specified as the “until”
//	information in the previous line in the file used by the
//	previous line.  Continuation lines may contain “until”
//	information, just as zone lines do, indicating that the
//	next line is a further continuation.
func parseZoneContinuationLine(fields []string) (ZoneLine, error) {
	if len(fields) < 3 {
		return ZoneLine{}, fmt.Errorf("expected at least 3 fields, got %d", len(fields))
	}
	if len(fields) > 7 {
		return ZoneLine{}, fmt.Errorf("expected at most 7 fields, got %d", len(fields))
	}
	var (
		z    ZoneLine
		errs error
		err  error
	)
	z.Continuation = true // Continuation lines are always continuation lines.
	if z.Offset, err = parseZoneSTDOFF(fields[0]); err != nil {
		errs = errors.Join(errs, fmt.Errorf("STDOFF %q: %w", fields[0], err))
	}
	if z.Rules, err = parseZoneRULES(fields[1]); err != nil {
		errs = errors.Join(errs, fmt.Errorf("RULES %q: %w", fields[1], err))
	}
	if z.Format, err = parseZoneFORMAT(fields[2]); err != nil {
		errs = errors.Join(errs, fmt.Errorf("FORMAT %q: %w", fields[2], err))
	}
	if len(fields) > 3 {
		until := strings.Join(fields[3:], " ")
		if z.Until, err = parseZoneUNTIL(until); err != nil {
			errs = errors.Join(errs, fmt.Errorf("UNTIL %q: %w", fields[2], err))
		}
	}
	return z, errs
}

// parseRuleLine parses a rule line.
//
// The spec says:
//
//	A rule line has the form
//
//	    Rule  NAME  FROM  TO    -  IN   ON       AT     SAVE   LETTER/S
//
//	For example:
//
//	    Rule  US    1967  1973  -  Apr  lastSun  2:00w  1:00d  D
func parseRuleLine(fields []string) (RuleLine, error) {
	if len(fields) != 10 {
		return RuleLine{}, fmt.Errorf("expected 10 fields, got %d", len(fields))
	}
	if fields[0] != "Rule" {
		return RuleLine{}, fmt.Errorf("expected 'Rule', got %q", fields[0])
	}
	var (
		r    RuleLine
		errs error
		err  error
	)
	if r.Name, err = parseRuleNAME(fields[1]); err != nil {
		errs = errors.Join(errs, fmt.Errorf("NAME %q: %w", fields[1], err))
	}
	if r.From, err = parseRuleFROM(fields[2]); err != nil {
		errs = errors.Join(errs, fmt.Errorf("FROM %q: %w", fields[2], err))
	}
	if r.To, err = parseRuleTO(fields[3], r.From); err != nil {
		errs = errors.Join(errs, fmt.Errorf("TO %q: %w", fields[3], err))
	}
	if r.In, err = parseRuleIN(fields[5]); err != nil {
		errs = errors.Join(errs, fmt.Errorf("IN %q: %w", fields[5], err))
	}
	if r.On, err = parseRuleON(fields[6]); err != nil {
		errs = errors.Join(errs, fmt.Errorf("ON %q: %w", fields[6], err))
	}
	if r.At, err = parseRuleAT(fields[7]); err != nil {
		errs = errors.Join(errs, fmt.Errorf("AT %q: %w", fields[7], err))
	}
	if r.Save, err = parseRuleSAVE(fields[8]); err != nil {
		errs = errors.Join(errs, fmt.Errorf("SAVE %q: %w", fields[8], err))
	}
	if r.Letter, err = parseRuleLETTERS(fields[9]); err != nil {
		errs = errors.Join(errs, fmt.Errorf("LETTER/S %q: %w", fields[9], err))
	}
	return r, errs
}

// splitLine splits a line according to the spec.
// It returns nil if the line is a comment or empty.
//
// The spec says:
//
//	Input lines are made up of fields.  Fields are separated from one
//	another by one or more white space characters.  The white space
//	characters are space, form feed, carriage return, newline, tab,
//	and vertical tab.  Leading and trailing white space on input
//	lines is ignored.  An unquoted sharp character (#) in the input
//	introduces a comment which extends to the end of the line the
//	sharp character appears on.  White space characters and sharp
//	characters may be enclosed in double quotes (") if they're to be
//	used as part of a field.  Any line that is blank (after comment
//	stripping) is ignored.  Nonblank lines are expected to be of one
//	of three types: rule lines, zone lines, and link lines.
func splitLine(line string) ([]string, error) {
	// Remove comments.
	i := strings.Index(line, "#")
	if i != -1 {
		line = line[:i]
	}
	// Remove leading and trailing white space.
	line = strings.TrimSpace(line)

	if len(line) == 0 {
		return nil, nil
	}

	return strings.Fields(line), nil

	//// Split fields, respecting quoted fields.
	//var fields []string
	//for len(line) > 0 {
	//	if line[0] == '"' {
	//		// Quoted field.
	//		i := strings.Index(line[1:], `"`)
	//		if i == -1 {
	//			// No closing quote.
	//			return nil, fmt.Errorf("no closing quote: %q", line)
	//		}
	//		fields = append(fields, line[1:i+1])
	//		line = line[i+2:]
	//	} else {
	//		// Unquoted field.
	//		i := strings.IndexAny(line, " \f\r\n\t\v")
	//		if i == -1 {
	//			// Last field.
	//			fields = append(fields, line)
	//			break
	//		}
	//		fields = append(fields, line[:i])
	//		line = line[i+1:]
	//	}
	//}
	//return fields, nil
}

// parseRuleNAME parses the NAME column of a rule.
// It returns an error if the name is invalid according to spec.
//
// The spec says:
//
//	Gives the name of the rule set that contains this line.
//	The name must start with a character that is neither an
//	ASCII digit nor “-” nor “+”.  To allow for future
//	extensions, an unquoted name should not contain characters
//	from the set “!$%&'()*,/:;<=>?@[\]^`{|}~”.
func parseRuleNAME(s string) (string, error) {
	if len(s) == 0 {
		return "", fmt.Errorf("empty name")
	}
	if s[0] >= '0' && s[0] <= '9' {
		return "", fmt.Errorf("name starts with a digit: %q", s)
	}
	if s[0] == '-' || s[0] == '+' {
		return "", fmt.Errorf("name starts with a sign: %q", s)
	}

	unquoted, wasQuoted := unquote(s)
	if !wasQuoted {
		// unquoted name
		if containsSpecialChar(s) {
			return "", fmt.Errorf("name contains special character: %q", s)
		}
	}
	return unquoted, nil
}

// containsSpecialChar returns true if the string contains any of the special characters
func containsSpecialChar(s string) bool {
	specialChars := "!$%&'()*,/:;<=>?@[\\]^`{|}~"
	for _, char := range specialChars {
		if strings.ContainsRune(s, char) {
			return true
		}
	}
	return false
}

// unquote removes quotes from a string.
// It returns the unquoted string and true if the string was quoted.
// Otherwise, it returns the original string and false.
func unquote(s string) (string, bool) {
	if s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1], true
	}
	return s, false
}

// parseRuleFROM parses the FROM columns a rule.
// It returns an error if the year is invalid according to spec.
//
// The spec says:
//
//	Gives the first year in which the rule applies.  Any
//	signed integer year can be supplied; the proleptic
//	Gregorian calendar is assumed, with year 0 preceding year
//	1.  The word minimum (or an abbreviation) means the
//	indefinite past.  The word maximum (or an abbreviation)
//	means the indefinite future.  Rules can describe times
//	that are not representable as time values, with the
//	unrepresentable times ignored; this allows rules to be
//	portable among hosts with differing time value types.
func parseRuleFROM(s string) (Year, error) {
	if isAbbrev(s, "minimum", "mi") {
		return MinYear, nil
	}
	if isAbbrev(s, "maximum", "ma") {
		return MaxYear, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	return Year(n), nil
}

// parseRuleTO parses the TO columns of a rule.
// It returns an error if the year is invalid according to spec.
//
// The spec says:
//
//	Gives the final year in which the rule applies.  In
//	addition to minimum and maximum (as above), the word only
//	(or an abbreviation) may be used to repeat the value of
//	the FROM field.
func parseRuleTO(s string, from Year) (Year, error) {
	if isAbbrev(s, "minimum", "mi") {
		return MinYear, nil
	}
	if isAbbrev(s, "maximum", "ma") {
		return MaxYear, nil
	}
	if isAbbrev(s, "only", "o") {
		return from, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	return Year(n), nil
}

// parseRuleON parses the IN columns of a rule.
// It returns an error if the day is invalid according to spec.
//
// The spec says:
//
//	Names the month in which the rule takes effect.  Month
//	names may be abbreviated.
func parseRuleIN(s string) (time.Month, error) {
	return parseMonth(s)
}

func parseMonth(s string) (time.Month, error) {
	if len(s) < 3 {
		return 0, fmt.Errorf("month %q: too short", s)
	}
	l := strings.ToLower(s)
	if isAbbrev(l, "january", "jan") {
		return time.January, nil
	}
	if isAbbrev(l, "february", "feb") {
		return time.February, nil
	}
	if isAbbrev(l, "march", "mar") {
		return time.March, nil
	}
	if isAbbrev(l, "april", "apr") {
		return time.April, nil
	}
	if isAbbrev(l, "may", "may") {
		return time.May, nil
	}
	if isAbbrev(l, "june", "jun") {
		return time.June, nil
	}
	if isAbbrev(l, "july", "jul") {
		return time.July, nil
	}
	if isAbbrev(l, "august", "aug") {
		return time.August, nil
	}
	if isAbbrev(l, "september", "sep") {
		return time.September, nil
	}
	if isAbbrev(l, "october", "oct") {
		return time.October, nil
	}
	if isAbbrev(l, "november", "nov") {
		return time.November, nil
	}
	if isAbbrev(l, "december", "dec") {
		return time.December, nil
	}
	return 0, fmt.Errorf("month %q: invalid", s)
}

// parseRuleON parses the ON columns of a rule.
// It returns an error if the day is invalid according to spec.
//
// The spec says:
//
//	Gives the day on which the rule takes effect.  Recognized
//	forms include:
//
//	     5        the fifth of the month
//	     lastSun  the last Sunday in the month
//	     lastMon  the last Monday in the month
//	     Sun>=8   first Sunday on or after the eighth
//	     Sun<=25  last Sunday on or before the 25th
//
//	A weekday name (e.g., Sunday) or a weekday name preceded
//	by “last” (e.g., lastSunday) may be abbreviated or spelled
//	out in full.  There must be no white space characters
//	within the ON field.  The “<=” and “>=” constructs can
//	result in a day in the neighboring month; for example, the
//	IN-ON combination “Oct Sun>=31” stands for the first
//	Sunday on or after October 31, even if that Sunday occurs
//	in November.
func parseRuleON(s string) (Day, error) {
	if n, err := strconv.Atoi(s); err == nil {
		return Day{Form: DayFormDayNum, Num: n}, nil
	}
	if strings.HasPrefix(s, "last") {
		day, err := parseWeekday(s[4:])
		if err != nil {
			return Day{}, err
		}
		return Day{Form: DayFormLast, Day: day}, nil
	}
	if strings.Contains(s, "=") {
		form := DayFormBefore
		parts := strings.Split(s, "<=")
		if len(parts) != 2 {
			form = DayFormAfter
			parts = strings.Split(s, ">=")
		}
		if len(parts) != 2 || len(parts[0]) == 0 || len(parts[1]) == 0 {
			return Day{}, fmt.Errorf("expected weekday<=dayofmonth or weekday>=dayofmonth")
		}
		day, err := parseWeekday(parts[0])
		if err != nil {
			return Day{}, fmt.Errorf("left part of comparison %q: %w", parts[0], err)
		}
		n, err := strconv.Atoi(parts[1])
		if err != nil {
			return Day{}, fmt.Errorf("right part of comparison %q: %w", parts[1], err)
		}
		return Day{Form: form, Day: day, Num: n}, nil
	}
	return Day{}, fmt.Errorf("invalid")
}

// parseRuleAT parses the AT column of a rule.
// It returns an error if the time is invalid according to spec.
//
// The spec says:
//
//	Gives the time of day at which the rule takes effect,
//	relative to 00:00, the start of a calendar day.
//	Recognized forms include:
//
//	     2            time in hours
//	     2:00         time in hours and minutes
//	     01:28:14     time in hours, minutes, and seconds
//	     00:19:32.13  time with fractional seconds
//	     12:00        midday, 12 hours after 00:00
//	     15:00        3 PM, 15 hours after 00:00
//	     24:00        end of day, 24 hours after 00:00
//	     260:00       260 hours after 00:00
//	     -2:30        2.5 hours before 00:00
//	     -            equivalent to 0
//
//	Although zic rounds times to the nearest integer second
//	(breaking ties to the even integer), the fractions may be
//	useful to other applications requiring greater precision.
//	The source format does not specify any maximum precision.
//	Any of these forms may be followed by the letter w if the
//	given time is local or “wall clock” time, s if the given
//	time is standard time without any adjustment for daylight
//	saving, or u (or g or z) if the given time is universal
//	time; in the absence of an indicator, local (wall clock)
//	time is assumed.  These forms ignore leap seconds; for
//	example, if a leap second occurs at 00:59:60 local time,
//	“1:00” stands for 3601 seconds after local midnight
//	instead of the usual 3600 seconds.  The intent is that a
//	rule line describes the instants when a clock/calendar set
//	to the type of time specified in the AT field would show
//	the specified date and time of day.
func parseRuleAT(s string) (Time, error) {
	d, suffix, err := parseTimeOfDayWithSuffix(s, []string{"w", "s", "u", "g", "z"})
	if err != nil {
		return Time{}, err
	}
	var form TimeForm
	switch suffix {
	case "w":
		form = WallClock
	case "s":
		form = StandardTime
	case "u", "g", "z":
		form = UniversalTime
	default:
		form = WallClock
	}
	return Time{Duration: d, Form: form}, nil
}

// parseRuleSAVE parses the SAVE columns of a rule.
// It returns an error if the time is invalid according to spec.
//
// The spec says:
//
//	Gives the amount of time to be added to local standard
//	time when the rule is in effect, and whether the resulting
//	time is standard or daylight saving.  This field has the
//	same format as the AT field except with a different set of
//	suffix letters: s for standard time and d for daylight
//	saving time.  The suffix letter is typically omitted, and
//	defaults to s if the offset is zero and to d otherwise.
//	Negative offsets are allowed; in Ireland, for example,
//	daylight saving time is observed in winter and has a
//	negative offset relative to Irish Standard Time.  The
//	offset is merely added to standard time; for example, zic
//	does not distinguish a 10:30 standard time plus an 0:30
//	SAVE from a 10:00 standard time plus a 1:00 SAVE.
func parseRuleSAVE(s string) (Time, error) {
	d, suffix, err := parseTimeOfDayWithSuffix(s, []string{"s", "d"})
	if err != nil {
		return Time{}, err
	}
	var form TimeForm
	switch suffix {
	case "s":
		form = StandardTime
	case "d":
		form = DaylightSavingTime
	default:
		if d == 0 {
			form = StandardTime
		} else {
			form = DaylightSavingTime
		}
	}
	return Time{Duration: d, Form: form}, nil
}

// parseRuleLETTERS parses the LETTER/S columns of a rule.
// It returns an error if the letter is invalid according to spec.
//
// The spec says:
//
//	Gives the “variable part” (for example, the “S” or “D” in
//	“EST” or “EDT”) of time zone abbreviations to be used when
//	this rule is in effect.  If this field is “-”, the
//	variable part is null.
func parseRuleLETTERS(s string) (string, error) {
	if len(s) == 0 {
		return "", fmt.Errorf("empty letter")
	}
	if s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}
	if s == "-" {
		return "", nil
	}
	return s, nil
}

func parseTimeOfDayWithSuffix(timeStr string, suffixes []string) (time.Duration, string, error) {
	for _, suffix := range suffixes {
		if strings.HasSuffix(timeStr, suffix) {
			woSuffix := strings.TrimSuffix(timeStr, suffix)
			d, err := parseTimeOfDay(woSuffix)
			if err != nil {
				return 0, "", err
			}
			return d, suffix, nil
		}
	}
	d, err := parseTimeOfDay(timeStr)
	if err != nil {
		return 0, "", err
	}
	return d, "", nil
}

// parseTimeOfDay parses the at time of a rule.
// The returned time.Duration is the time of day, relative to 00:00, the start of a calendar day.
// It returns an error if the time is invalid according to spec.
//
// The spec says:
//
//	  Recognized forms include:
//
//		     2            time in hours
//		     2:00         time in hours and minutes
//		     01:28:14     time in hours, minutes, and seconds
//		     00:19:32.13  time with fractional seconds
//		     12:00        midday, 12 hours after 00:00
//		     15:00        3 PM, 15 hours after 00:00
//		     24:00        end of day, 24 hours after 00:00
//		     260:00       260 hours after 00:00
//		     -2:30        2.5 hours before 00:00
//		     -            equivalent to 0
func parseTimeOfDay(s string) (time.Duration, error) {
	if s == "-" {
		return 0, nil // Equivalent to 0 duration.
	}

	// Handle negative time.
	isNegative := strings.HasPrefix(s, "-")
	if isNegative {
		s = strings.TrimPrefix(s, "-")
	}

	// Split the time into components.
	parts := strings.Split(s, ":")
	var hours, minutes, seconds, fractional int
	var err error

	// Parse hours.
	hours, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, fmt.Errorf("invalid hour format: %v", err)
	}

	// Parse minutes, if present.
	if len(parts) > 1 {
		minutes, err = strconv.Atoi(parts[1])
		if err != nil {
			return 0, fmt.Errorf("invalid minute format: %v", err)
		}
	}

	// Parse seconds and fractional seconds, if present.
	if len(parts) > 2 {
		secondParts := strings.Split(parts[2], ".")
		seconds, err = strconv.Atoi(secondParts[0])
		if err != nil {
			return 0, fmt.Errorf("invalid second format: %v", err)
		}
		if len(secondParts) > 1 {
			// Convert fractional seconds to milliseconds.
			fractionalStr := secondParts[1]
			// Pad or truncate to 3 digits (milliseconds).
			if len(fractionalStr) > 3 {
				fractionalStr = fractionalStr[:3]
			} else {
				for len(fractionalStr) < 3 {
					fractionalStr += "0"
				}
			}
			fractional, err = strconv.Atoi(fractionalStr)
			if err != nil {
				return 0, fmt.Errorf("invalid fractional second format: %v", err)
			}
		}
	}

	// Calculate total duration.
	totalDuration := time.Duration(hours)*time.Hour +
		time.Duration(minutes)*time.Minute +
		time.Duration(seconds)*time.Second +
		time.Duration(fractional)*time.Millisecond

	if isNegative {
		totalDuration = -totalDuration
	}

	return totalDuration, nil
}

func parseWeekday(s string) (time.Weekday, error) {
	l := strings.ToLower(s)
	if isAbbrev(l, "sunday", "su") {
		return time.Sunday, nil
	}
	if isAbbrev(l, "monday", "m") {
		return time.Monday, nil
	}
	if isAbbrev(l, "tuesday", "tu") {
		return time.Tuesday, nil
	}
	if isAbbrev(l, "wednesday", "w") {
		return time.Wednesday, nil
	}
	if isAbbrev(l, "thursday", "th") {
		return time.Thursday, nil
	}
	if isAbbrev(l, "friday", "f") {
		return time.Friday, nil
	}
	if isAbbrev(l, "saturday", "sa") {
		return time.Saturday, nil
	}
	return 0, fmt.Errorf("invalid weekday %q", s)
}

func isAbbrev(s string, long string, min string) bool {
	return strings.HasPrefix(s, min) && strings.HasPrefix(long, s)
}
