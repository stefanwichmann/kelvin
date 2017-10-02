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
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/stefanwichmann/go.hue"
)

var lightsSupportingDimming = []string{"Dimmable Light", "Color Temperature Light", "Color Light", "Extended Color Light"}
var lightsSupportingColorTemperature = []string{"Color Temperature Light", "Extended Color Light"}
var lightsSupportingXYColor = []string{"Color Light", "Extended Color Light"}

// Light represents a single physical hue light in your system.
// It is used to read and write it's state.
type Light struct {
	ID                       int
	Name                     string
	HueLight                 hue.Light
	SavedLightState          LightState
	CurrentLightState        LightState
	TargetLightState         LightState
	Scheduled                bool
	On                       bool
	Reachable                bool
	Tracking                 bool
	Automatic                bool
	Dimmable                 bool
	SupportsColorTemperature bool
	SupportsXYColor          bool
	Configuration            *Configuration
	Schedule                 Schedule
	Interval                 Interval
}

const lightUpdateIntervalInSeconds = 1
const lightTransitionIntervalInSeconds = 1
const stateUpdateIntervalInSeconds = 60

func (light *Light) updateCyclic(configuration *Configuration) {
	light.Configuration = configuration

	// Filter devices we can't control
	if !light.Dimmable && !light.SupportsXYColor && !light.SupportsColorTemperature {
		log.Printf("💡 Device %v doesn't support any functionality Kelvin uses. Ignoring...", light.Name)
		return
	}

	light.updateSchedule()
	light.updateInterval()
	light.updateTargetLightState()

	// Start cyclic polling
	log.Debugf("💡 Light %s: Starting cyclic update...", light.Name)
	lightUpdateTick := time.Tick(lightUpdateIntervalInSeconds * time.Second)
	stateUpdateTick := time.Tick(stateUpdateIntervalInSeconds * time.Second)
	for {
		select {
		case <-time.After(time.Until(light.Schedule.endOfDay) + 1*time.Second):
			// Day has ended, calculate new schedule
			light.updateSchedule()
		case <-stateUpdateTick:
			// update interval and color every minute
			light.updateInterval()
			light.updateTargetLightState()
		case <-lightUpdateTick:
			light.update()
		}
	}
}

func (light *Light) initialize() error {
	attr, err := light.HueLight.GetLightAttributes()
	if err != nil {
		return err
	}

	// initialize non changing values
	light.Name = attr.Name
	light.Dimmable = containsString(lightsSupportingDimming, attr.Type)
	light.SupportsColorTemperature = containsString(lightsSupportingColorTemperature, attr.Type)
	light.SupportsXYColor = containsString(lightsSupportingXYColor, attr.Type)

	light.updateCurrentLightState()
	return nil
}

func (light *Light) updateCurrentLightState() error {
	attr, err := light.HueLight.GetLightAttributes()
	if err != nil {
		return err
	}
	light.Reachable = attr.State.Reachable
	if !light.Reachable {
		light.On = false
		light.CurrentLightState = LightState{0, []float32{0, 0}, 0}
	} else {
		light.On = attr.State.On
		light.CurrentLightState = lightStateFromHueValues(attr.State.Ct, attr.State.Xy, attr.State.Bri)
	}

	// validate lightstate after updating
	if !light.CurrentLightState.isValid() {
		log.Warningf("Validation failed in updateCurrentLightState for light %s", light.Name)
	}
	return nil
}

func (light *Light) update() error {
	// refresh current state
	light.updateCurrentLightState()

	// is the light associated to any schedule?
	if !light.Scheduled {
		return nil
	}

	// Light reachable or on?
	if !light.Reachable || !light.On {
		if light.Tracking {
			log.Printf("💡 Light %s is no longer reachable or turned on. Clearing state.", light.Name)
			light.Tracking = false
			light.Automatic = false
			return nil
		}

		// ignore light because we are not tracking it.
		return nil
	}

	// did the light just appear?
	if !light.Tracking {
		log.Printf("💡 Light %s just appeared. Initializing state to %vK at %v%% brightness.", light.Name, light.TargetLightState.ColorTemperature, light.TargetLightState.Brightness)

		// For initialization we set the state again and again for 10 seconds
		// because during startup the zigbee communication might be unstable
		for index := 0; index < 9; index++ {
			light.setLightState(light.TargetLightState)
		}
		// safe the state of the last iteration
		setLightState, err := light.setLightState(light.TargetLightState)
		if err != nil {
			return err
		}
		if !setLightState.equals(light.TargetLightState) {
			log.Warningf("💡 Light %s - targetLightState and set state differ: %v, %v", light.Name, light.TargetLightState, setLightState)
		}
		light.SavedLightState = setLightState
		light.Tracking = true
		light.Automatic = true
		log.Debugf("💡 Light %s was updated to %vK at %v%% brightness", light.Name, light.SavedLightState.ColorTemperature, light.SavedLightState.Brightness)

		// Debug: Update current state to double check SetState
		if log.GetLevel() == log.DebugLevel {
			light.updateCurrentLightState()
			if !light.CurrentLightState.equals(light.SavedLightState) {
				log.Warningf("💡 Light %s failed to update it's state from %+v to %+v", light.Name, light.CurrentLightState, light.SavedLightState)
			} else {
				log.Debugf("💡 Light %s was successfully updated.", light.Name)
			}
		}
		return nil
	}

	// light in manual state
	if !light.Automatic {
		return nil
	}

	// did the user manually change the light state?
	if !light.SavedLightState.equals(light.CurrentLightState) {
		if log.GetLevel() == log.DebugLevel {
			log.Debugf("💡 Light %s was manually changed. Current state: %+v - Saved state: %+v", light.Name, light.CurrentLightState, light.SavedLightState)
		} else {
			log.Printf("💡 Light %s was changed manually. Disabling Kelvin.", light.Name)
		}
		light.Automatic = false
		return nil
	}

	// Update needed?
	if light.TargetLightState.equals(light.CurrentLightState) {
		return nil
	}
	log.Debugf("💡 Light %s - Target and Current state differ: %v, %v", light.Name, light.TargetLightState, light.CurrentLightState)

	// Light is reachable, turned on and in automatic state. Calculate next state and set it
	nextLightState := calculateNextLightstate(light.CurrentLightState, light.TargetLightState)
	setLightState, err := light.setLightState(nextLightState)
	if err != nil {
		return err
	}

	// Did the light accept the values?
	if !setLightState.equals(nextLightState) {
		log.Warningf("💡 Light %s - nextLightState and set state differ: %v, %v", light.Name, nextLightState, setLightState)
	}
	light.SavedLightState = setLightState
	log.Printf("💡 Light %s was updated to %vK at %v%% brightness", light.Name, light.SavedLightState.ColorTemperature, light.SavedLightState.Brightness)

	// Debug: Update current state to double check SetState
	if log.GetLevel() == log.DebugLevel {
		light.updateCurrentLightState()
		if !light.CurrentLightState.equals(light.SavedLightState) {
			log.Warningf("💡 Light %s failed to update it's state from %+v to %+v", light.Name, light.CurrentLightState, light.SavedLightState)
		} else {
			log.Debugf("💡 Light %s was successfully updated.", light.Name)
		}
	}

	return nil
}

func (light *Light) setLightState(state LightState) (LightState, error) {
	if !state.isValid() {
		log.Warningf("Validation failed in setLightState for light %s", light.Name)
	}

	var hueLightState hue.SetLightState
	colorTemperature, color, brightness := state.convertValuesToHue()

	// Should change colortemperature?
	if state.ColorTemperature != -1 {
		// Set supported colormodes. If both are, the brigde will choose xy
		if light.SupportsXYColor {
			hueLightState.Xy = []float32{color[0], color[1]}
		}
		if light.SupportsColorTemperature {
			hueLightState.Ct = strconv.Itoa(colorTemperature)
		}
	}

	// Should change brightness?
	if state.Brightness != -1 {
		if state.Brightness == 0 {
			// target brightness zero should turn the light off.
			hueLightState.On = "Off"
		} else if light.Dimmable {
			hueLightState.Bri = strconv.Itoa(brightness)
		}
	}
	hueLightState.TransitionTime = strconv.Itoa(lightTransitionIntervalInSeconds * 10) // conversion to 100ms-value

	// Send new state to the light
	hueResults, err := light.HueLight.SetState(hueLightState)
	if err != nil {
		return LightState{0, []float32{0, 0}, 0}, err
	}

	// iterate over result to acquire set values
	for _, result := range hueResults {
		for key, value := range result.Success {
			path := strings.Split(key, "/")

			switch path[len(path)-1] {
			case "ct":
				colorTemperature = int(value.(float64))
			case "xy":
				color = []float32{} // clear old color values
				for _, elem := range value.([]interface{}) {
					color = append(color, float32(elem.(float64)))
				}
			case "bri":
				brightness = int(value.(float64))
			case "on":
				brightness = 0
			}
		}
	}
	setLightState := lightStateFromHueValues(colorTemperature, color, brightness)
	//log.Debugf("Parsed SetLightState from %+v to %+v", hueResults, setLightState)
	if !setLightState.isValid() {
		log.Warningf("Validation failed in setLightState for light %s", light.Name)
	}

	// wait while the light is in transition before returning
	time.Sleep(lightTransitionIntervalInSeconds + 1*time.Second)
	return setLightState, nil
}

func calculateNextLightstate(currentLightState LightState, targetLightState LightState) LightState {
	nextLightState := LightState{-1, []float32{-1, -1}, -1}

	if targetLightState.ColorTemperature != -1 {
		if currentLightState.ColorTemperature < targetLightState.ColorTemperature {
			nextLightState.ColorTemperature = currentLightState.ColorTemperature + 5
		} else {
			nextLightState.ColorTemperature = currentLightState.ColorTemperature - 5
		}
		x, y := colorTemperatureToXYColor(nextLightState.ColorTemperature)
		nextLightState.Color = []float32{float32(x), float32(y)}
	}

	if targetLightState.Brightness != -1 {
		if currentLightState.Brightness < targetLightState.Brightness {
			nextLightState.Brightness = currentLightState.Brightness + 5
		} else {
			nextLightState.Brightness = currentLightState.Brightness - 5
		}
	}

	if !nextLightState.isValid() {
		log.Debugf("Validation failed in calculateNextLightstate")
	}
	return nextLightState
}

func (light *Light) updateConfiguration(configuration *Configuration) {
	light.Configuration = configuration
	light.updateSchedule()
	light.updateInterval()
	light.updateTargetLightState()
}

func (light *Light) updateSchedule() {
	newSchedule, err := light.Configuration.lightScheduleForDay(light.ID, time.Now())
	if err != nil {
		log.Printf("💡 Light %v is not associated to any schedule. Ignoring...", light.Name)
		light.Schedule = newSchedule // assign empty schedule
		light.Scheduled = false
		return
	}
	light.Schedule = newSchedule
	light.Scheduled = true
	log.Printf("💡 Light %s: Activating schedule for %v (sunrise: %v, sunset: %v)", light.Name, light.Schedule.endOfDay.Format("Jan 2 2006"), light.Schedule.sunrise.Time.Format("15:04"), light.Schedule.sunset.Time.Format("15:04"))
	light.updateInterval()
}

func (light *Light) updateInterval() {
	if !light.Scheduled {
		log.Debugf("💡 Light %v is not associated to any schedule. No interval to update...", light.Name)
		return
	}

	newInterval, err := light.Schedule.currentInterval(time.Now())
	if err != nil {
		log.Printf("💡 Light %v has no active interval. Ignoring...", light.Name)
		light.Interval = newInterval // assign empty interval
		return
	}
	if newInterval != light.Interval {
		light.Interval = newInterval
		log.Printf("💡 Light %s: Activating interval %v - %v", light.Name, light.Interval.Start.Time.Format("15:04"), light.Interval.End.Time.Format("15:04"))
	}
}

func (light *Light) updateTargetLightState() {
	if !light.Scheduled {
		log.Debugf("💡 Light %v is not associated to any schedule. No target light state to update...", light.Name)
		return
	}

	newLightState := light.Interval.getTargetLightState(time.Now())
	if !newLightState.isValid() {
		log.Warningf("Validation failed in updateTargetLightState for light %s", light.Name)
	}

	// First initialization of the TargetLightState
	if light.TargetLightState.ColorTemperature == 0 && len(light.TargetLightState.Color) == 0 && light.TargetLightState.Brightness == 0 {
		light.TargetLightState = newLightState
		log.Debugf("💡 Light %s: Initialized target state to %+v", light.Name, light.TargetLightState)
		return
	}

	if !light.TargetLightState.equals(newLightState) {
		light.TargetLightState = newLightState
		log.Debugf("💡 Light %s: Updated target state to %+v", light.Name, light.TargetLightState)
	}
}
