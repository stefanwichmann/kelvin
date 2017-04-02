// MIT License
//
// Copyright (c) 2017 Stefan Wichmann
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
	"github.com/stefanwichmann/go.hue"
	"log"
	"strconv"
	"strings"
	"time"
)

var lightsSupportingDimming = []string{"Dimmable Light", "Color Temperature Light", "Color Light", "Extended Color Light"}
var lightsSupportingColorTemperature = []string{"Color Temperature Light", "Extended Color Light"}
var lightsSupportingXYColor = []string{"Color Light", "Extended Color Light"}

// Light represents a single physical hue light in your system.
// It is used to read and write it's state.
type Light struct {
	id                       int
	name                     string
	hueLight                 hue.Light
	savedLightState          LightState
	currentLightState        LightState
	targetLightState         LightState
	on                       bool
	reachable                bool
	tracking                 bool
	manually                 bool
	dimmable                 bool
	supportsColorTemperature bool
	supportsXYColor          bool
	ignored                  bool
}

const lightUpdateIntervalInSeconds = 1
const lightTransitionIntervalInSeconds = 1

func (light *Light) updateCyclic(channel <-chan LightState) {
	log.Printf("💡 Starting cyclic update for %v\n", light.name)
	time.Sleep(2 * time.Second)
	for {
		select {
		case newLightState, ok := <-channel:
			if !ok {
				log.Printf("💡 Channel closed for light %v\n", light.name)
				return
			}
			light.targetLightState = newLightState
			light.update()
			light.clearChannel(channel)
		default:
			light.update()
			time.Sleep(lightUpdateIntervalInSeconds * time.Second)
		}
	}
}

func (light *Light) initialize() error {
	attr, err := light.hueLight.GetLightAttributes()
	if err != nil {
		return err
	}

	// initialize non changing values
	light.name = attr.Name
	light.dimmable = containsString(lightsSupportingDimming, attr.Type)
	light.supportsColorTemperature = containsString(lightsSupportingColorTemperature, attr.Type)
	light.supportsXYColor = containsString(lightsSupportingXYColor, attr.Type)
	light.ignored = false

	// initialize changing values
	light.on = attr.State.On
	light.reachable = attr.State.Reachable
	light.currentLightState = lightStateFromHueValues(attr.State.Ct, attr.State.Xy, attr.State.Bri)

	return nil
}

func (light *Light) updateCurrentLightState() error {
	attr, err := light.hueLight.GetLightAttributes()
	if err != nil {
		return err
	}
	light.reachable = attr.State.Reachable
	light.on = attr.State.On
	light.currentLightState = lightStateFromHueValues(attr.State.Ct, attr.State.Xy, attr.State.Bri)

	return nil
}

func (light *Light) update() error {
	// refresh current state
	light.updateCurrentLightState()

	// Light reachable or on?
	if !light.reachable || !light.on {
		if light.tracking {
			log.Printf("💡 Light %s is no longer reachable or turned on. Clearing state.\n", light.name)
			light.tracking = false
			light.manually = false
			return nil
		}

		// ignore light because we are not tracking it.
		return nil
	}

	// did the light just appear?
	if !light.tracking {
		log.Printf("💡 Light %s just appeared. Initializing state to %vK at %v%s\n", light.name, light.targetLightState.colorTemperature, light.targetLightState.brightness, "%")

		// For initialization we set the state again and again for 10 seconds
		// because during startup the zigbee communication is unstable
		for index := 0; index < 9; index++ {
			light.setLightState(light.targetLightState)
		}
		// safe the state of the last iteration
		setLightState, err := light.setLightState(light.targetLightState)
		if err != nil {
			return err
		}
		light.savedLightState = setLightState
		light.targetLightState = setLightState
		light.tracking = true
		return nil
	}

	// light in manual state
	if light.manually {
		return nil
	}

	// did the user manually change the light state?
	if !light.savedLightState.equals(light.currentLightState) {
		log.Printf("💡 Light %s was manually changed - current: %+v - saved: %+v\n", light.name, light.currentLightState, light.savedLightState)
		light.manually = true
		return nil
	}

	// Update needed?
	if light.targetLightState.equals(light.currentLightState) {
		return nil
	}

	// Light is reachable, on and in automatic state. Update to new color!
	log.Printf("💡 Updating light %s to %vK at %v%s\n", light.name, light.targetLightState.colorTemperature, light.targetLightState.brightness, "%")

	setLightState, err := light.setLightState(light.targetLightState)
	if err != nil {
		return err
	}

	// Debug: compare values
	if !setLightState.equals(light.targetLightState) {
		log.Printf("Target and Set state differ: %v, %v\n", light.targetLightState, setLightState)
	}

	light.savedLightState = setLightState
	light.targetLightState = setLightState
	return nil
}

func (light *Light) setLightState(state LightState) (LightState, error) {
	// Don't send repeated "On" as this slows the bridge down (see https://developers.meethue.com/faq-page #Performance)
	var hueLightState hue.SetLightState
	colorTemperature, color, brightness := state.convertValuesToHue()
	if light.supportsXYColor && state.color[0] != 0 && state.color[1] != 0 {
		hueLightState.Xy = []float32{color[0], color[1]}
	} else if light.supportsColorTemperature && state.colorTemperature != 0 {
		hueLightState.Ct = strconv.Itoa(colorTemperature)
	}
	if light.dimmable && state.brightness != 0 {
		hueLightState.Bri = strconv.Itoa(brightness)
	}
	hueLightState.TransitionTime = strconv.Itoa(lightTransitionIntervalInSeconds * 10) // conversion to 100ms-value

	results, err := light.hueLight.SetState(hueLightState)
	if err != nil {
		return LightState{0, []float32{0, 0}, 0}, err
	}

	// iterate over result to acquire set values
	for _, result := range results {
		for key, value := range result.Success {
			path := strings.Split(key, "/")

			switch path[len(path)-1] {
			case "xy":
				color = []float32{} // clear old color values
				for _, elem := range value.([]interface{}) {
					color = append(color, float32(elem.(float64)))
				}
			case "bri":
				brightness = int(value.(float64))
			case "ct":
				colorTemperature = int(value.(float64))
			}
		}
	}
	setLightState := lightStateFromHueValues(colorTemperature, color, brightness)

	// wait while the light is in transition before returning
	time.Sleep(lightTransitionIntervalInSeconds + 1*time.Second)
	return setLightState, nil
}

func (light *Light) clearChannel(c <-chan LightState) {
	for {
		select {
		case state, ok := <-c:
			if !ok {
				return
			}
			log.Printf("Cleared light state from channel: %+v\n", state)
		default:
			return
		}
	}
}
