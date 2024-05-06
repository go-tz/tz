package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ngrash/go-tz/tzif"
)

var (
	printV1Flag          = flag.Bool("v1", false, "Always print v1 header and data")
)

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) != 1 {
		fmt.Println("Usage: tzinfo <tzif file>")
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

	printData(data)
	printRest(r)
}

func printData(d tzif.Data) {
	if d.Version == tzif.V1 || *printV1Flag {
		printV1(d.V1Header, d.V1Data)
	}
	if d.Version > tzif.V1 {
		printV2(d.V2Header, d.V2Data, d.V2Footer)
	}
}

func printFooter(f tzif.Footer) {
	fmt.Println("Footer")
	fmt.Println("  TZString =", string(f.TZString))
	fmt.Println()
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
	fmt.Println("Header")
	fmt.Println("  version =", h.Version)
	fmt.Println("  isutcnt =", h.Isutcnt)
	fmt.Println("  isutcnt =", h.Isutcnt)
	fmt.Println("  leapcnt =", h.Leapcnt)
	fmt.Println("  timecnt =", h.Timecnt)
	fmt.Println("  typecnt =", h.Typecnt)
	fmt.Println("  charcnt =", h.Charcnt)
	fmt.Println()
}

func printV1(h tzif.Header, b tzif.V1DataBlock) {
	printHeader(h)

	fmt.Println("Data block", tzif.V1)
	fmt.Printf("  TransitionTimes (%d) = %v\n", len(b.TransitionTimes), b.TransitionTimes)
	fmt.Printf("  TransitionTypes (%d) = %v\n", len(b.TransitionTypes), b.TransitionTypes)
	fmt.Printf("  LocalTimeTypeRecord (%d) = %+v\n", len(b.LocalTimeTypeRecord), b.LocalTimeTypeRecord)
	fmt.Printf("  TimeZoneDesignation (%d) = %v\n", len(b.TimeZoneDesignation), strings.Split(string(b.TimeZoneDesignation), "\x00"))
	fmt.Printf("  LeapSecondRecords (%d) = %+v\n", len(b.LeapSecondRecords), b.LeapSecondRecords)
	fmt.Printf("  StandardWallIndicators (%d) = %v\n", len(b.StandardWallIndicators), b.StandardWallIndicators)
	fmt.Printf("  UTLocalIndicators (%d) = %v\n", len(b.UTLocalIndicators), b.UTLocalIndicators)
	fmt.Println()
}

func printV2(h tzif.Header, b tzif.V2DataBlock, f tzif.Footer) {
	printHeader(h)

	fmt.Println("Data block", h.Version)
	fmt.Printf("  TransitionTimes (%d) = %v\n", len(b.TransitionTimes), b.TransitionTimes)
	fmt.Printf("  TransitionTypes (%d) = %v\n", len(b.TransitionTypes), b.TransitionTypes)
	fmt.Printf("  LocalTimeTypeRecord (%d) = %+v\n", len(b.LocalTimeTypeRecord), b.LocalTimeTypeRecord)
	fmt.Printf("  TimeZoneDesignation (%d) = %v\n", len(b.TimeZoneDesignation), strings.Split(string(b.TimeZoneDesignation), "\x00"))
	fmt.Printf("  LeapSecondRecords (%d) = %+v\n", len(b.LeapSecondRecords), b.LeapSecondRecords)
	fmt.Printf("  StandardWallIndicators (%d) = %v\n", len(b.StandardWallIndicators), b.StandardWallIndicators)
	fmt.Printf("  UTLocalIndicators (%d) = %v\n", len(b.UTLocalIndicators), b.UTLocalIndicators)
	fmt.Println()

	printFooter(f)
}
