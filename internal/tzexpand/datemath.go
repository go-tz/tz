package tzexpand

// isLeapYear determines if the year is a leap year.
func isLeapYear(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

// daysInMonth returns the number of days in a given month for a specific year.
func daysInMonth(month, year int) int {
	if month == 2 {
		if isLeapYear(year) {
			return 29
		}
		return 28
	}
	if month == 4 || month == 6 || month == 9 || month == 11 {
		return 30
	}
	return 31
}

// calculateDayOfWeek calculates the day of the week for a given date,
// where 0=Sunday, 1=Monday, ..., 6=Saturday.
func calculateDayOfWeek(day, month, year int) int {
	// Zeller's Congruence algorithm adjustment for Gregorian calendar
	if month < 3 {
		month += 12
		year -= 1
	}
	k := year % 100
	j := year / 100
	h := (day + ((13 * (month + 1)) / 5) + k + (k / 4) + (j / 4) + (5 * j)) % 7
	// Adjust result to fit Sunday=0, Monday=1, ..., Saturday=6
	return (h + 6) % 7
}

// lastWeekdayOfMonth finds the last instance of a given weekday in a specific month and year.
func lastWeekdayOfMonth(year, month, weekday int) int {
	lastDay := daysInMonth(month, year)
	lastDayWeekday := calculateDayOfWeek(lastDay, month, year)

	// Calculate how many days to subtract from the last day to get the last instance of the given weekday.
	offset := (lastDayWeekday - weekday + 7) % 7
	return lastDay - offset
}

// nextWeekday calculates the next occurrence of a weekday on or after a given day in the specified month and year,
// accounting for overflow into the next month or year. Returns a tuple of (year, month, day).
func nextWeekday(year, month, day, targetWeekday int) (int, int, int) {
	dayOfWeek := calculateDayOfWeek(day, month, year)
	diff := targetWeekday - dayOfWeek
	if diff < 0 {
		diff += 7 // Ensure a positive difference
	}

	nextOccurrence := day + diff
	daysInCurrentMonth := daysInMonth(month, year)

	// Check if the next occurrence overflows into the next month
	if nextOccurrence > daysInCurrentMonth {
		nextOccurrence -= daysInCurrentMonth // Calculate the day in the next month
		month += 1                           // Move to the next month
		if month > 12 {                      // Check for year change
			month = 1
			year += 1
		}
	}

	return year, month, nextOccurrence
}

// lastWeekday finds the last occurrence of a given weekday before or on a given day in the specified month and year,
// accounting for overflow into the previous month or year. Returns a tuple of (year, month, day).
func lastWeekday(year, month, day, targetWeekday int) (int, int, int) {
	dayOfWeek := calculateDayOfWeek(day, month, year)
	diff := dayOfWeek - targetWeekday
	if diff < 0 {
		diff += 7 // Ensure a positive difference
	}

	lastOccurrence := day - diff
	if lastOccurrence < 1 { // Check if the last occurrence overflows into the previous month
		month -= 1     // Move to the previous month
		if month < 1 { // Check for year change
			month = 12
			year -= 1
		}
		daysInPreviousMonth := daysInMonth(month, year)
		lastOccurrence = daysInPreviousMonth + lastOccurrence // Calculate the day in the previous month
	}

	return year, month, lastOccurrence
}
