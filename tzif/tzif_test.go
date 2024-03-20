package tzif

import (
	"bytes"
	"github.com/google/go-cmp/cmp"
	"strings"
	"testing"
)

func TestHeader_Write(t *testing.T) {
	buf := bytes.Buffer{}
	header := Header{
		Isutcnt:  1,
		Isstdcnt: 2,
		Leapcnt:  3,
		Timecnt:  4,
		Typecnt:  5,
		Charcnt:  6,
	}
	if err := header.Write(&buf); err != nil {
		t.Fatalf("WriteV1() failed: %v", err)
	}
	got := buf.Bytes()
	want := []byte{
		// 4 bytes magic
		'T', 'Z', 'i', 'f',
		// 1 byte version
		0,
		// 15 bytes reserved
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		0, 0, 0, 0, 0,
		// 6 4-byte integers
		0, 0, 0, 1, // isutcnt
		0, 0, 0, 2, // isstdcnt
		0, 0, 0, 3, // leapcnt
		0, 0, 0, 4, // timecnt
		0, 0, 0, 5, // typecnt
		0, 0, 0, 6, // charcnt
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("WriteV1() mismatch (-got +want):\n%s", diff)
	}
}

func TestV1FileRepresentingUTCWithLeapSeconds(t *testing.T) {
	// This is the example B.1. from RFC 8536.
	header := Header{
		Version:  V1,
		Reserved: [15]byte{},
		Isutcnt:  1,
		Isstdcnt: 1,
		Leapcnt:  27,
		Timecnt:  0,
		Typecnt:  1,
		Charcnt:  4,
	}
	block := V1DataBlock{
		TransitionTimes: nil,
		TransitionTypes: nil,
		LocalTimeTypeRecord: []LocalTimeTypeRecord{
			{
				Utoff: 0,
				Dst:   false,
				Idx:   0,
			},
		},
		TimeZoneDesignation: []byte("UTC\x00"),
		LeapSecondRecords: []V1LeapSecondRecord{
			{78796800, 1},
			{94694401, 2},
			{126230402, 3},
			{157766403, 4},
			{189302404, 5},
			{220924805, 6},
			{252460806, 7},
			{283996807, 8},
			{315532808, 9},
			{362793609, 10},
			{394329610, 11},
			{425865611, 12},
			{489024012, 13},
			{567993613, 14},
			{631152014, 15},
			{662688015, 16},
			{709948816, 17},
			{741484817, 18},
			{773020818, 19},
			{820454419, 20},
			{867715220, 21},
			{915148821, 22},
			{1136073622, 23},
			{1230768023, 24},
			{1341100824, 25},
			{1435708825, 26},
			{1483228826, 27},
		},
		StandardWallIndicators: []bool{false},
		UTLocalIndicators:      []bool{false},
	}

	var buf bytes.Buffer
	if err := header.Write(&buf); err != nil {
		t.Fatalf("write header: %v", err)
	}
	if err := block.Write(&buf); err != nil {
		t.Fatalf("write block: %v", err)
	}
	got := buf.Bytes()

	want := []byte{
		0x54, 0x5a, 0x69, 0x66, // magic
		0x00, // version
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x01, // isutcnt
		0x00, 0x00, 0x00, 0x01, // isstdcnt
		0x00, 0x00, 0x00, 0x1b, // leapcnt
		0x00, 0x00, 0x00, 0x00, // timecnt
		0x00, 0x00, 0x00, 0x01, // typecnt
		0x00, 0x00, 0x00, 0x04, // charcnt
		// localtimetype[0]
		0x00, 0x00, 0x00, 0x00, // utcoff
		0x00,                   // isdst
		0x00,                   // desigidx
		0x55, 0x54, 0x43, 0x00, // "designation[0]"
		// leapsecond[0]
		0x04, 0xb2, 0x58, 0x00, // occurrence
		0x00, 0x00, 0x00, 0x01, // correction
		// leapsecond[1]
		0x05, 0xa4, 0xec, 0x01, // occurrence
		0x00, 0x00, 0x00, 0x02, // correction
		// leapsecond[2]
		0x07, 0x86, 0x1f, 0x82, // occurrence
		0x00, 0x00, 0x00, 0x03, // correction
		// leapsecond[3]
		0x09, 0x67, 0x53, 0x03, // occurrence
		0x00, 0x00, 0x00, 0x04, // correction
		// leapsecond[4]
		0x0b, 0x48, 0x86, 0x84, // occurrence
		0x00, 0x00, 0x00, 0x05, // correction
		// leapsecond[5]
		0x0d, 0x2b, 0x0b, 0x85, // occurrence
		0x00, 0x00, 0x00, 0x06, // correction
		// leapsecond[6]
		0x0f, 0x0c, 0x3f, 0x06, // occurrence
		0x00, 0x00, 0x00, 0x07, // correction
		// leapsecond[7]
		0x10, 0xed, 0x72, 0x87, // occurrence
		0x00, 0x00, 0x00, 0x08, // correction
		// leapsecond[8]
		0x12, 0xce, 0xa6, 0x08, // occurrence
		0x00, 0x00, 0x00, 0x09, // correction
		// leapsecond[9]
		0x15, 0x9f, 0xca, 0x89, // occurrence
		0x00, 0x00, 0x00, 0x0a, // correction
		// leapsecond[10]
		0x17, 0x80, 0xfe, 0x0a, // occurrence
		0x00, 0x00, 0x00, 0x0b, // correction
		// leapsecond[11]
		0x19, 0x62, 0x31, 0x8b, // occurrence
		0x00, 0x00, 0x00, 0x0c, // correction
		// leapsecond[12]
		0x1d, 0x25, 0xea, 0x0c, // occurrence
		0x00, 0x00, 0x00, 0x0d, // correction
		// leapsecond[13]
		0x21, 0xda, 0xe5, 0x0d, // occurrence
		0x00, 0x00, 0x00, 0x0e, // correction
		// leapsecond[14]
		0x25, 0x9e, 0x9d, 0x8e, // occurrence
		0x00, 0x00, 0x00, 0x0f, // correction
		// leapsecond[15]
		0x27, 0x7f, 0xd1, 0x0f, // occurrence
		0x00, 0x00, 0x00, 0x10, // correction
		// leapsecond[16]
		0x2a, 0x50, 0xf5, 0x90, // occurrence
		0x00, 0x00, 0x00, 0x11, // correction
		// leapsecond[17]
		0x2c, 0x32, 0x29, 0x11, // occurrence
		0x00, 0x00, 0x00, 0x12, // correction
		// leapsecond[18]
		0x2e, 0x13, 0x5c, 0x92, // occurrence
		0x00, 0x00, 0x00, 0x13, // correction
		// leapsecond[19]
		0x30, 0xe7, 0x24, 0x13, // occurrence
		0x00, 0x00, 0x00, 0x14, // correction
		// leapsecond[20]
		0x33, 0xb8, 0x48, 0x94, // occurrence
		0x00, 0x00, 0x00, 0x15, // correction
		// leapsecond[21]
		0x36, 0x8c, 0x10, 0x15, // occurrence
		0x00, 0x00, 0x00, 0x16, // correction
		// leapsecond[22]
		0x43, 0xb7, 0x1b, 0x96, // occurrence
		0x00, 0x00, 0x00, 0x17, // correction
		// leapsecond[23]
		0x49, 0x5c, 0x07, 0x97, // occurrence
		0x00, 0x00, 0x00, 0x18, // correction
		// leapsecond[24]
		0x4f, 0xef, 0x93, 0x18, // occurrence
		0x00, 0x00, 0x00, 0x19, // correction
		// leapsecond[25]
		0x55, 0x93, 0x2d, 0x99, // occurrence
		0x00, 0x00, 0x00, 0x1a, // correction
		// leapsecond[26]
		0x58, 0x68, 0x46, 0x9a, // occurrence
		0x00, 0x00, 0x00, 0x1b, // correction
		0x00, // UT/local[0]
		0x00, // standard/wall[0]
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("WriteV1() mismatch (-got +want):\n%s", diff)
	}

	// Check that we can decode the file we just encoded.
	f, err := DecodeFile(bytes.NewReader(want))
	if err != nil {
		t.Fatalf("DecodeFile() failed: %v", err)
	}
	if diff := cmp.Diff(f.V1Header, header); diff != "" {
		t.Errorf("DecodeFile() V1Header mismatch (-got +want):\n%s", diff)
	}
	if diff := cmp.Diff(f.V1Data, block); diff != "" {
		t.Errorf("DecodeFile() V1Block mismatch (-got +want):\n%s", diff)
	}
}

func TestV2FileRepresentingPacificHonululu(t *testing.T) {
	// This is the example B.2. from RFC 8536.
	v1Header := Header{
		Version:  V1,
		Isutcnt:  6,
		Isstdcnt: 6,
		Leapcnt:  0,
		Timecnt:  7,
		Typecnt:  6,
		Charcnt:  20,
	}
	v1Block := V1DataBlock{
		TransitionTimes: []int32{
			-2147483648,
			-1157283000,
			-1155436200,
			-880198200,
			-769395600,
			-765376200,
			-712150200,
		},
		TransitionTypes: []uint8{1, 2, 1, 3, 4, 1, 5},
		LocalTimeTypeRecord: []LocalTimeTypeRecord{
			{Utoff: -37886, Dst: false, Idx: 0},
			{Utoff: -37800, Dst: false, Idx: 4},
			{Utoff: -34200, Dst: true, Idx: 8},
			{Utoff: -34200, Dst: true, Idx: 12},
			{Utoff: -34200, Dst: true, Idx: 16},
			{Utoff: -36000, Dst: false, Idx: 4},
		},
		TimeZoneDesignation: []byte(strings.Join([]string{
			"LMT\x00",
			"HST\x00",
			"HDT\x00",
			"HWT\x00",
			"HPT\x00"},
			"")),
		UTLocalIndicators: []bool{
			true, false, false, false, true, false,
		},
		StandardWallIndicators: []bool{
			true, false, false, false, true, false,
		},
	}
	v2Header := Header{
		Version:  V2,
		Isutcnt:  6,
		Isstdcnt: 6,
		Leapcnt:  0,
		Timecnt:  7,
		Typecnt:  6,
		Charcnt:  20,
	}
	v2Block := V2DataBlock{
		TransitionTimes: []int64{
			-2334101314,
			-1157283000,
			-1155436200,
			-880198200,
			-769395600,
			-765376200,
			-712150200,
		},
		TransitionTypes: []uint8{1, 2, 1, 3, 4, 1, 5},
		LocalTimeTypeRecord: []LocalTimeTypeRecord{
			{Utoff: -37886, Dst: false, Idx: 0},
			{Utoff: -37800, Dst: false, Idx: 4},
			{Utoff: -34200, Dst: true, Idx: 8},
			{Utoff: -34200, Dst: true, Idx: 12},
			{Utoff: -34200, Dst: true, Idx: 16},
			{Utoff: -36000, Dst: false, Idx: 4},
		},
		TimeZoneDesignation: []byte(strings.Join([]string{
			"LMT\x00",
			"HST\x00",
			"HDT\x00",
			"HWT\x00",
			"HPT\x00"}, "")),
		UTLocalIndicators: []bool{
			false, false, false, false, true, false,
		},
		StandardWallIndicators: []bool{
			false, false, false, false, true, false,
		},
	}
	v2Footer := Footer{
		TZString: []byte("HST10"),
	}

	var buf bytes.Buffer
	if err := v1Header.Write(&buf); err != nil {
		t.Fatalf("write header: %v", err)
	}
	if err := v1Block.Write(&buf); err != nil {
		t.Fatalf("write block: %v", err)
	}
	if err := v2Header.Write(&buf); err != nil {
		t.Fatalf("write header: %v", err)
	}
	if err := v2Block.Write(&buf); err != nil {
		t.Fatalf("write block: %v", err)
	}
	if err := v2Footer.Write(&buf); err != nil {
		t.Fatalf("write footer: %v", err)
	}
	got := buf.Bytes()

	want := []byte{
		// v1 header
		0x54, 0x5a, 0x69, 0x66, // magic
		0x00, // version // TODO: report bug
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x06, // isutcnt
		0x00, 0x00, 0x00, 0x06, // isstdcnt
		0x00, 0x00, 0x00, 0x00, // leapcnt
		0x00, 0x00, 0x00, 0x07, // timecnt
		0x00, 0x00, 0x00, 0x06, // typecnt
		0x00, 0x00, 0x00, 0x14, // charcnt
		// v1 block
		0x80, 0x00, 0x00, 0x00, // trans time[0]
		0xbb, 0x05, 0x43, 0x48, // trans time[1]
		0xbb, 0x21, 0x71, 0x58, // trans time[2]
		0xcb, 0x89, 0x3d, 0xc8, // trans time[3]
		0xd2, 0x23, 0xf4, 0x70, // trans time[4]
		0xd2, 0x61, 0x49, 0x38, // trans time[5]
		0xd5, 0x8d, 0x73, 0x48, // trans time[6]
		0x01, // trans type[0]
		0x02, // trans type[1]
		0x01, // trans type[2]
		0x03, // trans type[3]
		0x04, // trans type[4]
		0x01, // trans type[5]
		0x05, // trans type[6]
		// localtimetype[0]
		0xff, 0xff, 0x6c, 0x02, // utcoff
		0x00, // isdst
		0x00, // desigidx
		// localtimetype[1]
		0xff, 0xff, 0x6c, 0x58, // utcoff
		0x00, // isdst
		0x04, // desigidx
		// localtimetype[2]
		0xff, 0xff, 0x7a, 0x68, // utcoff
		0x01, // isdst
		0x08, // desigidx
		// localtimetype[3]
		0xff, 0xff, 0x7a, 0x68, // utcoff
		0x01, // isdst
		0x0c, // desigidx
		// localtimetype[4]
		0xff, 0xff, 0x7a, 0x68, // utcoff
		0x01, // isdst
		0x10, // desigidx
		// localtimetype[5]
		0xff, 0xff, 0x73, 0x60, // utcoff
		0x00,                   // isdst
		0x04,                   // desigidx
		0x4c, 0x4d, 0x54, 0x00, // designations[0]
		0x48, 0x53, 0x54, 0x00, // designations[4]
		0x48, 0x44, 0x54, 0x00, // designations[8]
		0x48, 0x57, 0x54, 0x00, // designations[12]
		0x48, 0x50, 0x54, 0x00, // designations[16]
		0x01, // UT/local[0] // TODO: report bug
		0x00, // UT/local[1]
		0x00, // UT/local[2]
		0x00, // UT/local[3]
		0x01, // UT/local[4]
		0x00, // UT/local[5]
		0x01, // standard/wall[0] // TODO: report bug
		0x00, // standard/wall[1]
		0x00, // standard/wall[2]
		0x00, // standard/wall[3]
		0x01, // standard/wall[4]
		0x00, // standard/wall[5]
		// v2 header
		0x54, 0x5a, 0x69, 0x66, // magic
		0x32, // version
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x06, // isutcnt
		0x00, 0x00, 0x00, 0x06, // isstdcnt
		0x00, 0x00, 0x00, 0x00, // leapcnt
		0x00, 0x00, 0x00, 0x07, // timecnt
		0x00, 0x00, 0x00, 0x06, // typecnt
		0x00, 0x00, 0x00, 0x14, // charcnt
		// v2 block
		0xff, 0xff, 0xff, 0xff, // trans time[0]
		0x74, 0xe0, 0x70, 0xbe,
		0xff, 0xff, 0xff, 0xff, // trans time[1]
		0xbb, 0x05, 0x43, 0x48,
		0xff, 0xff, 0xff, 0xff, // trans time[2]
		0xbb, 0x21, 0x71, 0x58,
		0xff, 0xff, 0xff, 0xff, // trans time[3]
		0xcb, 0x89, 0x3d, 0xc8,
		0xff, 0xff, 0xff, 0xff, // trans time[4]
		0xd2, 0x23, 0xf4, 0x70,
		0xff, 0xff, 0xff, 0xff, // trans time[5]
		0xd2, 0x61, 0x49, 0x38,
		0xff, 0xff, 0xff, 0xff, // trans time[6]
		0xd5, 0x8d, 0x73, 0x48,
		0x01, // trans type[0]
		0x02, // trans type[1]
		0x01, // trans type[2]
		0x03, // trans type[3]
		0x04, // trans type[4]
		0x01, // trans type[5]
		0x05, // trans type[6]
		// localtimetype[0]
		0xff, 0xff, 0x6c, 0x02, // utcoff
		0x00, // isdst
		0x00, // desigidx
		// localtimetype[1]
		0xff, 0xff, 0x6c, 0x58, // utcoff
		0x00, // isdst
		0x04, // desigidx
		// localtimetype[2]
		0xff, 0xff, 0x7a, 0x68, // utcoff
		0x01, // isdst
		0x08, // desigidx
		// localtimetype[3]
		0xff, 0xff, 0x7a, 0x68, // utcoff
		0x01, // isdst
		0x0c, // desigidx
		// localtimetype[4]
		0xff, 0xff, 0x7a, 0x68, // utcoff
		0x01, // isdst
		0x10, // desigidx
		// localtimetype[5]
		0xff, 0xff, 0x73, 0x60, // utcoff
		0x00,                   // isdst
		0x04,                   // desigidx
		0x4c, 0x4d, 0x54, 0x00, // designations[0]
		0x48, 0x53, 0x54, 0x00, // designations[4]
		0x48, 0x44, 0x54, 0x00, // designations[8]
		0x48, 0x57, 0x54, 0x00, // designations[12]
		0x48, 0x50, 0x54, 0x00, // designations[16]
		0x00, // UT/local[0]
		0x00, // UT/local[1]
		0x00, // UT/local[2]
		0x00, // UT/local[3]
		0x01, // UT/local[4]
		0x00, // UT/local[5]
		0x00, // standard/wall[0]
		0x00, // standard/wall[1]
		0x00, // standard/wall[2]
		0x00, // standard/wall[3]
		0x01, // standard/wall[4]
		0x00, // standard/wall[5]
		// v2 footer
		0x0a,                   // NL
		0x48, 0x53, 0x54, 0x31, // TZ string
		0x30,
		0x0a, // NL
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("WriteV1() mismatch (-got +want):\n%s", diff)
	}

	// Check that we can decode the file we just encoded.
	f, err := DecodeFile(bytes.NewReader(want))
	if err != nil {
		t.Fatalf("DecodeFile() failed: %v", err)
	}
	if diff := cmp.Diff(f.V1Header, v1Header); diff != "" {
		t.Errorf("DecodeFile() V1Header mismatch (-got +want):\n%s", diff)
	}
	if diff := cmp.Diff(f.V1Data, v1Block); diff != "" {
		t.Errorf("DecodeFile() V1Block mismatch (-got +want):\n%s", diff)
	}
	if diff := cmp.Diff(f.V2Header, v2Header); diff != "" {
		t.Errorf("DecodeFile() V2Header mismatch (-got +want):\n%s", diff)
	}
	if diff := cmp.Diff(f.V2Data, v2Block); diff != "" {
		t.Errorf("DecodeFile() V2Block mismatch (-got +want):\n%s", diff)
	}
	if diff := cmp.Diff(f.V2Footer, v2Footer); diff != "" {
		t.Errorf("DecodeFile() Footer mismatch (-got +want):\n%s", diff)
	}
}

func TestV3FileRepresentingAsiaJerusalem(t *testing.T) {
	// This is the example B.3. from RFC 8536.
	v1Header := Header{
		Version:  V1,
		Isutcnt:  0,
		Isstdcnt: 0,
		Leapcnt:  0,
		Timecnt:  0,
		Typecnt:  0,
		Charcnt:  0,
	}
	v3Header := Header{
		Version:  V3,
		Isutcnt:  1,
		Isstdcnt: 1,
		Leapcnt:  0,
		Timecnt:  1,
		Typecnt:  1,
		Charcnt:  4,
	}
	v3Block := V2DataBlock{
		TransitionTimes: []int64{2145916800},
		TransitionTypes: []uint8{0},
		LocalTimeTypeRecord: []LocalTimeTypeRecord{
			{Utoff: 7200, Dst: false, Idx: 0},
		},
		TimeZoneDesignation:    []byte("IST\x00"),
		UTLocalIndicators:      []bool{true},
		StandardWallIndicators: []bool{true},
	}
	v3Footer := Footer{
		TZString: []byte("IST-2IDT,M3.4.4/26,M10.5.0"),
	}

	var buf bytes.Buffer
	if err := v1Header.Write(&buf); err != nil {
		t.Fatalf("write header: %v", err)
	}
	if err := v3Header.Write(&buf); err != nil {
		t.Fatalf("write header: %v", err)
	}
	if err := v3Block.Write(&buf); err != nil {
		t.Fatalf("write block: %v", err)
	}
	if err := v3Footer.Write(&buf); err != nil {
		t.Fatalf("write footer: %v", err)
	}
	got := buf.Bytes()

	want := []byte{
		// v1 header
		0x54, 0x5a, 0x69, 0x66, // magic
		0x00, // version // TODO: report bug
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, // isutcnt
		0x00, 0x00, 0x00, 0x00, // isstdcnt
		0x00, 0x00, 0x00, 0x00, // leapcnt
		0x00, 0x00, 0x00, 0x00, // timecnt
		0x00, 0x00, 0x00, 0x00, // typecnt
		0x00, 0x00, 0x00, 0x00, // charcnt
		// v3 header
		0x54, 0x5a, 0x69, 0x66, // magic
		0x33, // version
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x01, // isutcnt // TODO: report bug
		0x00, 0x00, 0x00, 0x01, // isstdcnt // TODO: report bug
		0x00, 0x00, 0x00, 0x00, // leapcnt
		0x00, 0x00, 0x00, 0x01, // timecnt // TODO: report bug
		0x00, 0x00, 0x00, 0x01, // typecnt // TODO: report bug
		0x00, 0x00, 0x00, 0x04, // charcnt // TODO: report bug
		// v3 block
		0x00, 0x00, 0x00, 0x00, // trans time[0]
		0x7f, 0xe8, 0x17, 0x80,
		0x00, // trans type [0]
		// localtimetype[0]
		0x00, 0x00, 0x1c, 0x20, // utcoff
		0x00,                   // isdst
		0x00,                   // desigidx
		0x49, 0x53, 0x54, 0x00, // designations[0]
		0x01, // UT/local[0]
		0x01, // standard/wall[0]
		// v3 footer
		0x0a,                   // NL
		0x49, 0x53, 0x54, 0x2d, // TZ string
		0x32, 0x49, 0x44, 0x54,
		0x2c, 0x4d, 0x33, 0x2e,
		0x34, 0x2e, 0x34, 0x2f,
		0x32, 0x36, 0x2c, 0x4d,
		0x31, 0x30, 0x2e, 0x35,
		0x2e, 0x30,
		0x0a, // NL
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("WriteV1() mismatch (-got +want):\n%s", diff)
	}

	// Check that we can decode the file we just encoded.
	f, err := DecodeFile(bytes.NewReader(want))
	if err != nil {
		t.Fatalf("DecodeFile() failed: %v", err)
	}
	if diff := cmp.Diff(f.V1Header, v1Header); diff != "" {
		t.Errorf("DecodeFile() V1Header mismatch (-got +want):\n%s", diff)
	}
	if diff := cmp.Diff(f.V2Header, v3Header); diff != "" {
		t.Errorf("DecodeFile() V2Header mismatch (-got +want):\n%s", diff)
	}
	if diff := cmp.Diff(f.V2Data, v3Block); diff != "" {
		t.Errorf("DecodeFile() V2Block mismatch (-got +want):\n%s", diff)
	}
	if diff := cmp.Diff(f.V2Footer, v3Footer); diff != "" {
		t.Errorf("DecodeFile() Footer mismatch (-got +want):\n%s", diff)
	}
}

func TestReadHeader(t *testing.T) {
	h := Header{
		Version:  V1,
		Isutcnt:  10,
		Isstdcnt: 20,
		Leapcnt:  30,
		Timecnt:  40,
		Typecnt:  50,
		Charcnt:  60,
	}
	var buf bytes.Buffer
	if err := h.Write(&buf); err != nil {
		t.Fatalf("write header: %v", err)
	}
	got, err := ReadHeader(&buf)
	if err != nil {
		t.Fatalf("read header: %v", err)
	}
	if diff := cmp.Diff(got, h); diff != "" {
		t.Errorf("ReadHeader() mismatch (-got +want):\n%s", diff)
	}
}

func TestReadV1DataBlock(t *testing.T) {
	h := Header{
		Version:  V1,
		Isutcnt:  2,
		Isstdcnt: 2,
		Leapcnt:  2,
		Timecnt:  2,
		Typecnt:  2,
		Charcnt:  6,
	}
	b := V1DataBlock{
		TransitionTimes: []int32{1, 2},
		TransitionTypes: []uint8{3, 4},
		LocalTimeTypeRecord: []LocalTimeTypeRecord{
			{Utoff: 5, Dst: true, Idx: 6},
			{Utoff: 7, Dst: false, Idx: 8},
		},
		LeapSecondRecords: []V1LeapSecondRecord{
			{Occur: 9, Corr: 10},
			{Occur: 11, Corr: 12},
		},
		TimeZoneDesignation:    []byte("TZ\x00ZT\x00"),
		UTLocalIndicators:      []bool{true, false},
		StandardWallIndicators: []bool{true, false},
	}
	var buf bytes.Buffer
	if err := b.Write(&buf); err != nil {
		t.Fatalf("write block: %v", err)
	}

	got, err := ReadV1DataBlock(&buf, h)
	if err != nil {
		t.Fatalf("read block: %v", err)
	}

	if diff := cmp.Diff(got, b); diff != "" {
		t.Errorf("ReadV1DataBlock() mismatch (-got +want):\n%s", diff)
	}
}

func TestReadV2DataBlock(t *testing.T) {
	h := Header{
		Version:  V2,
		Isutcnt:  2,
		Isstdcnt: 2,
		Leapcnt:  2,
		Timecnt:  2,
		Typecnt:  2,
		Charcnt:  6,
	}
	b := V2DataBlock{
		TransitionTimes: []int64{1, 2},
		TransitionTypes: []uint8{3, 4},
		LocalTimeTypeRecord: []LocalTimeTypeRecord{
			{Utoff: 5, Dst: true, Idx: 6},
			{Utoff: 7, Dst: false, Idx: 8},
		},
		LeapSecondRecords: []V2LeapSecondRecord{
			{Occur: 9, Corr: 10},
			{Occur: 11, Corr: 12},
		},
		TimeZoneDesignation:    []byte("TZ\x00ZT\x00"),
		UTLocalIndicators:      []bool{true, false},
		StandardWallIndicators: []bool{true, false},
	}
	var buf bytes.Buffer
	if err := b.Write(&buf); err != nil {
		t.Fatalf("write block: %v", err)
	}

	got, err := ReadV2DataBlock(&buf, h)
	if err != nil {
		t.Fatalf("read block: %v", err)
	}

	if diff := cmp.Diff(got, b); diff != "" {
		t.Errorf("ReadV2DataBlock() mismatch (-got +want):\n%s", diff)
	}
}

func TestReadFooter(t *testing.T) {
	f := Footer{
		TZString: []byte("TZ"),
	}
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatalf("write footer: %v", err)
	}
	got, err := ReadFooter(&buf)
	if err != nil {
		t.Fatalf("read footer: %v", err)
	}
	if diff := cmp.Diff(got, f); diff != "" {
		t.Errorf("ReadFooter() mismatch (-got +want):\n%s", diff)
	}
}

func TestFile_Encode_V1(t *testing.T) {
	v1h := Header{
		Version:  V1,
		Isutcnt:  2,
		Isstdcnt: 2,
		Leapcnt:  2,
		Timecnt:  2,
		Typecnt:  2,
		Charcnt:  6,
	}
	v1b := V1DataBlock{
		TransitionTimes: []int32{1, 2},
		TransitionTypes: []uint8{3, 4},
		LocalTimeTypeRecord: []LocalTimeTypeRecord{
			{Utoff: 5, Dst: true, Idx: 6},
			{Utoff: 7, Dst: false, Idx: 8},
		},
		LeapSecondRecords: []V1LeapSecondRecord{
			{Occur: 9, Corr: 10},
			{Occur: 11, Corr: 12},
		},
		TimeZoneDesignation:    []byte("TZ\x00ZT\x00"),
		UTLocalIndicators:      []bool{true, false},
		StandardWallIndicators: []bool{true, false},
	}

	f := File{
		V1Header: v1h,
		V1Data:   v1b,
	}
	var buf bytes.Buffer
	if err := f.Encode(&buf); err != nil {
		t.Fatalf("encode: %v", err)
	}
	decodeBuf := bytes.NewBuffer(buf.Bytes()) // copy for decode test

	gotH, err := ReadHeader(&buf)
	if err != nil {
		t.Fatalf("read header: %v", err)
	}
	if diff := cmp.Diff(gotH, v1h); diff != "" {
		t.Errorf("header mismatch (-got +want):\n%s", diff)
	}

	gotD, err := ReadV1DataBlock(&buf, gotH)
	if err != nil {
		t.Fatalf("read block: %v", err)
	}
	if diff := cmp.Diff(gotD, v1b); diff != "" {
		t.Errorf("block mismatch (-got +want):\n%s", diff)
	}

	if buf.Len() != 0 {
		t.Errorf("buffer not empty: %d", buf.Len())
	}

	gotF, err := DecodeFile(decodeBuf)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if diff := cmp.Diff(gotF, f); diff != "" {
		t.Errorf("decode mismatch (-got +want):\n%s", diff)
	}
}

func TestFile_Encode_V2(t *testing.T) {
	v1h := Header{
		Version:  V1,
		Isutcnt:  2,
		Isstdcnt: 2,
		Leapcnt:  2,
		Timecnt:  2,
		Typecnt:  2,
		Charcnt:  6,
	}
	v1b := V1DataBlock{
		TransitionTimes: []int32{1, 2},
		TransitionTypes: []uint8{3, 4},
		LocalTimeTypeRecord: []LocalTimeTypeRecord{
			{Utoff: 5, Dst: true, Idx: 6},
			{Utoff: 7, Dst: false, Idx: 8},
		},
		LeapSecondRecords: []V1LeapSecondRecord{
			{Occur: 9, Corr: 10},
			{Occur: 11, Corr: 12},
		},
		TimeZoneDesignation:    []byte("TZ\x00ZT\x00"),
		UTLocalIndicators:      []bool{true, false},
		StandardWallIndicators: []bool{true, false},
	}
	v2h := Header{
		Version:  V2,
		Isutcnt:  2,
		Isstdcnt: 2,
		Leapcnt:  2,
		Timecnt:  2,
		Typecnt:  2,
		Charcnt:  6,
	}
	v2b := V2DataBlock{
		TransitionTimes: []int64{1, 2},
		TransitionTypes: []uint8{3, 4},
		LocalTimeTypeRecord: []LocalTimeTypeRecord{
			{Utoff: 5, Dst: true, Idx: 6},
			{Utoff: 7, Dst: false, Idx: 8},
		},
		LeapSecondRecords: []V2LeapSecondRecord{
			{Occur: 9, Corr: 10},
			{Occur: 11, Corr: 12},
		},
		TimeZoneDesignation:    []byte("TZ\x00ZT\x00"),
		UTLocalIndicators:      []bool{true, false},
		StandardWallIndicators: []bool{true, false},
	}
	v2f := Footer{
		TZString: []byte("TZ"),
	}

	f := File{
		Version:  V2,
		V1Header: v1h,
		V1Data:   v1b,
		V2Header: v2h,
		V2Data:   v2b,
		V2Footer: v2f,
	}
	var buf bytes.Buffer
	if err := f.Encode(&buf); err != nil {
		t.Fatalf("encode: %v", err)
	}
	decodeBuf := bytes.NewBuffer(buf.Bytes()) // copy for decode test

	gotH1, err := ReadHeader(&buf)
	if err != nil {
		t.Fatalf("read v1 header: %v", err)
	}
	if diff := cmp.Diff(gotH1, v1h); diff != "" {
		t.Errorf("v1 header mismatch (-got +want):\n%s", diff)
	}

	gotD1, err := ReadV1DataBlock(&buf, gotH1)
	if err != nil {
		t.Fatalf("read v1 block: %v", err)
	}
	if diff := cmp.Diff(gotD1, v1b); diff != "" {
		t.Errorf("v1 block mismatch (-got +want):\n%s", diff)
	}

	gotH2, err := ReadHeader(&buf)
	if err != nil {
		t.Fatalf("read v2 header: %v", err)
	}
	if diff := cmp.Diff(gotH2, v2h); diff != "" {
		t.Errorf("v2 header mismatch (-got +want):\n%s", diff)
	}

	gotD2, err := ReadV2DataBlock(&buf, gotH2)
	if err != nil {
		t.Fatalf("read v2 block: %v", err)
	}
	if diff := cmp.Diff(gotD2, v2b); diff != "" {
		t.Errorf("v2 block mismatch (-got +want):\n%s", diff)
	}

	gotF2, err := ReadFooter(&buf)
	if err != nil {
		t.Fatalf("read footer: %v", err)
	}
	if diff := cmp.Diff(gotF2, v2f); diff != "" {
		t.Errorf("footer mismatch (-got +want):\n%s", diff)
	}

	if buf.Len() != 0 {
		t.Errorf("buffer not empty: %d", buf.Len())
	}

	gotF, err := DecodeFile(decodeBuf)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if diff := cmp.Diff(gotF, f); diff != "" {
		t.Errorf("decode mismatch (-got +want):\n%s", diff)
	}
}

func TestFile_V2WithV1Missing(t *testing.T) {
	v2h := Header{
		Version:  V2,
		Isutcnt:  2,
		Isstdcnt: 2,
		Leapcnt:  2,
		Timecnt:  2,
		Typecnt:  2,
		Charcnt:  6,
	}
	v2b := V2DataBlock{
		TransitionTimes: []int64{1, 2},
		TransitionTypes: []uint8{3, 4},
		LocalTimeTypeRecord: []LocalTimeTypeRecord{
			{Utoff: 5, Dst: true, Idx: 6},
			{Utoff: 7, Dst: false, Idx: 8},
		},
		LeapSecondRecords: []V2LeapSecondRecord{
			{Occur: 9, Corr: 10},
			{Occur: 11, Corr: 12},
		},
		TimeZoneDesignation:    []byte("TZ\x00ZT\x00"),
		UTLocalIndicators:      []bool{true, false},
		StandardWallIndicators: []bool{true, false},
	}
	v2f := Footer{
		TZString: []byte("TZ"),
	}

	f := File{
		Version:   V2,
		V1Missing: true,
		V2Header:  v2h,
		V2Data:    v2b,
		V2Footer:  v2f,
	}
	var buf bytes.Buffer
	if err := f.Encode(&buf); err != nil {
		t.Fatalf("encode: %v", err)
	}
	decodeBuf := bytes.NewBuffer(buf.Bytes()) // copy for decode test

	gotH2, err := ReadHeader(&buf)
	if err != nil {
		t.Errorf("read v2 header: %v", err)
	}
	if diff := cmp.Diff(gotH2, v2h); diff != "" {
		t.Errorf("v2 header mismatch (-got +want):\n%s", diff)
	}

	gotD2, err := ReadV2DataBlock(&buf, gotH2)
	if err != nil {
		t.Errorf("read v2 block: %v", err)
	}
	if diff := cmp.Diff(gotD2, v2b); diff != "" {
		t.Errorf("v2 block mismatch (-got +want):\n%s", diff)
	}

	gotF2, err := ReadFooter(&buf)
	if err != nil {
		t.Errorf("read footer: %v", err)
	}
	if diff := cmp.Diff(gotF2, v2f); diff != "" {
		t.Errorf("footer mismatch (-got +want):\n%s", diff)
	}

	if buf.Len() != 0 {
		t.Errorf("buffer not empty: %d", buf.Len())
	}

	gotF, err := DecodeFile(decodeBuf)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if diff := cmp.Diff(gotF, f); diff != "" {
		t.Errorf("decode mismatch (-got +want):\n%s", diff)
	}
}
