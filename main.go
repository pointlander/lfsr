// Copyright 2025 The LFSR Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"embed"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"math/bits"
	"net/http"
	"os"
	"sort"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"

	"github.com/pointlander/compress"
)

//go:embed data/*
var Data embed.FS

// LFSRMask is a LFSR mask with a maximum period
const LFSRMask = 0x80000057

// Histogram generates a histogram
func Histogram() {
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

func parity(x uint16) uint16 {
	count := bits.OnesCount16(x)
	if count%2 == 0 {
		return 0
	}
	return 1
}

// Entropy is the entropy of the byte array
func Entropy(in []byte) float64 {
	histogram := make([]int, 256)
	for k := range in {
		histogram[in[k]]++
	}
	entropy := 0.0
	for _, v := range histogram {
		if v == 0 {
			continue
		}
		p := float64(v) / float64(len(in))
		entropy += p * math.Log2(p)
	}
	entropy = -entropy
	return entropy
}

var (
	// FlagFetch fetchs the quantum data
	FlagFetch = flag.Bool("fetch", false, "fetch the quantum data")
	// FlagData is the data to use
	FlagData = flag.String("data", "data/AMillionRandomDigits.bin", "the data to use")
)

func main() {
	flag.Parse()

	if *FlagFetch {
		var quantum []byte
		for len(quantum) < 8*1024 {
			resp, err := http.Get("https://qrng.anu.edu.au/wp-content/plugins/colours-plugin/get_block_binary.php")
			if err != nil {
				panic(err)
			}
			defer resp.Body.Close()

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				panic(err)
			}
			quantum = append(quantum, body...)
			fmt.Println(len(quantum))
		}
		data := make([]byte, 0, 1024)
		b := byte(0)
		for i, v := range quantum {
			if i != 0 && i%8 == 0 {
				data = append(data, b)
				b = 0
			}
			b <<= 1
			if v == '1' {
				b |= 1
			}
		}
		data = append(data, b)
		out, err := os.Create("data/quantum.bin")
		if err != nil {
			panic(err)
		}
		defer out.Close()
		_, err = out.Write(data)
		if err != nil {
			panic(err)
		}
		return
	}

	file, err := Data.Open(*FlagData)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		panic(err)
	}
	initial := Entropy(data[:1024])
	fmt.Println("Entropy(data)", initial)
	min := math.MaxFloat64
	for i := 0; i < math.MaxUint16; i++ {
		mask := uint16(0x8000 | i)
		for j := 0; j < math.MaxUint16; j++ {
			lfsr, output := uint16(1), make([]byte, 1024)
			for k := range output {
				var t byte
				for l := 0; l < 8; l++ {
					lfsr = (lfsr >> 1) ^ (-(lfsr & 1) & mask)
					t |= uint8(parity(lfsr&uint16(j))) << l
				}
				output[k] = data[k] ^ t
			}
			entropy := Entropy(output[:1024])
			if entropy < min {
				buffer := bytes.Buffer{}
				compress.Mark1Compress1(output, &buffer)
				fmt.Println(entropy, (initial-entropy)*1024, buffer.Len()-1024)
				min = entropy
			}
		}
	}
}
