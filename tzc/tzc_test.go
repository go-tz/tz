package tzc

import (
	"bytes"
	"fmt"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/ngrash/go-tz/tzif"
	"os"
	"path/filepath"
	"strings"
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

	inputFiles, err := filepath.Glob("testdata/*.tzdata")
	if err != nil {
		t.Fatal(err)
	}

	for _, in := range inputFiles {
		content, err := os.ReadFile(in)
		if err != nil {
			t.Fatal(err)
		}

		// Extract the name of the zone from the file name; testdata/my_example.tzdata -> my_example
		name := strings.TrimSuffix(filepath.Base(in), ".tzdata")
		tc := testCase{Name: name, Input: content, Want: map[string][]byte{}}

		ifFiles, err := filepath.Glob(fmt.Sprintf("testdata/generated_tzif/%s/*", name))
		if err != nil {
			t.Fatal(err)
		}
		if len(ifFiles) == 0 {
			t.Fatalf("No tzif files found for %s. Did `make` fail?", name)
		}

		for _, ifFile := range ifFiles {
			c, err := os.ReadFile(ifFile)
			if err != nil {
				t.Fatal(err)
			}
			s := filepath.Base(ifFile)
			tc.Want[s] = c
		}
		cases = append(cases, tc)
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
				t.Run(zone, func(t *testing.T) {
					got, ok := compiled[zone]
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

					// TODO: Implement TZString footer generation. For now, ignore it.
					opts := cmpopts.IgnoreFields(tzif.Footer{}, "TZString")

					if diff := cmp.Diff(gotData, wantData, opts); diff != "" {
						t.Errorf("tzif mismatch (-got +want):\n%s", diff)
					}
				})
			}
		})
	}
}
