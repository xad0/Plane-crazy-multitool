package main

import "math"

// Calculate returns the full angle, half angle and raw offset.
func Calculate(teeth int) (full, half, offset float64) {
	if teeth < 3 {
		return 0, 0, 0
	}

	full = 360.0 / float64(teeth)
	half = full / 2.0

	a := full * math.Pi / 180.0
	b := half * math.Pi / 180.0

	r := math.Cos(b) / math.Sin(a)
	offset = r - 1.0

	return
}

// AutoCompressors returns the minimum number of compressors
// needed for the final compressor value to be <= 1.
//
// At least one compressor is always returned.
func AutoCompressors(offset float64) int {
	n := int(math.Ceil(offset))

	if n < 1 {
		return 1
	}

	return n
}

// CompressorValue returns the required value for the final compressor.
func CompressorValue(offset float64, compressors int) float64 {
	return offset - float64(compressors-1)
}

// Round3 rounds to three decimal places.
func Round3(value float64) float64 {
	return math.Round(value*1000) / 1000
}
