package tzc

import (
	"bytes"
	"fmt"
	"github.com/ngrash/go-tz/internal/tzexpand"
	"github.com/ngrash/go-tz/internal/unixtime"
	"github.com/ngrash/go-tz/tzdata"
	"github.com/ngrash/go-tz/tzif"
	"time"
)

func CompileBytes(dataBuf []byte) (map[string][]byte, error) {
	f, err := tzdata.Parse(bytes.NewReader(dataBuf))
	if err != nil {
		return nil, err
	}
	compiled, err := Compile(f)
	if err != nil {
		return nil, err
	}
	result := make(map[string][]byte)
	for zone, data := range compiled {
		buf := new(bytes.Buffer)
		if err := data.Encode(buf); err != nil {
			return nil, err
		}
		result[zone] = buf.Bytes()
	}
	return result, nil
}

func Compile(f tzdata.File) (map[string]tzif.Data, error) {
	// Group zone lines by zone name.
	var (
		zones    = make(map[string][]tzdata.ZoneLine)
		lastName string
	)
	for _, l := range f.ZoneLines {
		if !l.Continuation {
			lastName = l.Name
		}
		zones[lastName] = append(zones[lastName], l)
	}

	var result = make(map[string]tzif.Data)
	for name, zoneLines := range zones {
		z, err := compileZone(f, zoneLines)
		if err != nil {
			return nil, fmt.Errorf("compiling zone %s: %v", name, err)
		}
		result[name] = z
	}
	return result, nil
}

func appendDesignation(designations []byte, desig string) ([]byte, uint8) {
	if idx := bytes.Index(designations, append([]byte(desig), 0x00)); idx != -1 {
		return designations, uint8(idx)
	}
	return append(designations, append([]byte(desig), 0x00)...), uint8(len(designations))
}

func compileZone(f tzdata.File, lines []tzdata.ZoneLine) (tzif.Data, error) {
	var data tzif.Data
	data.Version = tzif.V2

	ir, ird, err := initialLTTR(f, lines)
	if err != nil {
		return data, fmt.Errorf("could not identify initial local time type record: %v", err)
	}
	data.V2Data.TimeZoneDesignation, ir.Idx = appendDesignation(data.V2Data.TimeZoneDesignation, ird)
	data.V2Data.LocalTimeTypeRecord = []tzif.LocalTimeTypeRecord{ir}
	data.V2Data.TransitionTimes, err = transitions(f, lines)
	if err != nil {
		return data, fmt.Errorf("could not determine transition times: %v", err)
	}
	// TODO: Implement transition types. For now, this is only a placeholder to allocate the correct number of bytes in the binary tzif format.
	data.V2Data.TransitionTypes = make([]uint8, len(data.V2Data.TransitionTimes))

	// Update header.
	data.V2Header.Version = tzif.V2
	data.V2Header.Timecnt = uint32(len(data.V2Data.TransitionTimes))
	data.V2Header.Typecnt = uint32(len(data.V2Data.LocalTimeTypeRecord))
	data.V2Header.Charcnt = uint32(len(data.V2Data.TimeZoneDesignation))

	// Update footer.
	data.V2Footer.TZString = []byte(tzString(data.V2Data))

	// Derive V1 data block.
	copyV1(&data)

	return data, nil
}

func tzString(_ tzif.V2DataBlock) string {
	return "TZA-1"
}

func copyV1(data *tzif.Data) {
	data.V1Data.LocalTimeTypeRecord = data.V2Data.LocalTimeTypeRecord
	data.V1Data.TimeZoneDesignation = data.V2Data.TimeZoneDesignation
	data.V1Data.TransitionTypes = data.V2Data.TransitionTypes

	for _, t := range data.V2Data.TransitionTimes {
		// TODO: This naive implementation will not work for timestamps greater than int32 max value.
		data.V1Data.TransitionTimes = append(data.V1Data.TransitionTimes, int32(t))
	}

	data.V1Header.Version = data.Version
	data.V1Header.Typecnt = uint32(len(data.V1Data.LocalTimeTypeRecord))
	data.V1Header.Charcnt = uint32(len(data.V1Data.TimeZoneDesignation))
	data.V1Header.Timecnt = uint32(len(data.V1Data.TransitionTimes))
}

func transitions(f tzdata.File, lines []tzdata.ZoneLine) ([]int64, error) {
	var times []int64
	var utcOff int64
	for _, l := range lines {
		utcOff = int64(time.Duration(l.Offset) / time.Second)

		if l.Rules.Form == tzdata.ZoneRulesName {
			rules, err := findRules(f.RuleLines, l.Rules.Name)
			if err != nil {
				return nil, err
			}
			for _, r := range rules {
				if !(r.From != tzdata.MinYear && r.From != tzdata.MaxYear && r.To == tzdata.MaxYear) {
					// TODO: This constraint limits us to the most basic rules.
					return nil, fmt.Errorf("unsupported rule range %d-%d", r.From, r.To)
				}

				// TODO: Ignore rules before the previous Zone lines UNTIL date and after this Zone lines UNTIL date.
				y, m, d := tzexpand.DayOfMonth(int(r.From), r.In, r.On)

				hours := int(time.Duration(r.At.TimeOfDay) / time.Hour)
				minutes := int(time.Duration(r.At.TimeOfDay) / time.Minute)
				seconds := int(time.Duration(r.At.TimeOfDay) / time.Second)

				local := unixtime.FromDateTime(y, int(m), d, hours, minutes, seconds)
				ut := local - utcOff
				times = append(times, ut)
			}
		}
	}
	return times, nil
}

func initialLTTR(f tzdata.File, lines []tzdata.ZoneLine) (tzif.LocalTimeTypeRecord, string, error) {
	l := lines[0]

	if l.Rules.Form == tzdata.ZoneRulesStandard {
		r := tzif.LocalTimeTypeRecord{
			Utoff: int32(time.Duration(l.Offset) / time.Second),
			Dst:   false,
			Idx:   0,
		}
		return r, l.Format, nil
	}

	if l.Rules.Form == tzdata.ZoneRulesTime {
		r := tzif.LocalTimeTypeRecord{
			Utoff: int32(time.Duration(l.Offset)/time.Second) + int32(time.Duration(l.Rules.Time.TimeOfDay)/time.Second),
			Dst:   true,
			Idx:   0,
		}
		return r, l.Format, nil
	}

	if l.Rules.Form == tzdata.ZoneRulesName {
		rules, err := findRules(f.RuleLines, l.Rules.Name)
		if err != nil {
			return tzif.LocalTimeTypeRecord{}, "", err
		}

		if len(rules) != 1 {
			return tzif.LocalTimeTypeRecord{}, "", fmt.Errorf("multiple rules found for name %s", l.Rules.Name)
		}
		offset := int32(time.Duration(rules[0].Save.TimeOfDay) / time.Second)
		var dst bool
		switch rules[0].Save.Form {
		//case tzdata.UniversalTime:
		//case tzdata.WallClock:
		case tzdata.StandardTime:
			dst = false
		case tzdata.DaylightSavingTime:
			dst = true
		default:
			return tzif.LocalTimeTypeRecord{}, "", fmt.Errorf("unsupported save form %s", rules[0].Save.Form)
		}

		r := tzif.LocalTimeTypeRecord{
			Utoff: int32(time.Duration(l.Offset)/time.Second) + offset,
			Dst:   dst,
			Idx:   0,
		}
		return r, l.Format, nil
	}

	return tzif.LocalTimeTypeRecord{}, "", fmt.Errorf("unsupported rule form %s", lines[0].Rules.Form)
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
