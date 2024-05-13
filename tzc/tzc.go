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
	var b builder
	b.minimalV1Compliance()

	irzs, err := tzir.Process(f, lines)
	if err != nil {
		fmt.Println(err)
	}

	for _, z := range irzs {
		if z.Expires {
			for _, t := range z.Transitions {
				b.addTransition(t.Occ)
			}
		}
	}

	b.deriveV2HeaderFromData()
	b.setFooter("TODO")

	return b.Data(), nil
}
