// Package ianadist downloads and extracts tzdb files distributed by IANA.
//
// Releases are downloaded from the [IANA data server]. Clients are advised
// to store the [ETags] returned in this package and pass them to subsequent
// calls to avoid downloading the same data multiple times.
//
// [ETags]: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/ETag
// [IANA data server]: https://www.iana.org/time-zones
package ianadist

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// TZDataFiles is a map of tzdb data file names to file contents.
// Filenames are never empty and file contents always start with
// the magic header that indicates the start of a data file:
//
//	# tzdb data for
//
// Example:
//
//	 TZDataFiles{
//		"africa", []byte("# tzdb data for Africa and environs\n..."),
//		"europe", []byte("# tzdb data for Europe and environs\n..."),
//	 	"etcetera", []byte("# tzdb data for shipts at sea and other miscellany\n..."),
//	 }
type TZDataFiles map[string][]byte

// Release is a parsed IANA time zone database release.
type Release struct {
	// Version is the version of the IANA time zone database.
	// For example, "2021a".
	Version string
	// DataFiles is a map of tzdb data file names to file contents.
	DataFiles TZDataFiles
	// LeapSecondsFile is the content of the leap seconds file.
	LeapSecondsFile []byte
}

// DefaultClient is the default client to download the IANA time zone database.
// It is ready to use and is used by the top-level functions [Latest] and [Download]
// in this package.
var DefaultClient = &Client{}

// Client is a client to download the IANA time zone database.
// The zero value is ready to use.
type Client struct {
	// HTTPClient is the http.Client used to download the IANA time zone database.
	// If HTTPClient is nil, http.DefaultClient is used.
	//
	// This variable is useful to prevent network calls during tests by using a
	// http.Client with a fake http.RoundTripper that returns canned responses.
	// You can also use it to set timeouts, control redirects, etc.
	// However, timeouts are also controlled by the context passed to the
	// Download and Latest methods.
	HTTPClient *http.Client
}

// httpClient returns the http.Client used by the client.
// If HTTPClient is nil, http.DefaultClient is returned.
func (c *Client) httpClient() *http.Client {
	if c.HTTPClient == nil {
		return http.DefaultClient
	}
	return c.HTTPClient
}

const (
	// baseURL is the base URL for time zones on the IANA data server.
	baseURL = "https://data.iana.org/time-zones/"
	// latestDataPath is the path to the latest IANA time zone database
	// relative to the baseURL.
	latestDataPath = "tzdata-latest.tar.gz"
	// dataFileMagicHeader is used to identify data files in the archive.
	dataFileMagicHeader = "# tzdb data for"
	// leapSecondsFilename is the name of the leap seconds file in the archive.
	leapSecondsFilename = "leapseconds"
	// versionFilename is the name of the version file in the archive.
	versionFilename = "version"
	// emptyEtag is the empty etag value.
	emptyEtag = ""
)

// ReadArchive unpacks the IANA time zone database from an archive.
//
// The io.Reader must contain a gzip-compressed tar archive as found at
// https://data.iana.org/time-zones/releases/.
func ReadArchive(r io.Reader) (*Release, error) {
	gunzip, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("read gzip: %w", err)
	}
	tr := tar.NewReader(gunzip)

	var (
		result   = Release{DataFiles: make(map[string][]byte)}
		magicBuf = make([]byte, len(dataFileMagicHeader))
	)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		switch header.Name {
		case leapSecondsFilename:
			result.LeapSecondsFile, err = io.ReadAll(tr)
			if err != nil {
				return nil, fmt.Errorf("read leap seconds file: %w", err)
			}
			continue
		case versionFilename:
			versionBytes, err := io.ReadAll(tr)
			if err != nil {
				return nil, fmt.Errorf("read version file: %w", err)
			}
			if len(versionBytes) == 0 {
				return nil, fmt.Errorf("empty version file")
			}
			result.Version = string(versionBytes)
			continue
		}

		if header.Size < int64(len(dataFileMagicHeader)) {
			// Too small to contain the magic string.
			continue
		}

		// Read only the magic string to check if it's a data file.
		_, err = io.ReadFull(tr, magicBuf)
		if err != nil {
			return nil, fmt.Errorf("read magic string %q: %w", header.Name, err)
		}
		if string(magicBuf) != dataFileMagicHeader {
			continue // Not a data file.
		}

		// Is data file. Prepare to read the rest of the file.
		data := make([]byte, header.Size)
		copy(data[:len(dataFileMagicHeader)], magicBuf)

		// Read the rest of the file.
		_, err = io.ReadFull(tr, data[len(dataFileMagicHeader):])
		if err != nil {
			return nil, fmt.Errorf("read rest of file %q: %w", header.Name, err)
		}

		result.DataFiles[header.Name] = data
	}

	if len(result.DataFiles) == 0 {
		return nil, fmt.Errorf("no data files found")
	}
	if result.Version == "" {
		return nil, fmt.Errorf("no version found")
	}

	return &result, nil
}

// Latest downloads and unpacks the latest IANA time zone database.
//
// If the server responds with a 304 Not Modified status code, the returned
// ETag is the same as the input and the returned Release and error are
// both nil.
//
// If an error is returned, the returned ETag is empty and the returned
// Release is nil.
//
// Latest is a wrapper around DefaultClient.Latest.
func Latest(ctx context.Context, etag string) (*Release, string, error) {
	return DefaultClient.Latest(ctx, etag)
}

// Latest downloads and unpacks the latest IANA time zone database.
//
// If the server responds with a 304 Not Modified status code, the returned
// ETag is the same as the input and the returned Release and error are
// both nil.
//
// If an error is returned, the returned ETag is empty and the returned
// Release is nil.
func (c *Client) Latest(ctx context.Context, etag string) (*Release, string, error) {
	r, newEtag, err := c.Download(ctx, latestDataPath, etag)
	if err != nil {
		return nil, emptyEtag, err
	}
	if r == nil {
		return nil, etag, nil // Not modified.
	}
	defer func() {
		// Drain and close the response body to ensure the
		// connection can be reused.
		_, _ = io.ReadAll(r)
		_ = r.Close()
	}()

	release, err := ReadArchive(r)
	if err != nil {
		return nil, emptyEtag, err
	}

	return release, newEtag, nil
}

// Download downloads the resource at the given path from the IANA time zone
// data server.
//
// The returned ETag is the ETag of the downloaded resource. If the server
// responds with a 304 Not Modified status code, the returned ETag is the same
// as the input and the returned io.ReadCloser and error are both nil.
//
// If no error is returned, the returned io.ReadCloser is a [http.Response.Body]
// and needs to be read fully and closed by the caller to prevent resource leaks.
// Read more about closing the response body at
// https://pkg.go.dev/net/http#Response.
//
// If a non-nil error is returned, the returned ETag is empty and the returned
// io.ReadCloser is nil.
//
// An error is returned for HTTP status codes other than 200 OK and 304 Not Modified.
//
// The given context.Context is passed to the http.Request and can be used to
// control cancellation and timeouts.
//
// Download is a wrapper around DefaultClient.Download.
func Download(ctx context.Context, path, etag string) (io.ReadCloser, string, error) {
	return DefaultClient.Download(ctx, path, etag)
}

// Download downloads the resource at the given path from the IANA time zone
// data server.
//
// The returned ETag is the ETag of the downloaded resource. If the server
// responds with a 304 Not Modified status code, the returned ETag is the same
// as the input and the returned io.ReadCloser and error are both nil.
//
// If no error is returned, the returned io.ReadCloser is a [http.Response.Body]
// and needs to be read fully and closed by the caller to prevent resource leaks.
// Read more about closing the response body at
// https://pkg.go.dev/net/http#Response.
//
// If a non-nil error is returned, the returned ETag is empty and the returned
// io.ReadCloser is nil.
//
// An error is returned for HTTP status codes other than 200 OK and 304 Not Modified.
//
// The given context.Context is passed to the http.Request and can be used to
// control cancellation and timeouts.
func (c *Client) Download(ctx context.Context, path, etag string) (io.ReadCloser, string, error) {
	u, err := url.JoinPath(baseURL, path)
	if err != nil {
		return nil, emptyEtag, fmt.Errorf("join URL: %w", err)
	}
	r, etag, err := c.downloadIfNoneMatch(ctx, u, etag)
	if err != nil {
		return nil, etag, err
	}
	return r, etag, nil
}

// downloadIfNoneMatch downloads the resource at the given URL with caching using the given ETag.
//
// If a non-nil error is returned, the returned io.ReadCloser is a [http.Response.Body]
// and needs to be read fully and closed by the caller to prevent resource leaks.
// Read more about closing the response body at https://pkg.go.dev/net/http#Response.
//
// If the etag is not empty and the server responds with a 304 Not Modified status code,
// the returned io.ReadCloser and error are both nil, and the etag is the same as the input.
func (c *Client) downloadIfNoneMatch(ctx context.Context, url, etag string) (io.ReadCloser, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, emptyEtag, fmt.Errorf("create request for %q: %w", url, err)
	}

	if etag != emptyEtag {
		req.Header.Set("If-None-Match", etag)
	}

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, emptyEtag, fmt.Errorf("GET %q: %w", url, err)
	}

	if resp.StatusCode != http.StatusOK {
		// Drain and close the response body to reuse the connection.
		// In theory, the server will not send a body with all status codes,
		// but draining before closing the body is the safe thing to do.
		_, _ = io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		// Not modified response means the resource has not changed
		// based on the ETag we sent. This is fine.
		if resp.StatusCode == http.StatusNotModified {
			return nil, etag, nil
		}

		return nil, emptyEtag, fmt.Errorf("response for %q: unexpected status: %s", url, resp.Status)
	}

	// Caller must take care of closing the response body.
	return resp.Body, resp.Header.Get("etag"), nil
}
