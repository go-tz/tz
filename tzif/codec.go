package tzif

import (
	"errors"
	"fmt"
	"io"
)

type File struct {
	Version Version

	V1Missing bool
	V1Header  Header
	V1Data    V1DataBlock

	V2Header Header
	V2Data   V2DataBlock
	V2Footer Footer
}

// Encode writes the given TZif data to the given writer.
// If the version v is V1, v2h, v2b, and v2f are ignored.
func (f File) Encode(w io.Writer) error {
	if !f.V1Missing {
		err := f.V1Header.Write(w)
		if err != nil {
			return fmt.Errorf("write v1 header: %w", err)
		}
		err = f.V1Data.Write(w)
		if err != nil {
			return fmt.Errorf("write v1 data: %w", err)
		}
	}

	if f.V2Header.Version != f.Version {
		return fmt.Errorf("version mismatch: file is %v and v2+ header is %v", f.Version, f.V2Header.Version)
	}

	if f.Version == V2 || f.Version == V3 {
		if err := f.V2Header.Write(w); err != nil {
			return fmt.Errorf("write v2 header: %w", err)
		}
		if err := f.V2Data.Write(w); err != nil {
			return fmt.Errorf("write v2 data: %w", err)
		}
		if err := f.V2Footer.Write(w); err != nil {
			return fmt.Errorf("write v2 footer: %w", err)
		}
	}

	return nil
}

// DecodeFile reads the given TZif data from the given reader.
// v is the version of the data, v1h, v1b, v2h, v2b, and v2f are the parsed data.
// If the version is V1, v2h, v2b, and v2f are empty values.
// If the version is V2 or V3, v1h and v2b are V1 data as the specification requires V1 data to be present always.
func DecodeFile(r io.Reader) (File, error) {
	var f File
	h, err := ReadHeader(r)
	if err != nil {
		return f, fmt.Errorf("read header: %w", err)
	}

	// Strictly speaking, each TZif file needs a V1 header, but we are relaxed in what we accept.
	f.V1Missing = h.Version != V1
	if !f.V1Missing {
		f.Version = V1
		f.V1Header = h
		f.V1Data, err = ReadV1DataBlock(r, h)
		if err != nil {
			return f, fmt.Errorf("read v1 data block: %w", err)
		}

		// Look for V2+ header.
		h, err = ReadHeader(r)
		if errors.Is(err, io.EOF) {
			// No V2+ data, we are done.
			return f, nil
		}
	}

	if h.Version != V2 && h.Version != V3 {
		return f, fmt.Errorf("unsupported version: %v", h.Version)
	}
	f.V2Header = h
	f.Version = h.Version // set max version

	f.V2Data, err = ReadV2DataBlock(r, h)
	if err != nil {
		return f, fmt.Errorf("read v2 data block: %w", err)
	}
	f.V2Footer, err = ReadFooter(r)
	if err != nil {
		return f, fmt.Errorf("read footer: %w", err)
	}

	return f, nil
}
