package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"

	"github.com/google/go-cmp/cmp"
	"github.com/ngrash/go-tz/tzif"
)

func main() {
	if err := run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run() error {
	flag.Parse()
	args := flag.Args()
	if len(args) != 2 {
		return fmt.Errorf("Usage: tzdiff <tzdata file A> <tzdata file B>\n")
	}

	af, err := os.ReadFile(args[0])
	if err != nil {
		return err
	}

	bf, err := os.ReadFile(args[1])
	if err != nil {
		return err
	}

	adata, err := tzif.DecodeData(bytes.NewReader(af))
	if err != nil {
		return err
	}

	bdata, err := tzif.DecodeData(bytes.NewReader(bf))
	if err != nil {
		return err
	}

	if diff := cmp.Diff(adata, bdata); diff != "" {
		fmt.Println("files are different: -A +B")
		fmt.Println(diff)
	} else {
		fmt.Println("files are identical")
	}

	return nil
}
