// MIT License
//
// Copyright (c) 2018 Stefan Wichmann
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

import log "github.com/Sirupsen/logrus"
import "github.com/stefanwichmann/go.hue"
import "strconv"
import "time"
import "errors"

var lightsSupportingDimming = []string{"Dimmable Light", "Color Temperature Light", "Color Light", "Extended Color Light"}
var lightsSupportingColorTemperature = []string{"Color Temperature Light", "Extended Color Light"}
var lightsSupportingXYColor = []string{"Color Light", "Extended Color Light"}

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
	On                       bool
	MinimumColorTemperature  int
}

const lightTransitionIntervalInSeconds = 1

func (light *HueLight) initialize() error {
	attr, err := light.HueLight.GetLightAttributes()
	if err != nil {
		return err
	}

	// initialize non changing values
	light.Name = attr.Name
	light.Dimmable = containsString(lightsSupportingDimming, attr.Type)
	light.SupportsColorTemperature = containsString(lightsSupportingColorTemperature, attr.Type)
	light.SupportsXYColor = containsString(lightsSupportingXYColor, attr.Type)

	// set minimum color temperature depending on type
	if attr.Type == "Color Temperature Light" {
		light.MinimumColorTemperature = 2200
	} else if light.SupportsXYColor || light.SupportsColorTemperature {
		light.MinimumColorTemperature = 2000
	} else {
		light.MinimumColorTemperature = 0
	}

	log.Debugf("💡 Light %s - Initialization complete. Identified as %s (ModelID: %s, Version: %s)", light.Name, attr.Type, attr.ModelId, attr.SoftwareVersion)

	light.updateCurrentLightState()
	return nil
}

func (light *HueLight) supportsColorTemperature() bool {
	if light.SupportsXYColor || light.SupportsColorTemperature {
		return true
	}
	return false
}

func (light *HueLight) supportsBrightness() bool {
	if light.Dimmable {
		return true
	}
	return false
}

func (light *HueLight) updateCurrentLightState() error {
	attr, err := light.HueLight.GetLightAttributes()
	if err != nil {
		return err
	}

	light.CurrentColorTemperature = attr.State.Ct

	var color []float32
	for _, value := range attr.State.Xy {
		color = append(color, roundFloat(value, 3))
	}
	light.CurrentColor = color
	light.CurrentBrightness = attr.State.Bri
	light.CurrentColorMode = attr.State.ColorMode

	if !attr.State.Reachable {
		light.On = false
	} else {
		light.On = attr.State.On
	}

	return nil
}

func (light *HueLight) setLightState(colorTemperature int, brightness int) error {
	if colorTemperature != -1 && (colorTemperature < 2000 || colorTemperature > 6500) {
		log.Warningf("💡 Light %s - Invalid color temperature %d", light.Name, colorTemperature)
	}
	if brightness < -1 || brightness > 100 {
		log.Warningf("💡 Light %s - Invalid brightness %d", light.Name, brightness)
	}

	if colorTemperature < light.MinimumColorTemperature {
		colorTemperature = light.MinimumColorTemperature
		log.Debugf("💡 Light %s - Adjusted color temperature to light capability of %dK", light.Name, colorTemperature)
	}

	log.Debugf("💡 HueLight %s - Setting light state to %dK and %d%% brightness.", light.Name, colorTemperature, brightness)
	light.SetColorTemperature = colorTemperature
	light.SetBrightness = brightness

	// map parameters to target values
	light.TargetColorTemperature = mapColorTemperature(colorTemperature)
	light.TargetColor = colorTemperatureToXYColor(colorTemperature)
	light.TargetBrightness = mapBrightness(brightness)

	// Send new state to light bulb
	var hueLightState hue.SetLightState

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
	hueLightState.TransitionTime = strconv.Itoa(lightTransitionIntervalInSeconds * 10) // Conversion to 100ms-value

	// Send new state to the light
	_, err := light.HueLight.SetState(hueLightState)
	if err != nil {
		return err
	}

	// Wait while the light is in transition before returning
	time.Sleep(lightTransitionIntervalInSeconds + 1*time.Second)

	// Debug: Update current state to double check
	if log.GetLevel() == log.DebugLevel {
		light.updateCurrentLightState()
		if light.hasChanged() {
			log.Warningf("💡 HueLight %s - Failed to update light state: %+v", light.Name, light)
		} else {
			log.Debugf("💡 HueLight %s - Light was successfully updated.", light.Name)
		}
	}

	return nil
}

func (light *HueLight) hasChanged() bool {
	if light.SupportsXYColor && light.CurrentColorMode == "xy" {
		if !equalsFloat(light.TargetColor, []float32{-1, -1}, 0) && !equalsFloat(light.TargetColor, light.CurrentColor, 0.001) {
			log.Debugf("💡 HueLight %s - Color has changed! Current light state: %+v", light.Name, light)
			return true
		}
	} else if light.SupportsColorTemperature && light.CurrentColorMode == "ct" {
		if light.TargetColorTemperature != -1 && !equalsInt(light.TargetColorTemperature, light.CurrentColorTemperature, 2) {
			log.Debugf("💡 HueLight %s - Color temperature has changed! Current light state: %+v", light.Name, light)
			return true
		}
	}

	if light.Dimmable && light.TargetBrightness != -1 && !equalsInt(light.TargetBrightness, light.CurrentBrightness, 2) {
		log.Debugf("💡 HueLight %s - Brightness has changed! Current light state: %+v", light.Name, light)
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
		if equalsFloat(colorTemperatureToXYColor(colorTemperature), light.CurrentColor, 0.001) {
			return true
		}
		return false
	} else if light.SupportsColorTemperature && light.CurrentColorMode == "ct" {
		if equalsInt(light.CurrentColorTemperature, mapColorTemperature(colorTemperature), 2) {
			return true
		}
		return false
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

	return 0, errors.New("Could not determine current color temperature")
}

func (light *HueLight) getCurrentBrightness() (int, error) {
	if !light.hasChanged() {
		return light.SetBrightness, nil
	}

	if light.CurrentBrightness >= 1 && light.CurrentBrightness <= 254 {
		return int((float64(light.CurrentBrightness) / float64(254)) * float64(100)), nil
	}

	return 0, errors.New("Could not determine current brightness")
}

func mapColorTemperature(colorTemperature int) int {
	if colorTemperature == -1 {
		return -1
	}

	if colorTemperature > 6500 {
		colorTemperature = 6500
	} else if colorTemperature < 2000 {
		colorTemperature = 2000
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
