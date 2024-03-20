package tzif

import (
	"fmt"
	"io"
)

// Data represents a TZif file.
type Data struct {
	Version Version

	V1Header Header
	V1Data   V1DataBlock

	V2Header Header
	V2Data   V2DataBlock
	V2Footer Footer
}

// Encode writes the given TZif data to the given writer.
// If the version is V1, the V2 fields are not written.
func (d Data) Encode(w io.Writer) error {
	if err := d.V1Header.Write(w); err != nil {
		return fmt.Errorf("write v1 header: %w", err)
	}
	if err := d.V1Data.Write(w); err != nil {
		return fmt.Errorf("write v1 data: %w", err)
	}
	if d.Version > V1 {
		if err := d.V2Header.Write(w); err != nil {
			return fmt.Errorf("write v2 header: %w", err)
		}
		if err := d.V2Data.Write(w); err != nil {
			return fmt.Errorf("write v2 data: %w", err)
		}
		if err := d.V2Footer.Write(w); err != nil {
			return fmt.Errorf("write v2 footer: %w", err)
		}
	}
	return nil
}

// DecodeData reads the TZif Data from the given reader.
// If the version is V1, the V2 fields should be ignored.
func DecodeData(r io.Reader) (Data, error) {
	var (
		d   Data
		err error
	)
	d.V1Header, err = ReadHeader(r)
	if err != nil {
		return d, fmt.Errorf("read v1 header: %w", err)
	}
	d.Version = d.V1Header.Version

	d.V1Data, err = ReadV1DataBlock(r, d.V1Header)
	if err != nil {
		return d, fmt.Errorf("read v1 data block: %w", err)
	}

	if d.Version > V1 {
		d.V2Header, err = ReadHeader(r)
		if err != nil {
			return d, fmt.Errorf("read v2 header: %w", err)
		}
		d.V2Data, err = ReadV2DataBlock(r, d.V2Header)
		if err != nil {
			return d, fmt.Errorf("read v2 data block: %w", err)
		}
		d.V2Footer, err = ReadFooter(r)
		if err != nil {
			return d, fmt.Errorf("read footer: %w", err)
		}
	}

	return d, nil
}
