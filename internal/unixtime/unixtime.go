package unixtime

// From converts a given date and time to a Unix timestamp, i.e. the number of seconds since 1970-01-01 00:00:00 UTC.
// It ignores leap seconds but respects leap years. It assumes the proleptic Gregorian calendar.
// This implementation is based on the Go standard library's time package but does not depend on time.Location.
// Depending on time.Location feels weird for a function that is supposed to be a low-level utility for creating
// timezone data which is then used by time.Location.
func FromDateTime(year int, month int, day int, hour int, minute int, second int) int64 {
	daysSinceStartOfYear := []uint64{0, 31, 59, 90, 120, 151, 181, 212, 243, 273, 304, 334}

	d := daysSinceEpoch(year) + daysSinceStartOfYear[month-1] + (uint64(day) - 1)
	if month > 2 && (year%4 == 0 && (year%100 != 0 || year%400 == 0)) {
		d++ // +leap year
	}
	abs := d*secondsPerDay + uint64(hour)*secondsPerHour + uint64(minute)*secondsPerMinute + uint64(second)
	unix := int64(abs) + (absoluteToInternal + internalToUnix)
	return unix
}

// The constants were copied from time.go in the Go standard library's time package.
const (
	secondsPerMinute = 60
	secondsPerHour   = 60 * secondsPerMinute
	secondsPerDay    = 24 * secondsPerHour
	daysPer400Years  = 365*400 + 97
	daysPer100Years  = 365*100 + 24
	daysPer4Years    = 365*4 + 1

	absoluteZeroYear         = -292277022399
	internalYear             = 1
	absoluteToInternal int64 = (absoluteZeroYear - internalYear) * 365.2425 * secondsPerDay
	unixToInternal     int64 = (1969*365 + 1969/4 - 1969/100 + 1969/400) * secondsPerDay
	internalToUnix     int64 = -unixToInternal
)

// daysSinceEpoch takes a year and returns the number of days from
// the absolute epoch to the start of that year.
// This is basically (year - zeroYear) * 365, but accounting for leap days.
//
// This function was copied from time.go in the Go standard library time package.
func daysSinceEpoch(year int) uint64 {
	y := uint64(int64(year) - absoluteZeroYear)

	// Add in days from 400-year cycles.
	n := y / 400
	y -= 400 * n
	d := daysPer400Years * n

	// Add in 100-year cycles.
	n = y / 100
	y -= 100 * n
	d += daysPer100Years * n

	// Add in 4-year cycles.
	n = y / 4
	y -= 4 * n
	d += daysPer4Years * n

	// Add in non-leap years.
	n = y
	d += 365 * n

	return d
}
