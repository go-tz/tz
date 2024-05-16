package tzc

import (
	"bytes"
	"fmt"
	"github.com/ngrash/go-tz/internal/tzir"
	"github.com/ngrash/go-tz/tzdata"
	"github.com/ngrash/go-tz/tzif"
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
		if err := tzif.Validate(z); err != nil {
			return nil, fmt.Errorf("compiling zone %s: invalid tzif: %v", name, err)
		}
		result[name] = z
	}
	return result, nil
}

func compileZone(f tzdata.File, lines []tzdata.ZoneLine) (tzif.Data, error) {
	irzs, err := tzir.Process(f, lines)
	if err != nil {
		fmt.Println(err)
	}

	if len(irzs) == 0 {
		return tzif.Data{}, fmt.Errorf("no zones found")
	}

	var b builder
	b.minimalV1Compliance()

	// Make sure we add designations in order of appearance and not in order of occurrence.
	// This is important because we want to be binary equivalent to the reference implementation zic.
	for _, z := range irzs {
		if z.Expires {
			for _, t := range z.Transitions {
				b.addDesignation(t.Desig)
			}
		} else {
			// Of the final zone, we only transition to the first rule as the rest is derived from TZ string in the footer.
			if len(z.Transitions) > 0 {
				b.addDesignation(z.Transitions[0].Desig)
			}
		}
	}

	// Add initial record and adjust first transition.
	if len(irzs) > 0 && irzs[0].HasInitialTransition {
		b.addLocalTimeTypeRecord(irzs[0].InitialTransition)

		// Apply offset from initial record to first transition occurrence timestamp.
		if len(irzs) > 0 && len(irzs[0].Transitions) > 0 {
			irzs[0].Transitions[0].Occ -= irzs[0].InitialTransition.Off
		}
	} else {
		return tzif.Data{}, fmt.Errorf("unable to determine initial record")
	}

	// Add transitions.
	for i, z := range irzs {
		hasContinuation := i != len(irzs)-1

		if z.Expires {
			for _, t := range z.Transitions {
				b.addTransition(t)
			}

			// Add transition to the first standard time rule of the next zone.
			if !hasContinuation {
				return tzif.Data{}, fmt.Errorf("final zone must not expire")
			}

			next := irzs[i+1]
			if !next.HasInitialTransition {
				return tzif.Data{}, fmt.Errorf("zone without standard time transition: %v", next.Line)
			}

			f := next.InitialTransition // copy
			f.Occ = z.ExpiresAt
			b.addTransition(f)
		}
	}

	// If we only have an initial record but no transitions, we need to add a dummy transition.
	if len(b.d.V2Data.TransitionTimes) == 0 && len(b.d.V2Data.LocalTimeTypeRecord) == 1 {
		if len(irzs) > 0 && len(irzs[0].Transitions) > 0 {
			b.addTransition(irzs[0].Transitions[0])
		}
	}

	b.setFooter("TODO")
	b.deriveV2HeaderFromData()

	return b.Data(), nil
}
