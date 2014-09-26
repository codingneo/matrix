// Copyright ©2014 The gonum Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build pool

package mat64

import (
	"sync"
)

var tab64 = [64]byte{
	0x3f, 0x00, 0x3a, 0x01, 0x3b, 0x2f, 0x35, 0x02,
	0x3c, 0x27, 0x30, 0x1b, 0x36, 0x21, 0x2a, 0x03,
	0x3d, 0x33, 0x25, 0x28, 0x31, 0x12, 0x1c, 0x14,
	0x37, 0x1e, 0x22, 0x0b, 0x2b, 0x0e, 0x16, 0x04,
	0x3e, 0x39, 0x2e, 0x34, 0x26, 0x1a, 0x20, 0x29,
	0x32, 0x24, 0x11, 0x13, 0x1d, 0x0a, 0x0d, 0x15,
	0x38, 0x2d, 0x19, 0x1f, 0x23, 0x10, 0x09, 0x0c,
	0x2c, 0x18, 0x0f, 0x08, 0x17, 0x07, 0x06, 0x05,
}

// bits returns the ceiling of base 2 log of v.
// Approach based on http://stackoverflow.com/a/11398748.
func bits(v uint64) byte {
	v--
	if v == 0 {
		return 0
	}
	v <<= 2
	v |= v >> 1
	v |= v >> 2
	v |= v >> 4
	v |= v >> 8
	v |= v >> 16
	v |= v >> 32
	return tab64[((v-(v>>1))*0x07EDD5E59A4E28C2)>>58] - 1
}

var pool [63]sync.Pool

func init() {
	for i := range pool {
		l := 1 << uint(i)
		pool[i].New = func() interface{} {
			return &Dense{RawMatrix{
				Data: make([]float64, l),
			}}
		}
	}
}

func get(r, c int, clear bool) *Dense {
	l := uint64(r * c)
	m := pool[bits(l)].Get().(*Dense)
	m.mat.Data = m.mat.Data[:l]
	if clear {
		zero(m.mat.Data)
	}
	m.mat.Rows = r
	m.mat.Cols = c
	m.mat.Stride = c
	return m
}

func put(m *Dense) {
	pool[bits(uint64(cap(m.mat.Data)))].Put(m)
}
