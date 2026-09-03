// MIT License
//
// # Copyright (c) 2019 Stefan Wichmann
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.
package main

import (
	"errors"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
	hue "github.com/stefanwichmann/go.hue"
)

var lightsSupportingDimming = []string{"Dimmable light", "Color temperature light", "Color light", "Extended color light"}
var lightsSupportingColorTemperature = []string{"Color temperature light", "Extended color light"}
var lightsSupportingXYColor = []string{"Color light", "Extended color light"}

// HueLight represents a physical hue light.
type HueLight struct {
	Name                     string
	HueLight                 hue.Light
	SetColorTemperature      int
	SetBrightness            int
	TargetColorTemperature   int
	TargetColor              []float32
	TargetBrightness         int
	CurrentColorTemperature  int
	CurrentColor             []float32
	CurrentBrightness        int
	CurrentColorMode         string
	SupportsColorTemperature bool
	SupportsXYColor          bool
	Dimmable                 bool
	Reachable                bool
	On                       bool
	MinimumColorTemperature  int
	Gamut                    [][]float32
	MinimumMirek             int
	MaximumMirek             int
}

func (light *HueLight) initialize(attr hue.LightAttributes) {
	// initialize non changing values
	light.Name = attr.Name
	light.Dimmable = containsString(lightsSupportingDimming, attr.Type)
	light.SupportsColorTemperature = containsString(lightsSupportingColorTemperature, attr.Type)
	light.SupportsXYColor = containsString(lightsSupportingXYColor, attr.Type)

	// set minimum color temperature depending on type
	if attr.Type == "Color temperature light" {
		light.MinimumColorTemperature = 2200
	} else if light.SupportsXYColor {
		light.MinimumColorTemperature = 1000
	} else if light.SupportsColorTemperature {
		light.MinimumColorTemperature = 2000
	} else {
		light.MinimumColorTemperature = 0
	}

	// A light's reported capabilities pin the exact values the bridge
	// clamps to, so targets can be computed as the bridge will report
	// them back (issue #129).
	control := attr.Capabilities.Control
	if len(control.ColorGamut) == 3 {
		light.Gamut = control.ColorGamut
	}
	if control.ColorTemperature.Min > 0 {
		light.MinimumMirek = control.ColorTemperature.Min
	}
	if control.ColorTemperature.Max > 0 {
		light.MaximumMirek = control.ColorTemperature.Max
		if !light.SupportsXYColor {
			// The warmest kelvin value this light reaches; xy-capable
			// lights keep the lower bound because warmer targets are
			// reachable through the gamut-clamped xy path.
			light.MinimumColorTemperature = (1000000 + control.ColorTemperature.Max - 1) / control.ColorTemperature.Max
		}
	}

	log.Debugf("💡 Light %s - Initialization complete. Identified as %s (ModelID: %s, Version: %s)", light.Name, attr.Type, attr.ModelId, attr.SoftwareVersion)

	light.updateCurrentLightState(attr)
}

func (light *HueLight) supportsColorTemperature() bool {
	if light.SupportsXYColor || light.SupportsColorTemperature {
		return true
	}
	return false
}

func (light *HueLight) supportsBrightness() bool {
	return light.Dimmable
}

func (light *HueLight) updateCurrentLightState(attr hue.LightAttributes) {
	light.CurrentColorTemperature = attr.State.Ct

	var color []float32
	for _, value := range attr.State.Xy {
		color = append(color, roundFloat(value, 3))
	}
	light.CurrentColor = color
	light.CurrentBrightness = attr.State.Bri
	light.CurrentColorMode = attr.State.ColorMode

	if !attr.State.Reachable {
		light.Reachable = false
		light.On = false
	} else {
		light.Reachable = true
		light.On = attr.State.On
	}
}

func (light *HueLight) setLightState(colorTemperature int, brightness int, transitionTime time.Duration) error {
	if colorTemperature != -1 && (colorTemperature < 1000 || colorTemperature > 6500) {
		log.Warningf("💡 Light %s - Invalid color temperature %d", light.Name, colorTemperature)
	}
	if brightness < -1 || brightness > 100 {
		log.Warningf("💡 Light %s - Invalid brightness %d", light.Name, brightness)
	}

	if colorTemperature != -1 && colorTemperature < light.MinimumColorTemperature {
		colorTemperature = light.MinimumColorTemperature
		log.Debugf("💡 Light %s - Adjusted color temperature to light capability of %dK", light.Name, colorTemperature)
	}

	light.SetColorTemperature = colorTemperature
	light.SetBrightness = brightness

	// Map parameters to the values the bridge will apply and report back.
	light.TargetColorTemperature = light.targetMirek(colorTemperature)
	light.TargetColor = light.targetColor(colorTemperature)
	light.TargetBrightness = mapBrightness(brightness)

	// Send new state to light bulb
	var hueLightState hue.SetLightState
	hueLightState.TransitionTime = strconv.Itoa(int(transitionTime / time.Millisecond / 100))

	if colorTemperature != -1 {
		// Set supported colormodes. If both are, the brigde will prefer xy colors
		if light.SupportsXYColor {
			hueLightState.Xy = light.TargetColor
		}
		if light.SupportsColorTemperature {
			hueLightState.Ct = strconv.Itoa(light.TargetColorTemperature)
		}
	}

	if brightness != -1 {
		if brightness == 0 {
			// Target brightness zero should turn the light off.
			hueLightState.On = "Off"
		} else if light.Dimmable {
			hueLightState.Bri = strconv.Itoa(light.TargetBrightness)
		}
	}

	// Send new state to the light
	log.Debugf("💡 HueLight %s - Setting light state to %dK and %d%% brightness (TargetColorTemperature: %d, CurrentColorTemperature: %d, TargetColor: %v, CurrentColor: %v, TargetBrightness: %d, CurrentBrightness: %d, TransitionTime: %s)", light.Name, colorTemperature, brightness, light.TargetColorTemperature, light.CurrentColorTemperature, light.TargetColor, light.CurrentColor, light.TargetBrightness, light.CurrentBrightness, hueLightState.TransitionTime)
	result, err := light.HueLight.SetState(hueLightState)
	if err != nil {
		log.Warningf("💡 HueLight %s - Setting light state failed: %v (Result: %v)", light.Name, err, result)
		return err
	}

	log.Debugf("💡 HueLight %s - Light was successfully updated (TargetColorTemperature: %d, CurrentColorTemperature: %d, TargetColor: %v, CurrentColor: %v, TargetBrightness: %d, CurrentBrightness: %d, TransitionTime: %s)", light.Name, light.TargetColorTemperature, light.CurrentColorTemperature, light.TargetColor, light.CurrentColor, light.TargetBrightness, light.CurrentBrightness, hueLightState.TransitionTime)
	return nil
}

func (light *HueLight) hasChanged() bool {
	if light.SupportsXYColor && light.CurrentColorMode == "xy" {
		if !equalsFloat(light.TargetColor, []float32{-1, -1}, 0) && !equalsFloat(light.TargetColor, light.CurrentColor, 0.001) {
			log.Debugf("💡 HueLight %s - Color has changed! CurrentColor: %v, TargetColor: %v (%dK)", light.Name, light.CurrentColor, light.TargetColor, light.SetColorTemperature)
			return true
		}
	} else if light.SupportsColorTemperature && light.CurrentColorMode == "ct" {
		if light.TargetColorTemperature != -1 && !equalsInt(light.TargetColorTemperature, light.CurrentColorTemperature, 2) {
			log.Debugf("💡 HueLight %s - Color temperature has changed! CurrentColorTemperature: %d, TargetColorTemperatur: %d (%dK)", light.Name, light.CurrentColorTemperature, light.TargetColorTemperature, light.SetColorTemperature)
			return true
		}
	}

	if light.Dimmable && light.TargetBrightness != -1 && !equalsInt(light.TargetBrightness, light.CurrentBrightness, 2) {
		log.Debugf("💡 HueLight %s - Brightness has changed! CurrentBrightness: %d, TargetBrightness: %d (%d%%)", light.Name, light.CurrentBrightness, light.TargetBrightness, light.SetBrightness)
		return true
	}

	return false
}

func (light *HueLight) hasState(colorTemperature int, brightness int) bool {
	return light.hasColorTemperature(colorTemperature) && light.hasBrightness(brightness)
}

func (light *HueLight) hasColorTemperature(colorTemperature int) bool {
	if colorTemperature == -1 || light.TargetColorTemperature == -1 {
		return true
	}
	if !light.SupportsXYColor && !light.SupportsColorTemperature {
		return true
	}

	if colorTemperature < light.MinimumColorTemperature {
		colorTemperature = light.MinimumColorTemperature
		log.Debugf("💡 Light %s - Adjusted color temperature to light capability of %dK", light.Name, colorTemperature)
	}

	if light.SupportsXYColor && light.CurrentColorMode == "xy" {
		return equalsFloat(light.targetColor(colorTemperature), light.CurrentColor, 0.001)
	} else if light.SupportsColorTemperature && light.CurrentColorMode == "ct" {
		return equalsInt(light.CurrentColorTemperature, light.targetMirek(colorTemperature), 2)
	}

	// Missmatch in color modes? Log warning for debug purposes and assume unchanged
	log.Warningf("💡 HueLight %s - Unknown color mode in HasColorTemperature method! Current light state: %+v", light.Name, light)

	return true
}

func (light *HueLight) hasBrightness(brightness int) bool {
	if brightness == -1 || light.TargetBrightness == -1 {
		return true
	}
	if !light.Dimmable {
		return true
	}
	if !equalsInt(light.CurrentBrightness, mapBrightness(brightness), 2) {
		return false
	}
	return true
}

func (light *HueLight) getCurrentColorTemperature() (int, error) {
	if !light.hasChanged() {
		return light.SetColorTemperature, nil
	}

	if light.CurrentColorTemperature >= 153 && light.CurrentColorTemperature <= 500 {
		return int(float64(1000000) / float64(light.CurrentColorTemperature)), nil
	}

	return 0, errors.New("could not determine current color temperature")
}

func (light *HueLight) getCurrentBrightness() (int, error) {
	if !light.hasChanged() {
		return light.SetBrightness, nil
	}

	if light.CurrentBrightness >= 1 && light.CurrentBrightness <= 254 {
		return int((float64(light.CurrentBrightness) / float64(254)) * float64(100)), nil
	}

	return 0, errors.New("could not determine current brightness")
}

// targetColor maps a kelvin value to the xy color the bridge will apply:
// converted, then clamped into the light's reported gamut.
func (light *HueLight) targetColor(colorTemperature int) []float32 {
	return clampToGamut(colorTemperatureToXYColor(colorTemperature), light.Gamut)
}

// targetMirek maps a kelvin value to the mirek value the bridge will
// apply: bounded by the Hue-wide range of 153 to 500 and tightened by the
// light's reported capability range.
func (light *HueLight) targetMirek(colorTemperature int) int {
	mirek := mapColorTemperature(colorTemperature)
	if mirek == -1 {
		return -1
	}
	minimum, maximum := 153, 500
	if light.MinimumMirek > 0 {
		minimum = light.MinimumMirek
	}
	if light.MaximumMirek > 0 {
		maximum = light.MaximumMirek
	}
	if mirek < minimum {
		return minimum
	}
	if mirek > maximum {
		return maximum
	}
	return mirek
}

func mapColorTemperature(colorTemperature int) int {
	if colorTemperature == -1 {
		return -1
	}

	if colorTemperature > 6500 {
		colorTemperature = 6500
	} else if colorTemperature < 1000 {
		colorTemperature = 1000
	}
	return int((float64(1) / float64(colorTemperature)) * float64(1000000))
}

func mapBrightness(brightness int) int {
	if brightness == -1 {
		return -1
	}

	if brightness > 100 {
		brightness = 100
	} else if brightness < 0 {
		brightness = 0
	}
	return int((float64(brightness) / float64(100)) * float64(254))
}
