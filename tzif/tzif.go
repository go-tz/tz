// Package tzif implements the TZif file format according to RFC8536.
// https://datatracker.ietf.org/doc/html/rfc8536
package tzif

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

// NOTE: All multi-octet integer values MUST be stored in network octet
// order format (high-order octet first, otherwise known as big-endian),
// with all bits significant.  Signed integer values MUST be represented
// using two's complement.
var order = binary.BigEndian

// Version represents the version of a TZif file.
// The version is an octet identifying the version of the file's format.
// In V1, time values are 32bit (four-octets) and in V2 upwards time values are 64bit (eight-octets).
// Therefore, V1DataBlock is only used by V1 and V2DataBlock is used by V2, V3 and V4.
type Version byte

func (v Version) String() string {
	switch v {
	case V1:
		return "V1 (0x00)"
	case V2:
		return "V2 (0x32)"
	case V3:
		return "V3 (0x33)"
	case V4:
		return "V4 (0x34)"
	default:
		return fmt.Sprintf("<undefined version (%d)>", v)
	}
}

const (
	// V1 represents a version 1 TZif file.
	//
	// NUL (0x00)  Version 1 - The file contains only the version 1
	// header and data block.  Version 1 files MUST NOT contain a
	// version 2+ header, data block, or footer.
	V1 Version = 0x00
	// V2 represents a version 2 TZif file.
	//
	// '2' (0x32)  Version 2 - The file MUST contain the version 1 header
	// and data block, a version 2+ header and data block, and a
	// footer.  The TZ string in the footer (Section 3.3), if
	// nonempty, MUST strictly adhere to the requirements for the TZ
	// environment variable as defined in Section 8.3 of the "Base
	// Definitions" volume of [POSIX] and MUST encode the POSIX
	// portable character set as ASCII.
	V2 Version = 0x32
	// V3 represents a version 3 TZif file.
	//
	// '3' (0x33)  Version 3 - The file MUST contain the version 1 header
	// and data block, a version 2+ header and data block, and a
	// footer.  The TZ string in the footer (Section 3.3), if
	// nonempty, MUST conform to POSIX requirements with ASCII
	// encoding, except that it MAY use the TZ string extensions
	// described in Section 3.3.1 of RFC8536.
	V3 Version = 0x33 // '3'
	// V4 represents a version 4 TZif file.
	// It is not specified in RFC8536 as of Feb 2019, but is specified in the tzfile(5) man page.
	//
	// The man page says:
	//
	//  For version-4-format TZif files, the first leap second record can
	//  have a correction that is neither +1 nor -1, to represent
	//  truncation of the TZif file at the start.  Also, if two or more
	//  leap second transitions are present and the last entry's
	//  correction equals the previous one, the last entry denotes the
	//  expiration of the leap second table instead of a leap second;
	//  timestamps after this expiration are unreliable in that future
	//  releases will likely add leap second entries after the
	//  expiration, and the added leap seconds will change how post-
	//  expiration timestamps are treated.
	V4 Version = 0x34 // '4'
)

// Magic is the four-octet ASCII sequence "TZif" (0x54 0x5A 0x69 0x66),
// which identifies the file as utilizing the Time Zone Information Format.
var Magic = [4]byte{'T', 'Z', 'i', 'f'}

// Header is the header of a TZif file.
//
// A TZif header is structured as follows (the lengths of multi-octet
// fields are shown in parentheses):
//
//	+---------------+---+
//	|  magic    (4) |ver|
//	+---------------+---+---------------------------------------+
//	|           [unused - reserved for future use] (15)         |
//	+---------------+---------------+---------------+-----------+
//	|  isutcnt  (4) |  isstdcnt (4) |  leapcnt  (4) |
//	+---------------+---------------+---------------+
//	|  timecnt  (4) |  typecnt  (4) |  charcnt  (4) |
//	+---------------+---------------+---------------+
type Header struct {
	// Version is an octet identifying the version of the file's format.
	Version Version
	// Reserved for future use.
	Reserved [15]byte

	// Isutcnt is a four-octet unsigned integer specifying the number of UT/
	// local indicators contained in the data block -- MUST either be
	// zero or equal to "typecnt".
	Isutcnt uint32

	// Isstdcnt is a four-octet unsigned integer specifying the number of
	// standard/wall indicators contained in the data block -- MUST
	// either be zero or equal to "typecnt".
	Isstdcnt uint32

	// Leapcnt is a four-octet unsigned integer specifying the number of
	// leap-second records contained in the data block.
	Leapcnt uint32

	// Timecnt is a four-octet unsigned integer specifying the number of
	// transition times contained in the data block.
	Timecnt uint32

	// Typecnt is a four-octet unsigned integer specifying the number of
	// local time type records contained in the data block -- MUST NOT be
	// zero.  (Although local time type records convey no useful
	// information in files that have nonempty TZ strings but no
	// transitions, at least one such record is nevertheless required
	// because many TZif readers reject files that have zero time types.)
	Typecnt uint32

	// Charcnt is a four-octet unsigned integer specifying the total number
	// of octets used by the set of time zone designations contained in
	// the data block - MUST NOT be zero.  The count includes the
	// trailing NUL (0x00) octet at the end of the last time zone
	// designation.
	Charcnt uint32
}

// Write writes the Header to w.
func (h Header) Write(w io.Writer) error {
	if _, err := w.Write(Magic[:]); err != nil {
		return err
	}
	return binary.Write(w, order, h)
}

func ReadHeader(r io.Reader) (Header, error) {
	var h Header
	magic := make([]byte, len(Magic))
	if err := binary.Read(r, order, &magic); err != nil {
		return h, fmt.Errorf("reading magic: %w", err)
	}
	if !bytes.Equal(magic, Magic[:]) {
		return h, fmt.Errorf("invalid magic: %v", magic)
	}
	err := binary.Read(r, order, &h)
	return h, err
}

// V1DataBlock is the data block of a version 1 TZif file.
// The data block is structured as follows with TIME_SIZE being 4:
//
//	+---------------------------------------------------------+
//	|  transition times          (timecnt x TIME_SIZE)        |
//	+---------------------------------------------------------+
//	|  transition types          (timecnt)                    |
//	+---------------------------------------------------------+
//	|  local time type records   (typecnt x 6)                |
//	+---------------------------------------------------------+
//	|  time zone designations    (charcnt)                    |
//	+---------------------------------------------------------+
//	|  leap-second records       (leapcnt x (TIME_SIZE + 4))  |
//	+---------------------------------------------------------+
//	|  standard/wall indicators  (isstdcnt)                   |
//	+---------------------------------------------------------+
//	|  UT/local indicators       (isutcnt)                    |
//	+---------------------------------------------------------+
type V1DataBlock struct {
	// TransitionTimes is a series of four-octet UNIX leap-time
	// values sorted in strictly ascending order.  Each value is used as
	// a transition time at which the rules for computing local time may
	// change.  The number of time values is specified by the "timecnt"
	// field in the header.  Each time value SHOULD be at least -2**59.
	// (-2**59 is the greatest negated power of 2 that predates the Big
	// Bang, and avoiding earlier timestamps works around known TZif
	// reader bugs relating to outlandishly negative timestamps.)
	TransitionTimes []int32

	// TransitionTypes is a series of one-octet unsigned integers specifying
	// the type of local time of the corresponding transition time.
	// These values serve as zero-based indices into the array of local
	// time type records.  The number of type indices is specified by the
	// "timecnt" field in the header.  Each type index MUST be in the
	// range [0, "typecnt" - 1].
	TransitionTypes []uint8

	// LocalTimeTypeRecord is a series of six-octet records specifying a
	// local time type.  The number of records is specified by the
	// "typecnt" field in the header.
	LocalTimeTypeRecord []LocalTimeTypeRecord

	// TimeZoneDesignation is a series of octets constituting an array of
	// NUL-terminated (0x00) time zone designation strings.  The total
	// number of octets is specified by the "charcnt" field in the
	// header.  Note that two designations MAY overlap if one is a suffix
	// of the other.  The character encoding of time zone designation
	// strings is not specified; however, see Section 4 of this document.
	TimeZoneDesignation []byte

	// LeapSecondRecords is a series of eight-octet records
	// specifying the corrections that need to be applied to UTC in order
	// to determine TAI.  The records are sorted by the occurrence time
	// in strictly ascending order.  The number of records is specified
	// by the "leapcnt" field in the header.  Each record has one of the
	// following structures (the lengths of multi-octet fields are shown
	// in parentheses):
	LeapSecondRecords []V1LeapSecondRecord

	// StandardWallIndicators is a series of one-octet values indicating
	// whether the transition times associated with local time types were
	// specified as standard time or wall-clock time.  Each value MUST be
	// 0 or 1.  A value of one (1) indicates standard time.  The value
	// MUST be set to one (1) if the corresponding UT/local indicator is
	// set to one (1).  A value of zero (0) indicates wall time.  The
	// number of values is specified by the "isstdcnt" field in the
	// header.  If "isstdcnt" is zero (0), all transition times
	// associated with local time types are assumed to be specified as
	// wall time.
	StandardWallIndicators []bool

	// UTLocalIndicators is a series of one-octet values indicating whether
	// the transition times associated with local time types were
	// specified as UT or local time.  Each value MUST be 0 or 1.  A
	// value of one (1) indicates UT, and the corresponding standard/wall
	// indicator MUST also be set to one (1).  A value of zero (0)
	// indicates local time.  The number of values is specified by the
	// "isutcnt" field in the header.  If "isutcnt" is zero (0), all
	// transition times associated with local time types are assumed to
	// be specified as local time.
	UTLocalIndicators []bool
}

func (b V1DataBlock) Write(w io.Writer) error {
	if err := binary.Write(w, order, b.TransitionTimes); err != nil {
		return err
	}
	if err := binary.Write(w, order, b.TransitionTypes); err != nil {
		return err
	}
	for _, r := range b.LocalTimeTypeRecord {
		if err := r.Write(w); err != nil {
			return err
		}
	}
	if _, err := w.Write(b.TimeZoneDesignation); err != nil {
		return err
	}
	for _, r := range b.LeapSecondRecords {
		if err := binary.Write(w, order, r); err != nil {
			return err
		}
	}
	for _, r := range b.StandardWallIndicators {
		if err := binary.Write(w, order, r); err != nil {
			return err
		}
	}
	for _, r := range b.UTLocalIndicators {
		if err := binary.Write(w, order, r); err != nil {
			return err
		}
	}
	return nil
}

func ReadV1DataBlock(r io.Reader, h Header) (V1DataBlock, error) {
	var b V1DataBlock
	if h.Timecnt > 0 {
		b.TransitionTimes = make([]int32, h.Timecnt)
		if err := binary.Read(r, order, &b.TransitionTimes); err != nil {
			return b, fmt.Errorf("reading transition times: %w", err)
		}
	}
	if h.Timecnt > 0 {
		b.TransitionTypes = make([]uint8, h.Timecnt)
		if err := binary.Read(r, order, &b.TransitionTypes); err != nil {
			return b, fmt.Errorf("reading transition types: %w", err)
		}
	}
	if h.Typecnt > 0 {
		b.LocalTimeTypeRecord = make([]LocalTimeTypeRecord, h.Typecnt)
		for i := range b.LocalTimeTypeRecord {
			if err := binary.Read(r, order, &b.LocalTimeTypeRecord[i]); err != nil {
				return b, fmt.Errorf("reading local time type record: %w", err)
			}
		}
	}
	if h.Charcnt > 0 {
		b.TimeZoneDesignation = make([]byte, h.Charcnt)
		if _, err := r.Read(b.TimeZoneDesignation); err != nil {
			return b, fmt.Errorf("reading time zone designation: %w", err)
		}
	}
	if h.Leapcnt > 0 {
		b.LeapSecondRecords = make([]V1LeapSecondRecord, h.Leapcnt)
		for i := range b.LeapSecondRecords {
			if err := binary.Read(r, order, &b.LeapSecondRecords[i]); err != nil {
				return b, fmt.Errorf("reading leap second record: %w", err)
			}
		}
	}
	if h.Isstdcnt > 0 {
		b.StandardWallIndicators = make([]bool, h.Isstdcnt)
		for i := range b.StandardWallIndicators {
			if err := binary.Read(r, order, &b.StandardWallIndicators[i]); err != nil {
				return b, fmt.Errorf("reading standard/wall indicator: %w", err)
			}
		}
	}
	if h.Isutcnt > 0 {
		b.UTLocalIndicators = make([]bool, h.Isutcnt)
		for i := range b.UTLocalIndicators {
			if err := binary.Read(r, order, &b.UTLocalIndicators[i]); err != nil {
				return b, fmt.Errorf("reading UT/local indicator: %w", err)
			}
		}
	}
	return b, nil
}

// V1LeapSecondRecord represents a leap-second record for a V1DataBlock.
// Each record has the following format (the lengths of multi-octet fields
// are shown in parentheses):
//
//	+---------------+---------------+
//	|  occur (4)    |  corr (4)     |
//	+---------------+---------------+
type V1LeapSecondRecord struct {
	// Occur is a four-octet UNIX leap time value
	// specifying the time at which a leap-second correction occurs.
	// The first value, if present, MUST be nonnegative, and each
	// later value MUST be at least 2419199 greater than the previous
	// value.  (This is 28 days' worth of seconds, minus a potential
	// negative leap second.)
	Occur int32

	// Corr is a four-octet signed integer specifying the value of
	// LEAPCORR on or after the occurrence.  The correction value in
	// the first leap-second record, if present, MUST be either one
	// (1) or minus one (-1).  The correction values in adjacent leap-
	// second records MUST differ by exactly one (1).  The value of
	// LEAPCORR is zero for timestamps that occur before the
	// occurrence time in the first leap-second record (or for all
	// timestamps if there are no leap-second records).
	Corr int32
}

func (r V1LeapSecondRecord) Write(w io.Writer) error {
	if err := binary.Write(w, order, r.Occur); err != nil {
		return err
	}
	return binary.Write(w, order, r.Corr)
}

// V2DataBlock is the data block of a version 2+ TZif file.
// V2, V3 and V4 files all use the V2DataBlock as the only difference
// to V1 is the size of time values.
// The data block is structured as follows with TIME_SIZE being 8:
//
//	+---------------------------------------------------------+
//	|  transition times          (timecnt x TIME_SIZE)        |
//	+---------------------------------------------------------+
//	|  transition types          (timecnt)                    |
//	+---------------------------------------------------------+
//	|  local time type records   (typecnt x 6)                |
//	+---------------------------------------------------------+
//	|  time zone designations    (charcnt)                    |
//	+---------------------------------------------------------+
//	|  leap-second records       (leapcnt x (TIME_SIZE + 4))  |
//	+---------------------------------------------------------+
//	|  standard/wall indicators  (isstdcnt)                   |
//	+---------------------------------------------------------+
//	|  UT/local indicators       (isutcnt)                    |
//	+---------------------------------------------------------+
type V2DataBlock struct {
	// TransitionTimes is a series of eight-octet UNIX leap-time
	// values sorted in strictly ascending order.  Each value is used as
	// a transition time at which the rules for computing local time may
	// change.  The number of time values is specified by the "timecnt"
	// field in the header.  Each time value SHOULD be at least -2**59.
	// (-2**59 is the greatest negated power of 2 that predates the Big
	// Bang, and avoiding earlier timestamps works around known TZif
	// reader bugs relating to outlandishly negative timestamps.)
	TransitionTimes []int64

	// TransitionTypes is a series of one-octet unsigned integers specifying
	// the type of local time of the corresponding transition time.
	// These values serve as zero-based indices into the array of local
	// time type records.  The number of type indices is specified by the
	// "timecnt" field in the header.  Each type index MUST be in the
	// range [0, "typecnt" - 1].
	TransitionTypes []uint8

	// LocalTimeTypeRecord is a series of six-octet records specifying a
	// local time type.  The number of records is specified by the
	// "typecnt" field in the header.  Each record has the following
	// format (the lengths of multi-octet fields are shown in
	// parentheses):
	LocalTimeTypeRecord []LocalTimeTypeRecord

	// TimeZoneDesignation is a series of octets constituting an array of
	// NUL-terminated (0x00) time zone designation strings.  The total
	// number of octets is specified by the "charcnt" field in the
	// header.  Note that two designations MAY overlap if one is a suffix
	// of the other.  The character encoding of time zone designation
	// strings is not specified; however, see Section 4 of this document.
	TimeZoneDesignation []byte

	// LeapSecondRecords is a series of eight-octet records
	// specifying the corrections that need to be applied to UTC in order
	// to determine TAI.  The records are sorted by the occurrence time
	// in strictly ascending order.  The number of records is specified
	// by the "leapcnt" field in the header.  Each record has one of the
	// following structures (the lengths of multi-octet fields are shown
	// in parentheses):
	LeapSecondRecords []V2LeapSecondRecord

	// StandardWallIndicators is a series of one-octet values indicating
	// whether the transition times associated with local time types were
	// specified as standard time or wall-clock time.  Each value MUST be
	// 0 or 1.  A value of one (1) indicates standard time.  The value
	// MUST be set to one (1) if the corresponding UT/local indicator is
	// set to one (1).  A value of zero (0) indicates wall time.  The
	// number of values is specified by the "isstdcnt" field in the
	// header.  If "isstdcnt" is zero (0), all transition times
	// associated with local time types are assumed to be specified as
	// wall time.
	StandardWallIndicators []bool

	// UTLocalIndicators is a series of one-octet values indicating whether
	// the transition times associated with local time types were
	// specified as UT or local time.  Each value MUST be 0 or 1.  A
	// value of one (1) indicates UT, and the corresponding standard/wall
	// indicator MUST also be set to one (1).  A value of zero (0)
	// indicates local time.  The number of values is specified by the
	// "isutcnt" field in the header.  If "isutcnt" is zero (0), all
	// transition times associated with local time types are assumed to
	// be specified as local time.
	UTLocalIndicators []bool
}

func (b V2DataBlock) Write(w io.Writer) error {
	if err := binary.Write(w, order, b.TransitionTimes); err != nil {
		return err
	}
	if err := binary.Write(w, order, b.TransitionTypes); err != nil {
		return err
	}
	for _, r := range b.LocalTimeTypeRecord {
		if err := r.Write(w); err != nil {
			return err
		}
	}
	if _, err := w.Write(b.TimeZoneDesignation); err != nil {
		return err
	}
	for _, r := range b.LeapSecondRecords {
		if err := binary.Write(w, order, r); err != nil {
			return err
		}
	}
	for _, r := range b.StandardWallIndicators {
		if err := binary.Write(w, order, r); err != nil {
			return err
		}
	}
	for _, r := range b.UTLocalIndicators {
		if err := binary.Write(w, order, r); err != nil {
			return err
		}
	}
	return nil
}

func ReadV2DataBlock(r io.Reader, h Header) (V2DataBlock, error) {
	if h.Version < V2 {
		return V2DataBlock{}, fmt.Errorf("invalid header version: %v", h.Version)
	}

	var b V2DataBlock
	if h.Timecnt > 0 {
		b.TransitionTimes = make([]int64, h.Timecnt)
		if err := binary.Read(r, order, &b.TransitionTimes); err != nil {
			return b, fmt.Errorf("reading transition times: %w", err)
		}
	}
	if h.Timecnt > 0 {
		b.TransitionTypes = make([]uint8, h.Timecnt)
		if err := binary.Read(r, order, &b.TransitionTypes); err != nil {
			return b, fmt.Errorf("reading transition types: %w", err)
		}
	}
	if h.Typecnt > 0 {
		b.LocalTimeTypeRecord = make([]LocalTimeTypeRecord, h.Typecnt)
		for i := range b.LocalTimeTypeRecord {
			if err := binary.Read(r, order, &b.LocalTimeTypeRecord[i]); err != nil {
				return b, fmt.Errorf("reading local time type record: %w", err)
			}
		}
	}
	if h.Charcnt > 0 {
		b.TimeZoneDesignation = make([]byte, h.Charcnt)
		if _, err := r.Read(b.TimeZoneDesignation); err != nil {
			return b, fmt.Errorf("reading time zone designation: %w", err)
		}
	}
	if h.Leapcnt > 0 {
		b.LeapSecondRecords = make([]V2LeapSecondRecord, h.Leapcnt)
		for i := range b.LeapSecondRecords {
			if err := binary.Read(r, order, &b.LeapSecondRecords[i]); err != nil {
				return b, fmt.Errorf("reading leap second record: %w", err)
			}
		}
	}
	if h.Isstdcnt > 0 {
		b.StandardWallIndicators = make([]bool, h.Isstdcnt)
		for i := range b.StandardWallIndicators {
			if err := binary.Read(r, order, &b.StandardWallIndicators[i]); err != nil {
				return b, fmt.Errorf("reading standard/wall indicator: %w", err)
			}
		}
	}
	if h.Isutcnt > 0 {
		b.UTLocalIndicators = make([]bool, h.Isutcnt)
		for i := range b.UTLocalIndicators {
			if err := binary.Read(r, order, &b.UTLocalIndicators[i]); err != nil {
				return b, fmt.Errorf("reading UT/local indicator: %w", err)
			}
		}
	}
	return b, nil
}

// V2LeapSecondRecord represents a leap-second record for a V2DataBlock.
// Each record has the following format (the lengths of multi-octet fields
// are shown in parentheses):
//
//	+---------------+---------------+---------------+
//	|  occur (8)                    |  corr (4)     |
//	+---------------+---------------+---------------+
type V2LeapSecondRecord struct {
	// Occur is a eight-octet UNIX leap time value
	// specifying the time at which a leap-second correction occurs.
	// The first value, if present, MUST be nonnegative, and each
	// later value MUST be at least 2419199 greater than the previous
	// value.  (This is 28 days' worth of seconds, minus a potential
	// negative leap second.)
	Occur int64

	// Corr is a four-octet signed integer specifying the value of
	// LEAPCORR on or after the occurrence.  The correction value in
	// the first leap-second record, if present, MUST be either one
	// (1) or minus one (-1).  The correction values in adjacent leap-
	// second records MUST differ by exactly one (1).  The value of
	// LEAPCORR is zero for timestamps that occur before the
	// occurrence time in the first leap-second record (or for all
	// timestamps if there are no leap-second records).
	Corr int32
}

func (r V2LeapSecondRecord) Write(w io.Writer) error {
	if err := binary.Write(w, order, r.Occur); err != nil {
		return err
	}
	return binary.Write(w, order, r.Corr)
}

// LocalTimeTypeRecord represents a local time type record.
// Each record has the following format (the lengths of multi-octet fields
// are shown in parentheses):
//
//	+---------------+---+---+
//	|  utoff (4)    |dst|idx|
//	+---------------+---+---+
type LocalTimeTypeRecord struct {
	// Utoff is a four-octet signed integer specifying the number of
	// seconds to be added to UT in order to determine local time.
	// The value MUST NOT be -2**31 and SHOULD be in the range
	// [-89999, 93599] (i.e., its value SHOULD be more than -25 hours
	// and less than 26 hours).  Avoiding -2**31 allows 32-bit clients
	// to negate the value without overflow.  Restricting it to
	// [-89999, 93599] allows easy support by implementations that
	// already support the POSIX-required range [-24:59:59, 25:59:59].
	Utoff int32

	// Dst is a one-octet value indicating whether local time should
	// be considered Daylight Saving Time (DST).  The value MUST be 0
	// or 1.  A value of one (1) indicates that this type of time is
	// DST.  A value of zero (0) indicates that this time type is
	// standard time.
	Dst bool

	// Idx is a one-octet unsigned integer specifying a zero-based
	// index into the series of time zone designation octets, thereby
	// selecting a particular designation string.  Each index MUST be
	// in the range [0, "charcnt" - 1]; it designates the
	// NUL-terminated string of octets starting at position "idx" in
	// the time zone designations.  (This string MAY be empty.)  A NUL
	// octet MUST exist in the time zone designations at or after
	// position "idx".
	Idx uint8
}

func (r LocalTimeTypeRecord) Write(w io.Writer) error {
	if err := binary.Write(w, order, r.Utoff); err != nil {
		return err
	}
	if err := binary.Write(w, order, r.Dst); err != nil {
		return err
	}
	return binary.Write(w, order, r.Idx)
}

// Footer represents the footer of a TZif file.
// The footer is structured as follows (the lengths of multi-octet
// fields are shown in parentheses):
//
//	+---+--------------------+---+
//	| NL|  TZ string (0...)  |NL |
//	+---+--------------------+---+
type Footer struct {
	// TZString contains a rule for computing local time changes after the last
	// transition time stored in the version 2+ data block.  The string
	// is either empty or uses the expanded format of the "TZ"
	// environment variable as defined in Section 8.3 of the "Base
	// Definitions" volume of [POSIX] with ASCII encoding, possibly
	// utilizing extensions described below (Section 3.3.1) in version 3
	// files.  If the string is empty, the corresponding information is
	// not available.  If the string is nonempty and one or more
	// transitions appear in the version 2+ data, the string MUST be
	// consistent with the last version 2+ transition.  In other words,
	// evaluating the TZ string at the time of the last transition should
	// yield the same time type as was specified in the last transition.
	// The string MUST NOT contain NUL octets or be NUL-terminated, and
	// it SHOULD NOT begin with the ':' (colon) character.
	TZString []byte
}

var asciiNewLine = byte(0x0A)

func (f Footer) Write(w io.Writer) error {
	if _, err := w.Write([]byte{asciiNewLine}); err != nil {
		return err
	}
	if _, err := w.Write(f.TZString); err != nil {
		return err
	}
	_, err := w.Write([]byte{asciiNewLine})
	return err
}

func ReadFooter(r io.Reader) (Footer, error) {
	var f Footer
	buf := make([]byte, 1)
	if _, err := r.Read(buf); err != nil {
		return f, fmt.Errorf("reading newline: %w", err)
	}
	if buf[0] != asciiNewLine {
		return f, fmt.Errorf("expected newline: %v", buf[0])
	}
	var b []byte
	for {
		if _, err := r.Read(buf); err != nil {
			return f, fmt.Errorf("reading TZ string: %w", err)
		}
		if buf[0] == asciiNewLine {
			break
		}
		b = append(b, buf[0])
	}
	f.TZString = b
	return f, nil
}
