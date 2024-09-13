package ianadist

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

// roundTripperFunc is a function that implements the http.RoundTripper interface.
// Useful to fake a http.Client with fakeClient.
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func fakeClient(fn roundTripperFunc) *http.Client {
	return &http.Client{Transport: fn}
}

// testTZDataFiles checks that the TZDataFiles map adheres to the expected format.
func testTZDataFiles(t *testing.T, files TZDataFiles) {
	t.Helper()
	for file, data := range files {
		if len(file) == 0 {
			t.Errorf("TZDataFiles: empty file name.")
		}
		if !strings.HasPrefix(string(data), "# tzdb data for") {
			t.Errorf("TZDataFiles: data missing magic string in %q", file)
		}
	}
}

// mustReadTestData reads the testdata file and returns its contents.
func mustReadTestData(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile("../testdata/tzdata-2024b.tar.gz")
	if err != nil {
		t.Fatalf("failed to read testdata: %v", err)
	}
	return data
}

func TestLatest(t *testing.T) {
	const (
		testEtag  = "test-etag"
		emptyEtag = ""
	)
	httpClient := fakeClient(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("unexpected method %q", req.Method)
		}
		if req.URL.String() != "https://data.iana.org/time-zones/tzdata-latest.tar.gz" {
			t.Errorf("unexpected URL %q", req.URL)
		}

		if req.Header.Get("If-None-Match") == testEtag {
			return &http.Response{
				StatusCode: http.StatusNotModified,
			}, nil
		}

		data := mustReadTestData(t)
		resp := &http.Response{
			Body:       io.NopCloser(bytes.NewReader(data)),
			StatusCode: http.StatusOK,
		}
		resp.Header = make(http.Header)
		resp.Header.Set("etag", testEtag)
		return resp, nil
	})

	DefaultClient = &Client{HTTPClient: httpClient}

	ctx := context.Background()

	// Test that Latest returns the latest data files.
	release, gotEtag, err := Latest(ctx, emptyEtag)
	if err != nil {
		t.Errorf("Latest(%v) returned unexpected error: %v", emptyEtag, err)
	}
	if gotEtag != testEtag {
		t.Errorf("Latest(%v) returned ETag %q, want %q", emptyEtag, gotEtag, testEtag)
	}
	testTZDataFiles(t, release.DataFiles)

	// Test that Latest returns no files when the ETag is up-to-date.
	release, newEtag, err := Latest(ctx, gotEtag)
	if err != nil {
		t.Errorf("Latest(%q) returned unexpected error: %v", gotEtag, err)
	}
	if newEtag != testEtag {
		t.Errorf("Latest(%q) returned ETag %q, want %q", gotEtag, newEtag, testEtag)
	}
	if release != nil {
		t.Errorf("Latest(%q) returned non-nil files", gotEtag)
	}
}

func TestReadArchive(t *testing.T) {
	data := mustReadTestData(t)
	release, err := ReadArchive(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("ReadArchive(...): unexpected non-nil error: %v", err)
	}
	testTZDataFiles(t, release.DataFiles)
}
