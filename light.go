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
	log "github.com/Sirupsen/logrus"
	"github.com/stefanwichmann/go.hue"
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
	ID                       int
	Name                     string
	HueLight                 hue.Light
	SavedLightState          LightState
	CurrentLightState        LightState
	TargetLightState         LightState
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
		case <-time.After(light.Schedule.endOfDay.Sub(time.Now()) + 1*time.Second):
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
	return nil
}

func (light *Light) update() error {
	// refresh current state
	light.updateCurrentLightState()

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
		light.SavedLightState = setLightState
		//light.TargetLightState = setLightState
		light.Tracking = true
		light.Automatic = true
		log.Debugf("💡 Light %s was updated to %vK at %v%% brightness", light.Name, light.TargetLightState.ColorTemperature, light.TargetLightState.Brightness)
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

	// Light is reachable, turned on and in automatic state. Update to new state.
	setLightState, err := light.setLightState(light.TargetLightState)
	if err != nil {
		return err
	}

	// Debug: Compare values
	if !setLightState.equals(light.TargetLightState) {
		log.Debugf("💡 Light %s - Target and Set state differ: %v, %v", light.Name, light.TargetLightState, setLightState)
	}

	light.SavedLightState = setLightState
	//light.TargetLightState = setLightState
	log.Printf("💡 Light %s was updated to %vK at %v%% brightness", light.Name, light.TargetLightState.ColorTemperature, light.TargetLightState.Brightness)
	return nil
}

func (light *Light) setLightState(state LightState) (LightState, error) {
	// Don't send repeated "On" as this slows the bridge down (see https://developers.meethue.com/faq-page #Performance)
	var hueLightState hue.SetLightState
	colorTemperature, color, brightness := state.convertValuesToHue()
	if light.SupportsColorTemperature && state.ColorTemperature != 0 {
		hueLightState.Ct = strconv.Itoa(colorTemperature)
	} else if light.SupportsXYColor && state.Color[0] != 0 && state.Color[1] != 0 {
		hueLightState.Xy = []float32{color[0], color[1]}
	}
	if light.Dimmable && state.Brightness != 0 {
		hueLightState.Bri = strconv.Itoa(brightness)
	}
	hueLightState.TransitionTime = strconv.Itoa(lightTransitionIntervalInSeconds * 10) // conversion to 100ms-value

	results, err := light.HueLight.SetState(hueLightState)
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
		return
	}
	light.Schedule = newSchedule
	log.Printf("💡 Light %s: Activating schedule for %v (sunrise: %v, sunset: %v)", light.Name, light.Schedule.endOfDay.Format("Jan 2 2006"), light.Schedule.sunrise.Time.Format("15:04"), light.Schedule.sunset.Time.Format("15:04"))
	light.updateInterval()
}

func (light *Light) updateInterval() {
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
	newLightState, err := light.Interval.calculateLightStateInInterval(time.Now())
	if err != nil {
		// interval seems to be wrong. Ignore and try again in one minute.
		log.Debugln(err)
		return
	}
	//log.Debugf("💡 Light %s: Updating target lightstate from %+v to %+v)", light.Name, light.TargetLightState, newLightState)

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
