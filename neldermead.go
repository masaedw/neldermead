package neldermead

import (
	"math"
	"sort"

	"github.com/pkg/errors"
)

// Vec []float64
type Vec []float64

func (x Vec) sub(v Vec) Vec {
	for i := range x {
		x[i] -= v[i]
	}
	return x
}

func (x Vec) mulScalar(v float64) Vec {
	for i := range x {
		x[i] *= v
	}
	return x
}

func (x Vec) add(v Vec) Vec {
	for i := range x {
		x[i] += v[i]
	}
	return x
}

// Function Objective function
type Function func(x Vec) float64

func (f Function) apply(x []Vec) []float64 {
	n := len(x)
	r := make([]float64, n)
	for i := 0; i < n; i++ {
		r[i] = f(x[i])
	}
	return r
}

type byPerm struct {
	src []float64
	ord []int
}

func (b byPerm) Len() int {
	return len(b.src)
}

func (b byPerm) Swap(i, j int) {
	b.ord[i], b.ord[j] = b.ord[j], b.ord[i]
}

func (b byPerm) Less(i, j int) bool {
	return b.src[b.ord[i]] < b.src[b.ord[j]]
}

func sortPerm(src []float64) []int {
	perm := byPerm{
		src: src,
		ord: make([]int, len(src)),
	}
	for i := range perm.ord {
		perm.ord[i] = i
	}

	sort.Sort(perm)

	return perm.ord
}

func ifelse(c bool, t, e float64) float64 {
	if c {
		return t
	}
	return e
}

func dup(x Vec) Vec {
	r := make(Vec, len(x))
	copy(r, x)
	return r
}

func fillzero(x Vec) {
	for i := range x {
		x[i] = 0
	}
}

func centroidInPlace(c Vec, x []Vec, h int) {
	fillzero(c)

	n := len(c)
	for i := 0; i <= n; i++ {
		if i == h {
			continue
		}

		xi := x[i]
		for j := range xi {
			c[j] += xi[j]
		}
	}

	for j := range c {
		c[j] /= float64(n)
	}
}

func centroid(x []Vec, h int) Vec {
	c := make(Vec, len(x[0]))
	centroidInPlace(c, x, h)
	return c
}

func similar(x Vec) Vec {
	r := make(Vec, len(x))
	return r
}

func initialSimplex(x0 Vec) []Vec {
	n := len(x0)
	s := make([]Vec, n+1)
	s[0] = dup(x0)
	for i := 0; i < n; i++ {
		s[i+1] = dup(x0)
		s[i+1][i] = ifelse(x0[i] != 0, 0.05, 0.00025)
	}

	return s
}

type vPair struct {
	x Vec
	v float64
}

// Execute Adaptive Nelder-Mead Simplex (ANMS) algorithm
func Execute(f Function, x0 Vec) (Vec, float64, error) {
	iterations := 1000000
	ftol := 1.0e-8
	xtol := 1.0e-8

	n := len(x0)

	if n < 2 {
		return nil, 0, errors.New("multivariate function is needed")
	}

	α := 1.0
	β := 1.0 + 2/float64(n)
	γ := 0.75 - 1/2*float64(n)
	δ := 1.0 - 1/float64(n)

	simplex := initialSimplex(x0)
	fvalues := f.apply(simplex)
	ord := sortPerm(fvalues)

	iter := 0
	domconv := false
	fvalconv := false

	c := centroid(simplex, ord[n])

	xr, xe, xc := similar(x0), similar(x0), similar(x0)

	for iter < iterations && !(fvalconv && domconv) {
		// highest, second highest, and lowest indices, respectively
		h, s, l := ord[n], ord[n-1], ord[0]

		xh := simplex[h]
		fh := fvalues[h]
		fs := fvalues[s]
		xl := simplex[l]
		fl := fvalues[l]

		var accept vPair

		// reflect
		for j := range xr {
			xr[j] = c[j] + α*(c[j]-xh[j])
		}
		fr := f(xr)
		doshrink := false

		if fr < fl {
			//expand
			for j := range xe {
				xe[j] = c[j] + β*(xr[j]-c[j])
			}
			fe := f(xe)
			if fe < fr {
				accept = vPair{xe, fe}
			} else {
				accept = vPair{xr, fr}
			}
		} else if fr < fs {
			accept = vPair{xr, fr}
		} else { // fs <= fr
			// contract
			if fr < fh {
				// outside
				for j := range xc {
					xc[j] = c[j] + γ*(xr[j]-c[j])
				}
				fc := f(xc)
				if fc <= fr {
					accept = vPair{xc, fc}
				} else {
					doshrink = true
				}
			} else {
				// inside
				for j := range xc {
					xc[j] = c[j] - γ*(xr[j]-c[j])
				}
				fc := f(xc)
				if fc < fh {
					accept = vPair{xc, fc}
				} else {
					doshrink = true
				}
			}

			// shrinkage almost never happen in practice
			if doshrink {
				// shrink
				for i := 1; i < n; i++ {
					o := ord[i]
					xi := simplex[o].sub(xl).mulScalar(δ).add(xl)
					fvalues[o] = f(xi)
				}
			}
		}

		// update simplex, function values and centroid cache

		if doshrink {
			ord = sortPerm(fvalues)
			centroidInPlace(c, simplex, ord[n])
		} else {
			x, fvalue := accept.x, accept.v
			for j := range x {
				simplex[h][j] = x[j]
			}

			// insert the new function value into an ordered position
			fvalues[h] = fvalue
			for i := n; 1 <= i; i-- {
				if fvalues[ord[i-1]] > fvalues[ord[i]] {
					ord[i-1], ord[i] = ord[i], ord[i-1]
				} else {
					break
				}
			}

			// add the new vertex, and subtract the highest vertex
			h = ord[n]
			xh = simplex[h]
			for j := range c {
				c[j] += (x[j] - xh[j]) / float64(n)
			}
		}

		l = ord[0]
		xl = simplex[0]
		fl = fvalues[0]

		// check convergence
		fvalconv = true
		for i := 1; i <= n; i++ {
			if math.Abs(fvalues[i]-fl) > ftol {
				fvalconv = false
				break
			}
		}

		domconv = true
	dom:
		for i := 1; i <= n; i++ {
			for j := 0; j < n; j++ {
				if math.Abs(simplex[i][j]-xl[j]) > xtol {
					domconv = false
					break dom
				}
			}
		}

		iter++
	}

	// return the lowest vertex (or the centroid of the simplex) and the function value
	centroidInPlace(c, simplex, -1)
	fcent := f(c)
	o := ord[0]
	if fcent < fvalues[o] {
		return c, fcent, nil
	}

	return simplex[o], fvalues[o], nil
}