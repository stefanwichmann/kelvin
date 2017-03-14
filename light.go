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
	"github.com/dillonhafer/go.hue"
	"log"
	"strconv"
	"strings"
	"time"
)

var lightsSupportingDimming = []string{"Dimmable Light", "Color Temperature Light", "Color Light", "Extended Color Light"}
var lightsSupportingColorTemperature = []string{"Color Temperature Light", "Extended Color Light"}
var lightsSupportingXYColor = []string{"Color Light", "Extended Color Light"}

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
}

const lightUpdateIntervalInSeconds = 1
const lightTransitionIntervalInSeconds = 1

func (self *Light) updateCyclic(channel <-chan LightState) {
	log.Printf("💡 Starting cyclic update for %v\n", self.name)
	time.Sleep(2 * time.Second)
	for {
		select {
		case newLightState, ok := <-channel:
			if !ok {
				log.Printf("💡 Channel closed for light %v\n", self.name)
				return
			}
			self.targetLightState = newLightState
			self.update()
			self.clearChannel(channel)
		default:
			self.update()
			time.Sleep(lightUpdateIntervalInSeconds * time.Second)
		}
	}
}

func (self *Light) initialize() error {
	attr, err := self.hueLight.GetLightAttributes()
	if err != nil {
		return err
	}

	// initialize non changing values
	self.name = attr.Name
	self.dimmable = contains(lightsSupportingDimming, attr.Type)
	self.supportsColorTemperature = contains(lightsSupportingColorTemperature, attr.Type)
	self.supportsXYColor = contains(lightsSupportingXYColor, attr.Type)

	// initialize changing values
	self.on = attr.State.On
	self.reachable = attr.State.Reachable
	self.currentLightState = lightStateFromHueValues(attr.State.Ct, attr.State.Xy, attr.State.Bri)

	return nil
}

func (self *Light) updateCurrentLightState() error {
	attr, err := self.hueLight.GetLightAttributes()
	if err != nil {
		return err
	}
	self.reachable = attr.State.Reachable
	self.on = attr.State.On
	self.currentLightState = lightStateFromHueValues(attr.State.Ct, attr.State.Xy, attr.State.Bri)
	return nil
}

func (self *Light) update() error {
	// refresh current state
	self.updateCurrentLightState()

	// Light reachable or on?
	if !self.reachable || !self.on {
		if self.tracking {
			log.Printf("💡 Light %s is no longer reachable or turned on. Clearing state.\n", self.name)
			self.tracking = false
			self.manually = false
			return nil
		} else {
			// ignore light because we are not tracking it.
			return nil
		}
	}

	// did the light just appear?
	if !self.tracking {
		log.Printf("💡 Light %s just appeared. Initializing state to %vK at %v%s\n", self.name, self.targetLightState.colorTemperature, self.targetLightState.brightness, "%")

		// For initialization we set the state again and again for 10 seconds because during startup the zigbee communication is unstable
		for index := 0; index < 9; index++ {
			self.setLightState(self.targetLightState)
		}
		// safe the state of the last iteration
		setLightState, err := self.setLightState(self.targetLightState)
		if err != nil {
			return err
		}
		self.savedLightState = setLightState
		self.targetLightState = setLightState
		self.tracking = true
		return nil
	}

	// light in manual state
	if self.manually {
		return nil
	}

	// did the user manually change the light state?
	if !self.currentLightState.equals(self.savedLightState) {
		log.Printf("💡 Light %s was manually changed - current: %+v - saved: %+v\n", self.name, self.currentLightState, self.savedLightState)
		self.manually = true
		return nil
	}

	// Update needed?
	if self.currentLightState.equals(self.targetLightState) {
		return nil
	}

	// Light is reachable, on and in automatic state. Update to new color!
	log.Printf("💡 Updating light %s to %vK at %v%s\n", self.name, self.targetLightState.colorTemperature, self.targetLightState.brightness, "%")

	setLightState, err := self.setLightState(self.targetLightState)
	if err != nil {
		return err
	}

	// Debug: compare values
	if !setLightState.equals(self.targetLightState) {
		log.Printf("Target and Set state differ: %v, %v\n", self.targetLightState, setLightState)
	}

	self.savedLightState = setLightState
	self.targetLightState = setLightState
	return nil
}

func (self *Light) setLightState(state LightState) (LightState, error) {
	// Don't send repeated "On" as this slows the bridge down (see https://developers.meethue.com/faq-page #Performance)
	var newLightState hue.SetLightState
	colorTemperature, color, brightness := state.convertValuesToHue()
	if self.supportsXYColor {
		newLightState.Xy = []float32{color[0], color[1]}
		newLightState.Ct = strconv.Itoa(colorTemperature)
	} else if self.supportsColorTemperature {
		newLightState.Ct = strconv.Itoa(colorTemperature)
	}
	if self.dimmable {
		newLightState.Bri = strconv.Itoa(brightness)
	}
	newLightState.TransitionTime = strconv.Itoa(lightTransitionIntervalInSeconds * 10) // conversion to 100ms-value

	results, err := self.hueLight.SetState(newLightState)
	if err != nil {
		return LightState{0, []float32{0, 0}, 0}, err
	}

	// iterate over result to aquire set values
	for _, result := range results {
		for key, value := range result.Success {
			path := strings.Split(key, "/")

			switch path[len(path)-1] {
			case "xy":
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

func (self *Light) clearChannel(c <-chan LightState) {
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
