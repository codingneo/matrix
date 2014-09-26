// Copyright ©2014 The gonum Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build pool

package mat64

import (
	check "launchpad.net/gocheck"
	"math"
	"testing"
)

func (s *S) TestPool(c *check.C) {
	for i := 1; i < 10; i++ {
		for j := 1; j < 10; j++ {
			var m *Dense
			for k := 0; k < 5; k++ {
				m = get(i, j, true)
				c.Check(m.mat, check.DeepEquals, NewDense(i, j, nil).mat)
				c.Check(cap(m.mat.Data) < 2*len(m.mat.Data), check.Equals, true, check.Commentf("r: %d c: %d -> len: %d cap: %d", i, j, len(m.mat.Data), cap(m.mat.Data)))
			}
			m.Set(0, 0, math.NaN())
			for k := 0; k < 5; k++ {
				put(m)
			}
			for k := 0; k < 5; k++ {
				m = get(i, j, true)
				c.Check(m.mat, check.DeepEquals, NewDense(i, j, nil).mat)
				c.Check(cap(m.mat.Data) < 2*len(m.mat.Data), check.Equals, true, check.Commentf("r: %d c: %d -> len: %d cap: %d", i, j, len(m.mat.Data), cap(m.mat.Data)))
			}
			m.Set(0, 0, math.NaN())
			for k := 0; k < 5; k++ {
				put(m)
			}
			for k := 0; k < 5; k++ {
				m = get(i, j, false)
				c.Check(math.IsNaN(m.At(0, 0)), check.Equals, true)
			}
		}
	}
}

var benchmat *Dense

func poolBenchmark(n, r, c int, clear bool) {
	for i := 0; i < n; i++ {
		benchmat = get(r, c, clear)
		put(benchmat)
	}
}

func newBenchmark(n, r, c int) {
	for i := 0; i < n; i++ {
		benchmat = NewDense(r, c, nil)
	}
}

func BenchmarkPool10by10Uncleared(b *testing.B)   { poolBenchmark(b.N, 10, 10, false) }
func BenchmarkPool10by10Cleared(b *testing.B)     { poolBenchmark(b.N, 10, 10, true) }
func BenchmarkNew10by10(b *testing.B)             { newBenchmark(b.N, 10, 10) }
func BenchmarkPool100by100Uncleared(b *testing.B) { poolBenchmark(b.N, 100, 100, false) }
func BenchmarkPool100by100Cleared(b *testing.B)   { poolBenchmark(b.N, 100, 100, true) }
func BenchmarkNew100by100(b *testing.B)           { newBenchmark(b.N, 100, 100) }
