package tzif

import (
	"errors"
	"fmt"
)

func Validate(d Data) error {
	var errs []error
	if d.Version != d.V1Header.Version || d.V1Header.Version != d.V2Header.Version {
		errs = append(errs, fmt.Errorf("inconsistent version: file = %v, v1 header = %v, v2 header = %v", d.Version, d.V1Header.Version, d.V2Header.Version))
	}

	if err := validateV1(d); err != nil {
		errs = append(errs, err...)
	}

	if d.Version > V1 {
		if err := validateV2(d); err != nil {
			errs = append(errs, err...)
		}
	}

	return errors.Join(errs...)
}

func validateV1(d Data) []error {
	var (
		err    []error
		data   = d.V1Data
		header = d.V1Header
	)

	// Isutcnt
	if header.Isutcnt != 0 && header.Isutcnt != header.Typecnt {
		err = append(err, fmt.Errorf("invalid v1 isutcnt (%d): must be 0 or equal to typecnt (%d)", header.Isutcnt, header.Typecnt))
	}
	if len(data.UTLocalIndicators) != int(header.Isutcnt) {
		err = append(err, fmt.Errorf("invalid v1 isutcnt: header = %d, data = %d", header.Isutcnt, len(data.UTLocalIndicators)))
	}

	// Isstdcnt
	if header.Isstdcnt != 0 && header.Isstdcnt != header.Typecnt {
		err = append(err, fmt.Errorf("invalid 1 isstdcnt (%d): must be 0 or equal to typecnt (%d)", header.Isstdcnt, header.Typecnt))
	}
	if len(data.StandardWallIndicators) != int(header.Isstdcnt) {
		err = append(err, fmt.Errorf("invalid v1 isstdcnt: header = %d, data = %d", header.Isstdcnt, len(data.StandardWallIndicators)))
	}

	// Leapcnt
	if len(data.LeapSecondRecords) != int(header.Leapcnt) {
		err = append(err, fmt.Errorf("invalid v1 leapcnt: header = %d, data = %d", header.Leapcnt, len(data.LeapSecondRecords)))
	}

	// Timecnt
	if len(data.TransitionTimes) != int(header.Timecnt) {
		err = append(err, fmt.Errorf("invalid v1 timecnt: header = %d, transition times = %d", header.Timecnt, len(data.TransitionTimes)))
	}
	if times, types := len(data.TransitionTimes), len(data.TransitionTypes); times != types {
		err = append(err, fmt.Errorf("inconsistent v1 transitions: transition times = %d, transition types = %d", times, types))
	}

	// Typecnt
	if header.Typecnt == 0 {
		err = append(err, fmt.Errorf("invalid v1 typecnt: must not be zero"))
	}
	if len(data.LocalTimeTypeRecord) != int(header.Typecnt) {
		err = append(err, fmt.Errorf("invalid v1 typecnt: header = %d, data = %d", header.Typecnt, len(data.LocalTimeTypeRecord)))
	}

	// Charcnt
	if header.Charcnt == 0 {
		err = append(err, fmt.Errorf("invalid v1 charcnt: must not be zero"))
	}
	if len(data.TimeZoneDesignation) != int(header.Charcnt) {
		err = append(err, fmt.Errorf("invalid v1 charcnt: header = %d, data = %d", header.Charcnt, len(data.TimeZoneDesignation)))
	}
	if header.Charcnt > 0 && data.TimeZoneDesignation[len(data.TimeZoneDesignation)-1] != 0 {
		err = append(err, fmt.Errorf("invalid v1 time zone designations: missing null terminator"))
	}
	return err
}

func validateV2(d Data) []error {
	var (
		err    []error
		data   = d.V2Data
		header = d.V2Header
	)

	// Isutcnt
	if header.Isutcnt != 0 && header.Isutcnt != header.Typecnt {
		err = append(err, fmt.Errorf("invalid v2 isutcnt (%d): must be 0 or equal to typecnt (%d)", header.Isutcnt, header.Typecnt))
	}
	if len(data.UTLocalIndicators) != int(header.Isutcnt) {
		err = append(err, fmt.Errorf("invalid v2 isutcnt: header = %d, data = %d", header.Isutcnt, len(data.UTLocalIndicators)))
	}

	// Isstdcnt
	if header.Isstdcnt != 0 && header.Isstdcnt != header.Typecnt {
		err = append(err, fmt.Errorf("invalid 1 isstdcnt (%d): must be 0 or equal to typecnt (%d)", header.Isstdcnt, header.Typecnt))
	}
	if len(data.StandardWallIndicators) != int(header.Isstdcnt) {
		err = append(err, fmt.Errorf("invalid v2 isstdcnt: header = %d, data = %d", header.Isstdcnt, len(data.StandardWallIndicators)))
	}

	// Leapcnt
	if len(data.LeapSecondRecords) != int(header.Leapcnt) {
		err = append(err, fmt.Errorf("invalid v2 leapcnt: header = %d, data = %d", header.Leapcnt, len(data.LeapSecondRecords)))
	}

	// Timecnt
	if len(data.TransitionTimes) != int(header.Timecnt) {
		err = append(err, fmt.Errorf("invalid v2 timecnt: header = %d, transition times = %d", header.Timecnt, len(data.TransitionTimes)))
	}
	if times, types := len(data.TransitionTimes), len(data.TransitionTypes); times != types {
		err = append(err, fmt.Errorf("inconsistent v2 transitions: transition times = %d, transition types = %d", times, types))
	}

	// Typecnt
	if header.Typecnt == 0 {
		err = append(err, fmt.Errorf("invalid v2 typecnt: must not be zero"))
	}
	if len(data.LocalTimeTypeRecord) != int(header.Typecnt) {
		err = append(err, fmt.Errorf("invalid v2 typecnt: header = %d, data = %d", header.Typecnt, len(data.LocalTimeTypeRecord)))
	}

	// Charcnt
	if header.Charcnt == 0 {
		err = append(err, fmt.Errorf("invalid v2 charcnt: must not be zero"))
	}
	if len(data.TimeZoneDesignation) != int(header.Charcnt) {
		err = append(err, fmt.Errorf("invalid v2 charcnt: header = %d, data = %d", header.Charcnt, len(data.TimeZoneDesignation)))
	}
	if header.Charcnt > 0 && data.TimeZoneDesignation[len(data.TimeZoneDesignation)-1] != 0 {
		err = append(err, fmt.Errorf("invalid v2 time zone designations: missing null terminator"))
	}
	return err
}
