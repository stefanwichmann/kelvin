package main

import (
	"testing"
)

// TestHasChangedAcceptsClampedTargets pins the root cause of issue #126:
// a warm target outside the light's gamut is clamped by the bridge, and
// comparing the readback against the unclamped target misreads the clamp
// as a manual user change. Targets must be clamped before comparison.
func TestHasChangedAcceptsClampedTargets(t *testing.T) {
	light := HueLight{Name: "test", SupportsXYColor: true, CurrentColorMode: "xy", Gamut: gamutB}
	light.TargetColor = light.targetColor(1000)
	light.TargetColorTemperature = light.targetMirek(1000)
	light.TargetBrightness = -1
	light.CurrentColor = light.TargetColor // the bridge reports the clamped value

	if light.hasChanged() {
		t.Error("clamped target misread as manual change")
	}

	light.TargetColor = colorTemperatureToXYColor(1000)
	if !light.hasChanged() {
		t.Error("the unclamped 1000K target must mismatch the clamped readback — the regression this fix removes")
	}
}

// TestHasColorTemperatureAcceptsGamutClampedReadback covers the scene
// activation path of issue #122: hasState compared a recomputed unclamped
// color against the bridge's clamped state and never matched.
func TestHasColorTemperatureAcceptsGamutClampedReadback(t *testing.T) {
	light := HueLight{Name: "test", SupportsXYColor: true, CurrentColorMode: "xy", MinimumColorTemperature: 1000, Gamut: gamutB, TargetColorTemperature: 500}
	light.CurrentColor = clampToGamut(colorTemperatureToXYColor(1000), gamutB)

	if !light.hasColorTemperature(1000) {
		t.Error("clamped readback misread as a different color temperature")
	}
}

// TestTargetMirekRespectsReportedRange pins the color-temperature-mode
// variant: the bridge clamps mirek values into the light's reported ct
// range, so the target must be clamped the same way.
func TestTargetMirekRespectsReportedRange(t *testing.T) {
	light := HueLight{Name: "test", MinimumMirek: 153, MaximumMirek: 454}
	if mirek := light.targetMirek(2000); mirek != 454 {
		t.Errorf("2000K on a 454-mirek light: got %d, want 454", mirek)
	}
	if mirek := light.targetMirek(-1); mirek != -1 {
		t.Errorf("-1 sentinel must pass unchanged, got %d", mirek)
	}

	// without reported capabilities the Hue-wide range 153-500 applies
	uncapped := HueLight{Name: "test"}
	if mirek := uncapped.targetMirek(1000); mirek != 500 {
		t.Errorf("1000K without capabilities: got %d, want 500", mirek)
	}
	if mirek := uncapped.targetMirek(6500); mirek != 153 {
		t.Errorf("6500K without capabilities: got %d, want 153", mirek)
	}
}
