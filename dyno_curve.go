package main

import "math"

func dynoPCHIPSlopes(
	points []dynoGraphPoint,
) []float64 {

	n := len(points)

	if n < 2 {
		return nil
	}

	h := make([]float64, n-1)
	delta := make([]float64, n-1)

	for i := 0; i < n-1; i++ {
		h[i] = points[i+1].RPM - points[i].RPM

		if h[i] <= 0 {
			return nil
		}

		delta[i] =
			(points[i+1].BHP -
				points[i].BHP) /
				h[i]
	}

	slopes := make([]float64, n)

	if n == 2 {
		slopes[0] = delta[0]
		slopes[1] = delta[0]
		return slopes
	}

	endpoint := func(
		h0 float64,
		h1 float64,
		d0 float64,
		d1 float64,
	) float64 {

		d :=
			((2*h0+h1)*d0 -
				h0*d1) /
				(h0 + h1)

		if d == 0 ||
			d0 == 0 ||
			d*d0 < 0 {

			return 0
		}

		if d0*d1 < 0 &&
			math.Abs(d) > math.Abs(3*d0) {

			return 3 * d0
		}

		return d
	}

	slopes[0] =
		endpoint(
			h[0],
			h[1],
			delta[0],
			delta[1],
		)

	for i := 1; i < n-1; i++ {

		d0 := delta[i-1]
		d1 := delta[i]

		if d0 == 0 ||
			d1 == 0 ||
			d0*d1 <= 0 {

			slopes[i] = 0
			continue
		}

		w1 :=
			2*h[i] +
				h[i-1]

		w2 :=
			h[i] +
				2*h[i-1]

		slopes[i] =
			(w1 + w2) /
				(w1/d0 +
					w2/d1)
	}

	slopes[n-1] =
		endpoint(
			h[n-2],
			h[n-3],
			delta[n-2],
			delta[n-3],
		)

	return slopes
}

func dynoPCHIPValue(
	p1 dynoGraphPoint,
	p2 dynoGraphPoint,
	d1 float64,
	d2 float64,
	t float64,
) float64 {

	h :=
		p2.RPM -
			p1.RPM

	if h <= 0 {
		return p1.BHP
	}

	t2 := t * t
	t3 := t2 * t

	h00 := 2*t3 - 3*t2 + 1
	h10 := t3 - 2*t2 + t
	h01 := -2*t3 + 3*t2
	h11 := t3 - t2

	value :=
		h00*p1.BHP +
			h10*h*d1 +
			h01*p2.BHP +
			h11*h*d2

	minValue :=
		math.Min(
			p1.BHP,
			p2.BHP,
		)

	maxValue :=
		math.Max(
			p1.BHP,
			p2.BHP,
		)

	if value < minValue {
		return minValue
	}

	if value > maxValue {
		return maxValue
	}

	return value
}
