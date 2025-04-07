// Copyright 2025 The LFSR Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"embed"
	"fmt"
	"io"
	"sort"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

//go:embed data/*
var Data embed.FS

// LFSRMask is a LFSR mask with a maximum period
const LFSRMask = 0x80000057

func main() {
	file, err := Data.Open("data/AMillionRandomDigits.bin")
	if err != nil {
		panic(err)
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		panic(err)
	}
	_ = data

	type Bucket struct {
		Index int
		Count int
	}
	buckets := make([]Bucket, 256)
	for i := range buckets {
		buckets[i].Index = i
	}
	var values plotter.Values
	for mask := byte(0x80); mask != 0; mask++ {
		length, lfsr := 0, byte(1)
		for {
			lfsr = (lfsr >> 1) ^ (-(lfsr & 1) & mask)
			values = append(values, float64(lfsr))
			buckets[lfsr].Count++
			if lfsr == 1 {
				break
			}
			length++
		}
		fmt.Println(length)
	}
	p := plot.New()
	p.Title.Text = "histogram plot"

	hist, err := plotter.NewHist(values, 256)
	if err != nil {
		panic(err)
	}
	p.Add(hist)

	if err := p.Save(8*vg.Inch, 8*vg.Inch, "histogram.png"); err != nil {
		panic(err)
	}

	sort.Slice(buckets, func(i, j int) bool {
		return buckets[i].Count < buckets[j].Count
	})
	for _, v := range buckets {
		fmt.Println(v)
	}
}
