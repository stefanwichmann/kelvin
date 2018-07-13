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

import (
	log "github.com/Sirupsen/logrus"
	"time"
)

// Light represents a light kelvin can automate in your system.
type Light struct {
	ID               int            `json:"id"`
	Name             string         `json:"name"`
	HueLight         HueLight       `json:"-"`
	TargetLightState LightState     `json:"targetLightState,omitempty"`
	Scheduled        bool           `json:"scheduled"`
	Reachable        bool           `json:"reachable"`
	On               bool           `json:"on"`
	Tracking         bool           `json:"-"`
	Automatic        bool           `json:"automatic"`
	Configuration    *Configuration `json:"-"`
	Schedule         Schedule       `json:"-"`
	Interval         Interval       `json:"interval"`
}

const lightUpdateIntervalInSeconds = 1
const stateUpdateIntervalInSeconds = 60

func (light *Light) updateCyclic(configuration *Configuration) {
	light.Configuration = configuration

	// Filter devices we can't control
	if !light.HueLight.supportsColorTemperature() && !light.HueLight.supportsBrightness() {
		log.Printf("💡 Light %s - This device doesn't support any functionality Kelvin uses. Ignoring...", light.Name)
		return
	}

	light.updateSchedule()
	light.updateInterval()
	light.updateTargetLightState()

	// Start cyclic polling
	log.Debugf("💡 Light %s - Starting cyclic update...", light.Name)
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
	err := light.HueLight.initialize()
	if err != nil {
		return err
	}

	// initialize values
	light.Name = light.HueLight.Name
	light.Reachable = light.HueLight.Reachable
	light.On = light.HueLight.On

	return nil
}

func (light *Light) updateCurrentLightState() error {
	err := light.HueLight.updateCurrentLightState()
	if err != nil {
		return err
	}
	light.Reachable = light.HueLight.Reachable
	light.On = light.HueLight.On
	return nil
}

func (light *Light) update() error {
	// Is the light associated to any schedule?
	if !light.Scheduled {
		return nil
	}

	// Refresh current light state from bridge
	light.updateCurrentLightState()

	// If the light is not reachable anymore clean up
	if !light.Reachable {
		if light.Tracking {
			log.Printf("💡 Light %s - Light is no longer reachable. Clearing state...", light.Name)
			light.Tracking = false
			light.Automatic = false
			return nil
		}

		// Ignore light because we are not tracking it.
		return nil
	}

	// If the light was turned off clean up
	if !light.On {
		if light.Tracking {
			log.Printf("💡 Light %s - Light was turned off. Clearing state...", light.Name)
			light.Tracking = false
			light.Automatic = false
			return nil
		}

		// Ignore light because we are not tracking it.
		return nil
	}

	// Did the light just appear?
	if !light.Tracking {
		log.Printf("💡 Light %s - Light just appeared.", light.Name)
		light.Tracking = true

		// Should we auto-enable Kelvin?
		if light.Schedule.enableWhenLightsAppear {
			log.Printf("💡 Light %s - Initializing state to %vK at %v%% brightness.", light.Name, light.TargetLightState.ColorTemperature, light.TargetLightState.Brightness)

			// For initialization we set the state again and again for 10 seconds
			// because during startup the zigbee communication might be unstable
			for index := 0; index < 10; index++ {
				err := light.HueLight.setLightState(light.TargetLightState.ColorTemperature, light.TargetLightState.Brightness)
				if err != nil {
					return err
				}
			}

			light.Automatic = true
			log.Debugf("💡 Light %s - Light was updated to %vK at %v%% brightness", light.Name, light.TargetLightState.ColorTemperature, light.TargetLightState.Brightness)

			return nil
		}
	}

	// Ignore light if it was changed manually
	if !light.Automatic {
		// if status == scene state --> Activate Kelvin
		if light.HueLight.hasState(light.TargetLightState.ColorTemperature, light.TargetLightState.Brightness) {
			log.Printf("💡 Light %s - Detected matching target state. Activating Kelvin...", light.Name)
			light.Automatic = true

			// set correct target lightstate on HueLight
			err := light.HueLight.setLightState(light.TargetLightState.ColorTemperature, light.TargetLightState.Brightness)
			if err != nil {
				return err
			}
		}
		return nil
	}

	// Did the user manually change the light state?
	if light.HueLight.hasChanged() {
		if log.GetLevel() == log.DebugLevel {
			log.Debugf("💡 Light %s - Light state has been changed manually: %+v", light.Name, light.HueLight)
		} else {
			log.Printf("💡 Light %s - Light state has been changed manually. Disabling Kelvin...", light.Name)
		}
		light.Automatic = false
		return nil
	}

	// Update of lightstate needed?
	if light.HueLight.hasState(light.TargetLightState.ColorTemperature, light.TargetLightState.Brightness) {
		return nil
	}

	// Light is turned on and in automatic state. Set target lightstate.
	err := light.HueLight.setLightState(light.TargetLightState.ColorTemperature, light.TargetLightState.Brightness)
	if err != nil {
		return err
	}

	log.Printf("💡 Light %s - Updated light state to %vK at %v%% brightness", light.Name, light.TargetLightState.ColorTemperature, light.TargetLightState.Brightness)
	return nil
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
		log.Printf("💡 Light %s - Light is not associated to any schedule. Ignoring...", light.Name)
		light.Schedule = newSchedule // Assign empty schedule
		light.Scheduled = false
		return
	}
	light.Schedule = newSchedule
	light.Scheduled = true
	log.Printf("💡 Light %s - Activating schedule for %v (Sunrise: %v, Sunset: %v)", light.Name, light.Schedule.endOfDay.Format("Jan 2 2006"), light.Schedule.sunrise.Time.Format("15:04"), light.Schedule.sunset.Time.Format("15:04"))
	light.updateInterval()
}

func (light *Light) updateInterval() {
	if !light.Scheduled {
		log.Debugf("💡 Light %s - Light is not associated to any schedule. No interval to update...", light.Name)
		return
	}

	newInterval, err := light.Schedule.currentInterval(time.Now())
	if err != nil {
		log.Printf("💡 Light %s - Light has no active interval. Ignoring...", light.Name)
		light.Interval = newInterval // Assign empty interval
		return
	}
	if newInterval != light.Interval {
		light.Interval = newInterval
		log.Printf("💡 Light %s - Activating interval %v - %v", light.Name, light.Interval.Start.Time.Format("15:04"), light.Interval.End.Time.Format("15:04"))
	}
}

func (light *Light) updateTargetLightState() {
	if !light.Scheduled {
		log.Debugf("💡 Light %s - Light is not associated to any schedule. No target light state to update...", light.Name)
		return
	}

	// Calculate the target lightstate from the interval
	newLightState := light.Interval.calculateLightStateInInterval(time.Now())
	log.Debugf("💡 Light %s - The calculated lightstate for the interval %v - %v is %+v", light.Name, light.Interval.Start.Time.Format("15:04"), light.Interval.End.Time.Format("15:04"), newLightState)

	// First initialization of the TargetLightState?
	if light.TargetLightState.ColorTemperature == 0 && light.TargetLightState.Brightness == 0 {
		light.TargetLightState = newLightState
		log.Debugf("💡 Light %s - Initialized target light state to %+v", light.Name, light.TargetLightState)
		return
	}

	light.TargetLightState = newLightState
	log.Debugf("💡 Light %s - Updated target state to %+v", light.Name, light.TargetLightState)
}
