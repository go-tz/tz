package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"zic/tzif"
)

func main() {
	flag.Parse()

	args := flag.Args()
	if len(args) != 1 {
		fmt.Println("Usage: tzinspect <tzif file>")
		flag.Usage()
	}
	data, err := os.ReadFile(args[0])
	if err != nil {
		fmt.Println("reading file:", err)
		os.Exit(1)
	}
	f, err := tzif.DecodeFile(bytes.NewReader(data))
	if err != nil {
		fmt.Println("decoding:", err)
		os.Exit(1)
	}
	fmt.Println("Version:", f.Version)
}
