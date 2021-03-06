// Copyright ©2013 The gonum Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// Based on the EigenvalueDecomposition class from Jama 1.0.3.

package mat64

import (
	"math"
)

func symmetric(m *Dense) bool {
	n, _ := m.Dims()
	for i := 0; i < n; i++ {
		for j := 0; j < i; j++ {
			if m.At(i, j) != m.At(j, i) {
				return false
			}
		}
	}
	return true
}

type EigenFactors struct {
	V    *Dense
	d, e []float64
}

// Eigen returns the Eigenvalues and eigenvectors of a square real matrix.
// The matrix a is overwritten during the decomposition. If a is symmetric,
// then a = v*D*v' where the eigenvalue matrix D is diagonal and the
// eigenvector matrix v is orthogonal.
//
// If a is not symmetric, then the eigenvalue matrix D is block diagonal
// with the real eigenvalues in 1-by-1 blocks and any complex eigenvalues,
// lambda + i*mu, in 2-by-2 blocks, [lambda, mu; -mu, lambda]. The
// columns of v represent the eigenvectors in the sense that a*v = v*D,
// i.e. a.v equals v.D. The matrix v may be badly conditioned, or even
// singular, so the validity of the equation a = v*D*inverse(v) depends
// upon the 2-norm condition number of v.
func Eigen(a *Dense, epsilon float64) EigenFactors {
	m, n := a.Dims()
	if m != n {
		panic(ErrSquare)
	}

	var v *Dense
	d := make([]float64, n)
	e := make([]float64, n)

	if symmetric(a) {
		// Tridiagonalize.
		v = tred2(a, d, e)

		// Diagonalize.
		tql2(d, e, v, epsilon)
	} else {
		// Reduce to Hessenberg form.
		var hess *Dense
		hess, v = orthes(a)

		// Reduce Hessenberg to real Schur form.
		hqr2(d, e, hess, v, epsilon)
	}

	return EigenFactors{v, d, e}
}

// Symmetric Householder reduction to tridiagonal form.
//
// This is derived from the Algol procedures tred2 by
// Bowdler, Martin, Reinsch, and Wilkinson, Handbook for
// Auto. Comp., Vol.ii-Linear Algebra, and the corresponding
// Fortran subroutine in EISPACK.
func tred2(a *Dense, d, e []float64) (v *Dense) {
	n := len(d)
	v = a

	for j := 0; j < n; j++ {
		d[j] = v.At(n-1, j)
	}

	// Householder reduction to tridiagonal form.
	for i := n - 1; i > 0; i-- {
		// Scale to avoid under/overflow.
		var (
			scale float64
			h     float64
		)
		for k := 0; k < i; k++ {
			scale += math.Abs(d[k])
		}
		if scale == 0 {
			e[i] = d[i-1]
			for j := 0; j < i; j++ {
				d[j] = v.At(i-1, j)
				v.Set(i, j, 0)
				v.Set(j, i, 0)
			}
		} else {
			// Generate Householder vector.
			for k := 0; k < i; k++ {
				d[k] /= scale
				h += d[k] * d[k]
			}
			f := d[i-1]
			g := math.Sqrt(h)
			if f > 0 {
				g = -g
			}
			e[i] = scale * g
			h -= f * g
			d[i-1] = f - g
			for j := 0; j < i; j++ {
				e[j] = 0
			}

			// Apply similarity transformation to remaining columns.
			for j := 0; j < i; j++ {
				f = d[j]
				v.Set(j, i, f)
				g = e[j] + v.At(j, j)*f
				for k := j + 1; k <= i-1; k++ {
					g += v.At(k, j) * d[k]
					e[k] += v.At(k, j) * f
				}
				e[j] = g
			}
			f = 0
			for j := 0; j < i; j++ {
				e[j] /= h
				f += e[j] * d[j]
			}
			hh := f / (h + h)
			for j := 0; j < i; j++ {
				e[j] -= hh * d[j]
			}
			for j := 0; j < i; j++ {
				f = d[j]
				g = e[j]
				for k := j; k <= i-1; k++ {
					v.Set(k, j, v.At(k, j)-(f*e[k]+g*d[k]))
				}
				d[j] = v.At(i-1, j)
				v.Set(i, j, 0)
			}
		}
		d[i] = h
	}

	// Accumulate transformations.
	for i := 0; i < n-1; i++ {
		v.Set(n-1, i, v.At(i, i))
		v.Set(i, i, 1)
		h := d[i+1]
		if h != 0 {
			for k := 0; k <= i; k++ {
				d[k] = v.At(k, i+1) / h
			}
			for j := 0; j <= i; j++ {
				var g float64
				for k := 0; k <= i; k++ {
					g += v.At(k, i+1) * v.At(k, j)
				}
				for k := 0; k <= i; k++ {
					v.Set(k, j, v.At(k, j)-g*d[k])
				}
			}
		}
		for k := 0; k <= i; k++ {
			v.Set(k, i+1, 0)
		}
	}
	for j := 0; j < n; j++ {
		d[j] = v.At(n-1, j)
		v.Set(n-1, j, 0)
	}
	v.Set(n-1, n-1, 1)
	e[0] = 0

	return v
}

// Symmetric tridiagonal QL algorithm.
//
// This is derived from the Algol procedures tql2, by
// Bowdler, Martin, Reinsch, and Wilkinson, Handbook for
// Auto. Comp., Vol.ii-Linear Algebra, and the corresponding
// Fortran subroutine in EISPACK.
func tql2(d, e []float64, v *Dense, epsilon float64) {
	n := len(d)
	for i := 1; i < n; i++ {
		e[i-1] = e[i]
	}
	e[n-1] = 0

	var (
		f    float64
		tst1 float64
	)
	for l := 0; l < n; l++ {
		// Find small subdiagonal element
		tst1 = math.Max(tst1, math.Abs(d[l])+math.Abs(e[l]))
		m := l
		for m < n {
			if math.Abs(e[m]) <= epsilon*tst1 {
				break
			}
			m++
		}

		// If m == l, d[l] is an eigenvalue, otherwise, iterate.
		if m > l {
			for iter := 0; ; iter++ { // Could check iteration count here.

				// Compute implicit shift
				g := d[l]
				p := (d[l+1] - g) / (2 * e[l])
				r := math.Hypot(p, 1)
				if p < 0 {
					r = -r
				}
				d[l] = e[l] / (p + r)
				d[l+1] = e[l] * (p + r)
				dl1 := d[l+1]
				h := g - d[l]
				for i := l + 2; i < n; i++ {
					d[i] -= h
				}
				f += h

				// Implicit QL transformation.
				p = d[m]
				c := 1.
				c2 := c
				c3 := c
				el1 := e[l+1]
				var (
					s  float64
					s2 float64
				)
				for i := m - 1; i >= l; i-- {
					c3 = c2
					c2 = c
					s2 = s
					g = c * e[i]
					h = c * p
					r = math.Hypot(p, e[i])
					e[i+1] = s * r
					s = e[i] / r
					c = p / r
					p = c*d[i] - s*g
					d[i+1] = h + s*(c*g+s*d[i])

					// Accumulate transformation.
					for k := 0; k < n; k++ {
						h = v.At(k, i+1)
						v.Set(k, i+1, s*v.At(k, i)+c*h)
						v.Set(k, i, c*v.At(k, i)-s*h)
					}
				}
				p = -s * s2 * c3 * el1 * e[l] / dl1
				e[l] = s * p
				d[l] = c * p

				// Check for convergence.
				if math.Abs(e[l]) <= epsilon*tst1 {
					break
				}
			}
		}
		d[l] += f
		e[l] = 0
	}

	// Sort eigenvalues and corresponding vectors.
	for i := 0; i < n-1; i++ {
		k := i
		p := d[i]
		for j := i + 1; j < n; j++ {
			if d[j] < p {
				k = j
				p = d[j]
			}
		}
		if k != i {
			d[k] = d[i]
			d[i] = p
			for j := 0; j < n; j++ {
				p = v.At(j, i)
				v.Set(j, i, v.At(j, k))
				v.Set(j, k, p)
			}
		}
	}
}

// Nonsymmetric reduction to Hessenberg form.
//
// This is derived from the Algol procedures orthes and ortran,
// by Martin and Wilkinson, Handbook for Auto. Comp.,
// Vol.ii-Linear Algebra, and the corresponding
// Fortran subroutines in EISPACK.
func orthes(a *Dense) (hess, v *Dense) {
	n, _ := a.Dims()
	hess = a

	ort := make([]float64, n)

	low := 0
	high := n - 1

	for m := low + 1; m <= high-1; m++ {
		// Scale column.
		var scale float64
		for i := m; i <= high; i++ {
			scale += math.Abs(hess.At(i, m-1))
		}
		if scale != 0 {
			// Compute Householder transformation.
			var h float64
			for i := high; i >= m; i-- {
				ort[i] = hess.At(i, m-1) / scale
				h += ort[i] * ort[i]
			}
			g := math.Sqrt(h)
			if ort[m] > 0 {
				g = -g
			}
			h -= ort[m] * g
			ort[m] -= g

			// Apply Householder similarity transformation
			// hess = (I-u*u'/h)*hess*(I-u*u')/h)
			for j := m; j < n; j++ {
				var f float64
				for i := high; i >= m; i-- {
					f += ort[i] * hess.At(i, j)
				}
				f /= h
				for i := m; i <= high; i++ {
					hess.Set(i, j, hess.At(i, j)-f*ort[i])
				}
			}

			for i := 0; i <= high; i++ {
				var f float64
				for j := high; j >= m; j-- {
					f += ort[j] * hess.At(i, j)
				}
				f /= h
				for j := m; j <= high; j++ {
					hess.Set(i, j, hess.At(i, j)-f*ort[j])
				}
			}
			ort[m] *= scale
			hess.Set(m, m-1, scale*g)
		}
	}

	// Accumulate transformations (Algol's ortran).
	v = NewDense(n, n, nil)
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if i == j {
				v.Set(i, j, 1)
			} else {
				v.Set(i, j, 0)
			}
		}
	}
	for m := high - 1; m >= low+1; m-- {
		if hess.At(m, m-1) != 0 {
			for i := m + 1; i <= high; i++ {
				ort[i] = hess.At(i, m-1)
			}
			for j := m; j <= high; j++ {
				var g float64
				for i := m; i <= high; i++ {
					g += ort[i] * v.At(i, j)
				}

				// Double division avoids possible underflow
				g = (g / ort[m]) / hess.At(m, m-1)
				for i := m; i <= high; i++ {
					v.Set(i, j, v.At(i, j)+g*ort[i])
				}
			}
		}
	}

	return hess, v
}

func cdiv(xr, xi, yr, yi float64) (float64, float64) {
	r := complex(xr, xi) / complex(yr, yi)
	return real(r), imag(r)
}

// Nonsymmetric reduction from Hessenberg to real Schur form.
//
// This is derived from the Algol procedure hqr2,
// by Martin and Wilkinson, Handbook for Auto. Comp.,
// Vol.ii-Linear Algebra, and the corresponding
// Fortran subroutine in EISPACK.
func hqr2(d, e []float64, hess, v *Dense, epsilon float64) {
	// Initialize
	nn := len(d)
	n := nn - 1

	low := 0
	high := n

	var exshift, p, q, r, s, z, t, w, x, y float64

	// Store roots isolated by balanc and compute matrix norm
	var norm float64
	for i := 0; i < nn; i++ {
		if i < low || i > high {
			d[i] = hess.At(i, i)
			e[i] = 0
		}
		for j := max(i-1, 0); j < nn; j++ {
			norm += math.Abs(hess.At(i, j))
		}
	}

	// Outer loop over eigenvalue index
	for iter := 0; n >= low; {
		// Look for single small sub-diagonal element
		l := n
		for l > low {
			s = math.Abs(hess.At(l-1, l-1)) + math.Abs(hess.At(l, l))
			if s == 0 {
				s = norm
			}
			if math.Abs(hess.At(l, l-1)) < epsilon*s {
				break
			}
			l--
		}

		// Check for convergence
		if l == n {
			// One root found
			hess.Set(n, n, hess.At(n, n)+exshift)
			d[n] = hess.At(n, n)
			e[n] = 0
			n--
			iter = 0
		} else if l == n-1 {
			// Two roots found
			w = hess.At(n, n-1) * hess.At(n-1, n)
			p = (hess.At(n-1, n-1) - hess.At(n, n)) / 2.0
			q = p*p + w
			z = math.Sqrt(math.Abs(q))
			hess.Set(n, n, hess.At(n, n)+exshift)
			hess.Set(n-1, n-1, hess.At(n-1, n-1)+exshift)
			x = hess.At(n, n)

			// Real pair
			if q >= 0 {
				if p >= 0 {
					z = p + z
				} else {
					z = p - z
				}
				d[n-1] = x + z
				d[n] = d[n-1]
				if z != 0 {
					d[n] = x - w/z
				}
				e[n-1] = 0
				e[n] = 0
				x = hess.At(n, n-1)
				s = math.Abs(x) + math.Abs(z)
				p = x / s
				q = z / s
				r = math.Hypot(p, q)
				p /= r
				q /= r

				// Row modification
				for j := n - 1; j < nn; j++ {
					z = hess.At(n-1, j)
					hess.Set(n-1, j, q*z+p*hess.At(n, j))
					hess.Set(n, j, q*hess.At(n, j)-p*z)
				}

				// Column modification
				for i := 0; i <= n; i++ {
					z = hess.At(i, n-1)
					hess.Set(i, n-1, q*z+p*hess.At(i, n))
					hess.Set(i, n, q*hess.At(i, n)-p*z)
				}

				// Accumulate transformations
				for i := low; i <= high; i++ {
					z = v.At(i, n-1)
					v.Set(i, n-1, q*z+p*v.At(i, n))
					v.Set(i, n, q*v.At(i, n)-p*z)
				}
			} else {
				// Complex pair
				d[n-1] = x + p
				d[n] = x + p
				e[n-1] = z
				e[n] = -z
			}
			n -= 2
			iter = 0
		} else {
			// No convergence yet

			// Form shift
			x = hess.At(n, n)
			y = 0
			w = 0
			if l < n {
				y = hess.At(n-1, n-1)
				w = hess.At(n, n-1) * hess.At(n-1, n)
			}

			// Wilkinson's original ad hoc shift
			if iter == 10 {
				exshift += x
				for i := low; i <= n; i++ {
					hess.Set(i, i, hess.At(i, i)-x)
				}
				s = math.Abs(hess.At(n, n-1)) + math.Abs(hess.At(n-1, n-2))
				x = 0.75 * s
				y = x
				w = -0.4375 * s * s
			}

			// MATLAB's new ad hoc shift
			if iter == 30 {
				s = (y - x) / 2
				s = s*s + w
				if s > 0 {
					s = math.Sqrt(s)
					if y < x {
						s = -s
					}
					s = x - w/((y-x)/2+s)
					for i := low; i <= n; i++ {
						hess.Set(i, i, hess.At(i, i)-s)
					}
					exshift += s
					x = 0.964
					y = x
					w = x
				}
			}

			iter++ // Could check iteration count here.

			// Look for two consecutive small sub-diagonal elements
			m := n - 2
			for m >= l {
				z = hess.At(m, m)
				r = x - z
				s = y - z
				p = (r*s-w)/hess.At(m+1, m) + hess.At(m, m+1)
				q = hess.At(m+1, m+1) - z - r - s
				r = hess.At(m+2, m+1)
				s = math.Abs(p) + math.Abs(q) + math.Abs(r)
				p /= s
				q /= s
				r /= s
				if m == l {
					break
				}
				if math.Abs(hess.At(m, m-1))*(math.Abs(q)+math.Abs(r)) <
					epsilon*(math.Abs(p)*(math.Abs(hess.At(m-1, m-1))+math.Abs(z)+math.Abs(hess.At(m+1, m+1)))) {
					break
				}
				m--
			}

			for i := m + 2; i <= n; i++ {
				hess.Set(i, i-2, 0)
				if i > m+2 {
					hess.Set(i, i-3, 0)
				}
			}

			// Double QR step involving rows l:n and columns m:n
			for k := m; k <= n-1; k++ {
				notlast := k != n-1
				if k != m {
					p = hess.At(k, k-1)
					q = hess.At(k+1, k-1)
					if notlast {
						r = hess.At(k+2, k-1)
					} else {
						r = 0
					}
					x = math.Abs(p) + math.Abs(q) + math.Abs(r)
					if x == 0 {
						continue
					}
					p /= x
					q /= x
					r /= x
				}

				s = math.Sqrt(p*p + q*q + r*r)
				if p < 0 {
					s = -s
				}
				if s != 0 {
					if k != m {
						hess.Set(k, k-1, -s*x)
					} else if l != m {
						hess.Set(k, k-1, -hess.At(k, k-1))
					}
					p += s
					x = p / s
					y = q / s
					z = r / s
					q /= p
					r /= p

					// Row modification
					for j := k; j < nn; j++ {
						p = hess.At(k, j) + q*hess.At(k+1, j)
						if notlast {
							p += r * hess.At(k+2, j)
							hess.Set(k+2, j, hess.At(k+2, j)-p*z)
						}
						hess.Set(k, j, hess.At(k, j)-p*x)
						hess.Set(k+1, j, hess.At(k+1, j)-p*y)
					}

					// Column modification
					for i := 0; i <= min(n, k+3); i++ {
						p = x*hess.At(i, k) + y*hess.At(i, k+1)
						if notlast {
							p += z * hess.At(i, k+2)
							hess.Set(i, k+2, hess.At(i, k+2)-p*r)
						}
						hess.Set(i, k, hess.At(i, k)-p)
						hess.Set(i, k+1, hess.At(i, k+1)-p*q)
					}

					// Accumulate transformations
					for i := low; i <= high; i++ {
						p = x*v.At(i, k) + y*v.At(i, k+1)
						if notlast {
							p += z * v.At(i, k+2)
							v.Set(i, k+2, v.At(i, k+2)-p*r)
						}
						v.Set(i, k, v.At(i, k)-p)
						v.Set(i, k+1, v.At(i, k+1)-p*q)
					}
				}
			}
		}
	}

	// Backsubstitute to find vectors of upper triangular form
	if norm == 0 {
		return
	}

	for n = nn - 1; n >= 0; n-- {
		p = d[n]
		q = e[n]

		if q == 0 {
			// Real vector
			l := n
			hess.Set(n, n, 1)
			for i := n - 1; i >= 0; i-- {
				w = hess.At(i, i) - p
				r = 0
				for j := l; j <= n; j++ {
					r += hess.At(i, j) * hess.At(j, n)
				}
				if e[i] < 0 {
					z = w
					s = r
				} else {
					l = i
					if e[i] == 0 {
						if w != 0 {
							hess.Set(i, n, -r/w)
						} else {
							hess.Set(i, n, -r/(epsilon*norm))
						}
					} else {
						// Solve real equations
						x = hess.At(i, i+1)
						y = hess.At(i+1, i)
						q = (d[i]-p)*(d[i]-p) + e[i]*e[i]
						t = (x*s - z*r) / q
						hess.Set(i, n, t)
						if math.Abs(x) > math.Abs(z) {
							hess.Set(i+1, n, (-r-w*t)/x)
						} else {
							hess.Set(i+1, n, (-s-y*t)/z)
						}
					}

					// Overflow control
					t = math.Abs(hess.At(i, n))
					if epsilon*t*t > 1 {
						for j := i; j <= n; j++ {
							hess.Set(j, n, hess.At(j, n)/t)
						}
					}
				}
			}
		} else if q < 0 {
			// Complex vector

			l := n - 1

			// Last vector component imaginary so matrix is triangular
			if math.Abs(hess.At(n, n-1)) > math.Abs(hess.At(n-1, n)) {
				hess.Set(n-1, n-1, q/hess.At(n, n-1))
				hess.Set(n-1, n, -(hess.At(n, n)-p)/hess.At(n, n-1))
			} else {
				re, im := cdiv(0, -hess.At(n-1, n), hess.At(n-1, n-1)-p, q)
				hess.Set(n-1, n-1, re)
				hess.Set(n-1, n, im)
			}
			hess.Set(n, n-1, 0)
			hess.Set(n, n, 1)

			for i := n - 2; i >= 0; i-- {
				var ra, sa, vr, vi float64
				for j := l; j <= n; j++ {
					ra += hess.At(i, j) * hess.At(j, n-1)
					sa += hess.At(i, j) * hess.At(j, n)
				}
				w = hess.At(i, i) - p

				if e[i] < 0 {
					z = w
					r = ra
					s = sa
				} else {
					l = i
					if e[i] == 0 {
						re, im := cdiv(-ra, -sa, w, q)
						hess.Set(i, n-1, re)
						hess.Set(i, n, im)
					} else {
						// Solve complex equations
						x = hess.At(i, i+1)
						y = hess.At(i+1, i)
						vr = (d[i]-p)*(d[i]-p) + e[i]*e[i] - q*q
						vi = (d[i] - p) * 2 * q
						if vr == 0 && vi == 0 {
							vr = epsilon * norm * (math.Abs(w) + math.Abs(q) + math.Abs(x) + math.Abs(y) + math.Abs(z))
						}
						re, im := cdiv(x*r-z*ra+q*sa, x*s-z*sa-q*ra, vr, vi)
						hess.Set(i, n-1, re)
						hess.Set(i, n, im)
						if math.Abs(x) > (math.Abs(z) + math.Abs(q)) {
							hess.Set(i+1, n-1, (-ra-w*hess.At(i, n-1)+q*hess.At(i, n))/x)
							hess.Set(i+1, n, (-sa-w*hess.At(i, n)-q*hess.At(i, n-1))/x)
						} else {
							re, im := cdiv(-r-y*hess.At(i, n-1), -s-y*hess.At(i, n), z, q)
							hess.Set(i+1, n-1, re)
							hess.Set(i+1, n, im)
						}
					}

					// Overflow control
					t = math.Max(math.Abs(hess.At(i, n-1)), math.Abs(hess.At(i, n)))
					if (epsilon*t)*t > 1 {
						for j := i; j <= n; j++ {
							hess.Set(j, n-1, hess.At(j, n-1)/t)
							hess.Set(j, n, hess.At(j, n)/t)
						}
					}
				}
			}
		}
	}

	// Vectors of isolated roots
	for i := 0; i < nn; i++ {
		if i < low || i > high {
			for j := i; j < nn; j++ {
				v.Set(i, j, hess.At(i, j))
			}
		}
	}

	// Back transformation to get eigenvectors of original matrix
	for j := nn - 1; j >= low; j-- {
		for i := low; i <= high; i++ {
			z = 0
			for k := low; k <= min(j, high); k++ {
				z += v.At(i, k) * hess.At(k, j)
			}
			v.Set(i, j, z)
		}
	}
}

// D returns the block diagonal eigenvalue matrix from the real and imaginary
// components d and e.
func (f EigenFactors) D() *Dense {
	d, e := f.d, f.e
	var n int
	if n = len(d); n != len(e) {
		panic(ErrSquare)
	}
	dm := NewDense(n, n, nil)
	for i := 0; i < n; i++ {
		dm.Set(i, i, d[i])
		if e[i] > 0 {
			dm.Set(i, i+1, e[i])
		} else if e[i] < 0 {
			dm.Set(i, i-1, e[i])
		}
	}
	return dm
}
