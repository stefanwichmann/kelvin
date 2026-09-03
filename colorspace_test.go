package main

import (
	"testing"
)

// Hue gamut B (older color bulbs) and gamut C (current ones), as lights
// report them in capabilities.control.colorgamut.
var gamutB = [][]float32{{0.675, 0.322}, {0.409, 0.518}, {0.167, 0.04}}
var gamutC = [][]float32{{0.6915, 0.3083}, {0.17, 0.7}, {0.1532, 0.0475}}

func insideGamut(xy []float32, gamut [][]float32) bool {
	clamped := clampToGamut(xy, gamut)
	return equalsFloat(clamped, xy, 0.0011)
}

func TestClampToGamutMovesWarmWhitesInside(t *testing.T) {
	for _, gamut := range [][][]float32{gamutB, gamutC} {
		warm := colorTemperatureToXYColor(1000)
		clamped := clampToGamut(warm, gamut)
		if equalsFloat(clamped, warm, 0) {
			t.Errorf("1000K lies outside gamut %v and must move: %v", gamut, clamped)
		}
		if !insideGamut(clamped, gamut) {
			t.Errorf("clamped point %v still outside gamut %v", clamped, gamut)
		}
	}
}

func TestClampToGamutKeepsReachableColors(t *testing.T) {
	neutral := colorTemperatureToXYColor(2700)
	if clamped := clampToGamut(neutral, gamutC); !equalsFloat(clamped, neutral, 0) {
		t.Errorf("in-gamut color must pass unchanged: %v became %v", neutral, clamped)
	}
}

func TestClampToGamutPassesSentinelAndUnknownGamut(t *testing.T) {
	sentinel := []float32{-1, -1}
	if clamped := clampToGamut(sentinel, gamutC); !equalsFloat(clamped, sentinel, 0) {
		t.Errorf("-1 sentinel must pass unchanged, got %v", clamped)
	}
	warm := colorTemperatureToXYColor(1000)
	if clamped := clampToGamut(warm, nil); !equalsFloat(clamped, warm, 0) {
		t.Errorf("unknown gamut must pass values unchanged, got %v", clamped)
	}
}
