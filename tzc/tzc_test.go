package tzc

import (
	"bytes"
	"fmt"
	"github.com/google/go-cmp/cmp"
	"github.com/ngrash/go-tz/tzif"
	"os"
	"path/filepath"
	"testing"
)

type testCase struct {
	Name  string
	Input []byte
	Want  map[string][]byte
}

func loadTestCases(t *testing.T) []testCase {
	t.Helper()

	var cases []testCase

	dataFiles, err := filepath.Glob("testdata/*.tzdata")
	if err != nil {
		t.Fatal(err)
	}

	for _, dataFile := range dataFiles {
		content, err := os.ReadFile(dataFile)
		if err != nil {
			t.Fatal(err)
		}

		name := dataFile[:len(dataFile)-len(".tzdata")]
		cases = append(cases, testCase{Name: name, Input: content, Want: map[string][]byte{}})

		ifFiles, err := filepath.Glob(fmt.Sprintf("%s_tzif/*", name))
		if err != nil {
			t.Fatal(err)
		}

		for _, ifFile := range ifFiles {
			content, err := os.ReadFile(ifFile)
			if err != nil {
				t.Fatal(err)
			}
			cases[len(cases)-1].Want[ifFile[len(name)+1+len("_tzif"):]] = content
		}
	}

	return cases
}

func TestCompile(t *testing.T) {
	data := loadTestCases(t)
	for _, d := range data {
		t.Run(d.Name, func(t *testing.T) {
			compiled, err := CompileBytes(d.Input)
			if err != nil {
				t.Fatalf("CompileBytes() error: %v", err)
			}
			for zone, want := range d.Want {
				got, ok := compiled[zone]

				t.Run(zone, func(t *testing.T) {
					var gotData tzif.Data
					if ok {
						if string(got) == string(want) {
							return // OK
						}
						// Decode the data to compare the contents.
						gotData, err = tzif.DecodeData(bytes.NewReader(got))
						if err != nil {
							t.Fatalf("decode got data: %v", err)
						}
					} else {
						// Zone is missing. Keep going and print the diff.
						t.Errorf("missing zone %s", zone)
					}

					wantData, err := tzif.DecodeData(bytes.NewReader(want))
					if err != nil {
						t.Fatalf("decode want data: %v", err)
					}
					if diff := cmp.Diff(gotData, wantData); diff != "" {
						t.Errorf("tzif mismatch (-got +want):\n%s", diff)
					}
				})
			}
		})
	}
}
