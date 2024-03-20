package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"zic/tzif"
)

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) != 1 {
		fmt.Println("Usage: tzinspect <tzif file>")
		os.Exit(1)
	}
	b, err := os.ReadFile(args[0])
	if err != nil {
		fmt.Println("reading file:", err)
		os.Exit(1)
	}

	r := bytes.NewReader(b)
	data, err := tzif.DecodeData(r)
	if err != nil {
		fmt.Println("decoding:", err)
		os.Exit(1)
	}

	printHeader(data.V1Header)
	printV1DataBlock(data.V1Data)
	if data.Version > tzif.V1 {
		printHeader(data.V2Header)
		printV2DataBlock(data.V2Data)
		fmt.Println("Footer", string(data.V2Footer.TZString))
	}
	printRest(r)
}

func printRest(r *bytes.Reader) {
	if r.Len() == 0 {
		return
	}
	rest, err := io.ReadAll(r)
	if err != nil {
		fmt.Println("reading remaining data:", err)
		os.Exit(1)
	}
	fmt.Println("remaining data:", len(rest), "bytes")
	fmt.Println(string(rest))
}

func printHeader(h tzif.Header) {
	fmt.Println("Header", h.Version)
	fmt.Println("  Isutcnt", h.Isutcnt)
	fmt.Println("  Isutcnt", h.Isutcnt)
	fmt.Println("  Leapcnt", h.Leapcnt)
	fmt.Println("  Timecnt", h.Timecnt)
	fmt.Println("  Typecnt", h.Typecnt)
	fmt.Println("  Charcnt", h.Charcnt)
	fmt.Println()
}

func printV1DataBlock(b tzif.V1DataBlock) {
	fmt.Println("Data block", tzif.V1)
	fmt.Println("  TransitionTimes: ", len(b.TransitionTimes))
	fmt.Println("  TransitionTypes: ", len(b.TransitionTypes))
	fmt.Println("  LocalTimeTypeRecord: ", len(b.LocalTimeTypeRecord))
	fmt.Println(" ", len(b.TimeZoneDesignation), "TimeZoneDesignation:", string(b.TimeZoneDesignation))
	fmt.Println("  LeapSecondRecords: ", len(b.LeapSecondRecords))
	fmt.Println("  StandardWallIndicators: ", len(b.StandardWallIndicators))
	fmt.Println("  UTLocalIndicators: ", len(b.UTLocalIndicators))
	fmt.Println()
}

func printV2DataBlock(b tzif.V2DataBlock) {
	fmt.Println("Data block", tzif.V2)
	fmt.Println("  TransitionTimes: ", len(b.TransitionTimes))
	fmt.Println("  TransitionTypes: ", len(b.TransitionTypes))
	fmt.Println("  LocalTimeTypeRecord: ", len(b.LocalTimeTypeRecord))
	fmt.Println(" ", len(b.TimeZoneDesignation), "TimeZoneDesignation:", string(b.TimeZoneDesignation))
	fmt.Println("  LeapSecondRecords: ", len(b.LeapSecondRecords))
	fmt.Println("  StandardWallIndicators: ", len(b.StandardWallIndicators))
	fmt.Println("  UTLocalIndicators: ", len(b.UTLocalIndicators))
	fmt.Println()
}
