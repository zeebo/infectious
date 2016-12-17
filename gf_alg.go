// Copyright (C) 2016 Space Monkey, Inc.

package infectuous

import (
	"fmt"
	"unsafe"

	"github.com/spacemonkeygo/errors"
)

//
// basic helpers around gf(2^8) values
//

type gfVal byte

func gfConst(val byte) gfVal {
	return gfVal(val)
}

func (b gfVal) pow(val int) gfVal {
	out := gfVal(1)
	mul_base := gf_mul_table[b][:]
	for i := 0; i < val; i++ {
		out = gfVal(mul_base[out])
	}
	return out
}

func (a gfVal) mul(b gfVal) gfVal {
	return gfVal(gf_mul_table[a][b])
}

func (a gfVal) div(b gfVal) gfVal {
	if b == 0 {
		panic("divide by zero")
	}
	return gfVal(gf_exp[gf_log[a]-gf_log[b]])
}

func (a gfVal) add(b gfVal) gfVal {
	return gfVal(a ^ b)
}

func (a gfVal) isZero() bool {
	return a == 0
}

func (a gfVal) inv() gfVal {
	if a == 0 {
		panic("invert zero")
	}
	return gfVal(gf_exp[255-gf_log[a]])
}

//
// basic helpers about a slice of gf(2^8) values
//

type gfVals []gfVal

func (a gfVals) unsafeBytes() []byte {
	return *(*[]byte)(unsafe.Pointer(&a))
}

func (a gfVals) dot(b gfVals) gfVal {
	out := gfConst(0)
	for i := range a {
		out = out.add(a[i].mul(b[i]))
	}
	return out
}

func (a gfVals) String() string {
	return fmt.Sprintf("%02x", a.unsafeBytes())
}

//
// basic helpers for dealing with polynomials with coefficients in gf(2^8)
//

type gfPoly []gfVal

func polyZero(size int) gfPoly {
	out := make(gfPoly, size)
	for i := range out {
		out[i] = gfConst(0)
	}
	return out
}

func (p gfPoly) isZero() bool {
	for _, coef := range p {
		if !coef.isZero() {
			return false
		}
	}
	return true
}

func (p gfPoly) deg() int {
	return len(p) - 1
}

func (p gfPoly) index(power int) gfVal {
	if power < 0 {
		return gfConst(0)
	}
	which := p.deg() - power
	if which < 0 {
		return gfConst(0)
	}
	return p[which]
}

func (p gfPoly) scale(factor gfVal) gfPoly {
	out := make(gfPoly, len(p))
	for i, coef := range p {
		out[i] = coef.mul(factor)
	}
	return out
}

func (p *gfPoly) set(pow int, coef gfVal) {
	which := p.deg() - pow
	if which < 0 {
		*p = append(polyZero(-which), *p...)
		which = p.deg() - pow
	}
	(*p)[which] = coef
}

func (p gfPoly) add(b gfPoly) gfPoly {
	size := len(p)
	if lb := len(b); lb > size {
		size = lb
	}
	out := make(gfPoly, size)
	for i := range out {
		pi := p.index(i)
		bi := b.index(i)
		out.set(i, pi.add(bi))
	}
	return out
}

func (p gfPoly) div(b gfPoly) (q, r gfPoly, err error) {
	// sanitize the divisor by removing leading zeros.
	for len(b) > 0 && b[0].isZero() {
		b = b[1:]
	}
	if len(b) == 0 {
		return nil, nil, errors.ProgrammerError.New("divide by zero")
	}

	// sanitize the base poly as well
	for len(p) > 0 && p[0].isZero() {
		p = p[1:]
	}
	if len(p) == 0 {
		return polyZero(1), polyZero(1), nil
	}

	for b.deg() <= p.deg() {
		leading_p := p.index(p.deg())
		leading_b := b.index(b.deg())
		coef := leading_p.div(leading_b)

		q = append(q, coef)

		scaled := b.scale(coef)
		padded := append(scaled, polyZero(p.deg()-scaled.deg())...)

		p = p.add(padded)
		if !p[0].isZero() {
			return nil, nil, errors.ProgrammerError.New("alg error: %x", p)
		}
		p = p[1:]
	}

	for len(p) > 1 && p[0].isZero() {
		p = p[1:]
	}

	return q, p, nil
}

func (p gfPoly) eval(x gfVal) gfVal {
	out := gfConst(0)
	for i := 0; i <= p.deg(); i++ {
		x_i := x.pow(i)
		p_i := p.index(i)
		out = out.add(p_i.mul(x_i))
	}
	return out
}

//
// basic helpers for matricies in gf(2^8)
//

type gfMat struct {
	d    gfVals
	r, c int
}

func matrixNew(i, j int) gfMat {
	return gfMat{
		d: make(gfVals, i*j),
		r: i, c: j,
	}
}

func (m gfMat) String() (out string) {
	if m.r == 0 {
		return ""
	}

	for i := 0; i < m.r-1; i++ {
		out += fmt.Sprintln(m.indexRow(i))
	}
	out += fmt.Sprint(m.indexRow(m.r - 1))

	return out
}

func (m gfMat) index(i, j int) int {
	return m.c*i + j
}

func (m gfMat) get(i, j int) gfVal {
	return m.d[m.index(i, j)]
}

func (m gfMat) set(i, j int, val gfVal) {
	m.d[m.index(i, j)] = val
}

func (m gfMat) indexRow(i int) gfVals {
	return m.d[m.index(i, 0):m.index(i+1, 0)]
}

func (m gfMat) swapRow(i, j int) {
	tmp := make(gfVals, m.r)
	ri := m.indexRow(i)
	rj := m.indexRow(j)
	copy(tmp, ri)
	copy(ri, rj)
	copy(rj, tmp)
}

func (m gfMat) scaleRow(i int, val gfVal) {
	ri := m.indexRow(i)
	for i := range ri {
		ri[i] = ri[i].mul(val)
	}
}

func (m gfMat) addmulRow(i, j int, val gfVal) {
	ri := m.indexRow(i)
	rj := m.indexRow(j)
	addmul(rj.unsafeBytes(), ri.unsafeBytes(), byte(val))
}

// in place invert. the output is put into a and m is turned into the identity
// matrix. a is expected to be the identity matrix.
func (m gfMat) invertWith(a gfMat) {
	for i := 0; i < m.r; i++ {
		p_row, p_val := i, m.get(i, i)
		for j := i + 1; j < m.r && p_val.isZero(); j++ {
			p_row, p_val = j, m.get(j, i)
		}
		if p_val.isZero() {
			continue
		}

		if p_row != i {
			m.swapRow(i, p_row)
			a.swapRow(i, p_row)
		}

		inv := p_val.inv()
		m.scaleRow(i, inv)
		a.scaleRow(i, inv)

		for j := i + 1; j < m.r; j++ {
			leading := m.get(j, i)
			m.addmulRow(i, j, leading)
			a.addmulRow(i, j, leading)
		}
	}

	for i := m.r - 1; i > 0; i-- {
		for j := i - 1; j >= 0; j-- {
			trailing := m.get(j, i)
			m.addmulRow(i, j, trailing)
			a.addmulRow(i, j, trailing)
		}
	}
}

// in place standardize.
func (m gfMat) standardize() {
	for i := 0; i < m.r; i++ {
		p_row, p_val := i, m.get(i, i)
		for j := i + 1; j < m.r && p_val.isZero(); j++ {
			p_row, p_val = j, m.get(j, i)
		}
		if p_val.isZero() {
			continue
		}

		if p_row != i {
			m.swapRow(i, p_row)
		}

		inv := p_val.inv()
		m.scaleRow(i, inv)

		for j := i + 1; j < m.r; j++ {
			leading := m.get(j, i)
			m.addmulRow(i, j, leading)
		}
	}

	for i := m.r - 1; i > 0; i-- {
		for j := i - 1; j >= 0; j-- {
			trailing := m.get(j, i)
			m.addmulRow(i, j, trailing)
		}
	}
}

// parity returns the new matrix because it changes dimensions and stuff. it
// can be done in place, but is easier to implement with a copy.
func (m gfMat) parity() gfMat {
	// we assume m is in standard form already
	// it is of form [I_r | P]
	// our output will be [-P_transpose | I_(c - r)]
	// but our field is of characteristic 2 so we do not need the negative.

	// In terms of m:
	// I_r has r rows and r columns.
	// P has r rows and c-r columns.
	// P_transpose has c-r rows, and r columns.
	// I_(c-r) has c-r rows and c-r columns.
	// so: out.r == c-r, out.c == r + c - r == c

	out := matrixNew(m.c-m.r, m.c)

	// step 1. fill in the identity. it starts at column offset r.
	for i := 0; i < m.c-m.r; i++ {
		out.set(i, i+m.r, gfConst(1))
	}

	// step 2: fill in the transposed P matrix. i and j are in terms of out.
	for i := 0; i < m.c-m.r; i++ {
		for j := 0; j < m.r; j++ {
			out.set(i, j, m.get(j, i+m.r))
		}
	}

	return out
}
