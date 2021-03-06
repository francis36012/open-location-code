// Copyright 2015 Tamás Gulácsi. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the 'License');
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an 'AS IS' BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package olc

import (
	"bytes"
	"io/ioutil"
	"math"
	"path/filepath"
	"strconv"
	"sync"
	"testing"

	"github.com/tgulacsi/go/loghlp/tsthlp"
)

var (
	validity []validityTest
	encoding []encodingTest
	shorten  []shortenTest
)

type (
	validityTest struct {
		code                     string
		isValid, isShort, isFull bool
	}

	encodingTest struct {
		code                                 string
		lat, lng, latLo, lngLo, latHi, lngHi float64
	}

	shortenTest struct {
		code     string
		lat, lng float64
		short    string
	}
)

func init() {
	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		for _, cols := range mustReadLines("validity") {
			validity = append(validity, validityTest{
				code:    string(cols[0]),
				isValid: cols[1][0] == 't',
				isShort: cols[2][0] == 't',
				isFull:  cols[3][0] == 't',
			})
		}
	}()

	go func() {
		defer wg.Done()
		for _, cols := range mustReadLines("encoding") {
			encoding = append(encoding, encodingTest{
				code: string(cols[0]),
				lat:  mustFloat(cols[1]), lng: mustFloat(cols[2]),
				latLo: mustFloat(cols[3]), lngLo: mustFloat(cols[4]),
				latHi: mustFloat(cols[5]), lngHi: mustFloat(cols[6]),
			})
		}
	}()

	go func() {
		defer wg.Done()
		for _, cols := range mustReadLines("shortCode") {
			shorten = append(shorten, shortenTest{
				code: string(cols[0]),
				lat:  mustFloat(cols[1]), lng: mustFloat(cols[2]),
				short: string(cols[3]),
			})
		}
	}()
	wg.Wait()
}

func TestCheck(t *testing.T) {
	Log.SetHandler(tsthlp.TestHandler(t))
	for i, elt := range validity {
		err := Check(elt.code)
		got := err == nil
		if got != elt.isValid {
			t.Errorf("%d. %q got %t (%v), awaited %t.", i, elt.code, got, err, elt.isValid)
		}
	}
}

func TestEncode(t *testing.T) {
	Log.SetHandler(tsthlp.TestHandler(t))
	for i, elt := range encoding {
		n := len(stripCode(elt.code))
		code := Encode(elt.lat, elt.lng, n)
		if code != elt.code {
			t.Errorf("%d. got %q for (%v,%v,%d), awaited %q.", i, code, elt.lat, elt.lng, n, elt.code)
			t.FailNow()
		}
	}
}

func TestDecode(t *testing.T) {
	Log.SetHandler(tsthlp.TestHandler(t))
	check := func(i int, code, name string, got, want float64) {
		if !closeEnough(got, want) {
			t.Errorf("%d. %q want %s=%f, got %f", i, code, name, want, got)
			t.FailNow()
		}
	}
	for i, elt := range encoding {
		area, err := Decode(elt.code)
		if err != nil {
			t.Errorf("%d. %q: %v", i, elt.code, err)
			continue
		}
		code := Encode(elt.lat, elt.lng, area.Len)
		if code != elt.code {
			t.Errorf("%d. encode (%f,%f) got %q, awaited %q", i, elt.lat, elt.lng, code, elt.code)
		}
		C := func(name string, got, want float64) {
			check(i, elt.code, name, got, want)
		}
		C("latLo", area.LatLo, elt.latLo)
		C("latHi", area.LatHi, elt.latHi)
		C("lngLo", area.LngLo, elt.lngLo)
		C("lngHi", area.LngHi, elt.lngHi)
	}
}

func TestShorten(t *testing.T) {
	Log.SetHandler(tsthlp.TestHandler(t))
	for i, elt := range shorten {
		got, err := Shorten(elt.code, elt.lat, elt.lng)
		if err != nil {
			t.Errorf("%d. shorten %q: %v", i, elt.code, err)
			t.FailNow()
		}
		if got != elt.short {
			t.Errorf("%d. shorten got %q, awaited %q.", i, got, elt.short)
			t.FailNow()
		}

		got, err = RecoverNearest(got, elt.lat, elt.lng)
		if err != nil {
			t.Errorf("%d. nearest %q: %v", i, got, err)
			t.FailNow()
		}
		if got != elt.code {
			t.Errorf("%d. nearest got %q, awaited %q.", i, got, elt.code)
			t.FailNow()
		}
	}
}

func closeEnough(a, b float64) bool {
	return a == b || math.Abs(a-b) <= 0.0000000001
}

func mustReadLines(name string) [][][]byte {
	rows, err := readLines(filepath.Join("..", "test_data", name+"Tests.csv"))
	if err != nil {
		panic(err)
	}
	return rows
}

func readLines(path string) (rows [][][]byte, err error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	for _, row := range bytes.Split(data, []byte{'\n'}) {
		if j := bytes.IndexByte(row, '#'); j >= 0 {
			row = row[:j]
		}
		row = bytes.TrimSpace(row)
		if len(row) == 0 {
			continue
		}
		rows = append(rows, bytes.Split(row, []byte{','}))
	}
	return rows, nil
}

func mustFloat(a []byte) float64 {
	f, err := strconv.ParseFloat(string(a), 64)
	if err != nil {
		panic(err)
	}
	return f
}
