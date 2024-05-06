package tzc

import (
	"bytes"
	"fmt"
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
	var zones = make(map[string]tzif.Data)
	for _, zl := range f.ZoneLines {
		z, err := compileZone(f, zl)
		if err != nil {
			return nil, fmt.Errorf("compiling zone %s: %v", zl.Name, err)
		}
		zones[zl.Name] = z
	}
	return zones, nil
}

func compileZone(f tzdata.File, zone tzdata.ZoneLine) (tzif.Data, error) {
	// TODO
	return tzif.Data{}, nil
}
