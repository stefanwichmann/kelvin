// MIT License
//
// Copyright (c) 2019 Stefan Wichmann
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
	"time"

	log "github.com/Sirupsen/logrus"
	hue "github.com/stefanwichmann/go.hue"
)

const initializationDuration = 3 * time.Second

// Light represents a light kelvin can automate in your system.
type Light struct {
	ID               int        `json:"id"`
	Name             string     `json:"name"`
	HueLight         HueLight   `json:"-"`
	TargetLightState LightState `json:"targetLightState,omitempty"`
	Scheduled        bool       `json:"scheduled"`
	Reachable        bool       `json:"reachable"`
	On               bool       `json:"on"`
	Tracking         bool       `json:"-"`
	Automatic        bool       `json:"automatic"`
	Initializing     bool       `json:"-"`
	Schedule         Schedule   `json:"-"`
	Interval         Interval   `json:"interval"`
	Appearance       time.Time  `json:"-"`
}

func (light *Light) updateCurrentLightState(attr hue.LightAttributes) error {
	light.HueLight.updateCurrentLightState(attr)
	light.Reachable = light.HueLight.Reachable
	light.On = light.HueLight.On
	return nil
}

func (light *Light) update(transistionTime time.Duration) (bool, error) {
	// Is the light associated to any schedule?
	if !light.Scheduled {
		return false, nil
	}

	// If the light is not reachable anymore clean up
	if !light.Reachable {
		if light.Tracking {
			log.Printf("💡 Light %s - Light is no longer reachable. Clearing state...", light.Name)
			light.Tracking = false
			light.Automatic = false
			light.Initializing = false
			return false, nil
		}

		// Ignore light because we are not tracking it.
		return false, nil
	}

	// If the light was turned off clean up
	if !light.On {
		if light.Tracking {
			log.Printf("💡 Light %s - Light was turned off. Clearing state...", light.Name)
			light.Tracking = false
			light.Automatic = false
			light.Initializing = false
			return false, nil
		}

		// Ignore light because we are not tracking it.
		return false, nil
	}

	// Did the light just appear?
	if !light.Tracking {
		log.Printf("💡 Light %s - Light just appeared.", light.Name)
		light.Tracking = true
		light.Appearance = time.Now()

		// Should we auto-enable Kelvin?
		if light.Schedule.enableWhenLightsAppear {
			log.Printf("💡 Light %s - Initializing state to %vK at %v%% brightness.", light.Name, light.TargetLightState.ColorTemperature, light.TargetLightState.Brightness)

			err := light.HueLight.setLightState(light.TargetLightState.ColorTemperature, light.TargetLightState.Brightness, transistionTime)
			if err != nil {
				log.Debugf("💡 Light %s - Could not initialize light after %v", light.Name, time.Since(light.Appearance))
				return true, err
			}

			light.Automatic = true
			light.Initializing = true
			log.Debugf("💡 Light %s - Light was initialized to %vK at %v%% brightness", light.Name, light.TargetLightState.ColorTemperature, light.TargetLightState.Brightness)
			return true, nil
		}
	}

	// Ignore light if it was changed manually
	if !light.Automatic {
		// return if we should ignore color temperature and brightness
		if light.TargetLightState.ColorTemperature == -1 && light.TargetLightState.Brightness == -1 {
			return false, nil
		}

		// if status == scene state --> Activate Kelvin
		if light.HueLight.hasState(light.TargetLightState.ColorTemperature, light.TargetLightState.Brightness) {
			log.Printf("💡 Light %s - Detected matching target state. Activating Kelvin...", light.Name)
			light.Automatic = true
			light.Initializing = true

			// set correct target lightstate on HueLight
			err := light.HueLight.setLightState(light.TargetLightState.ColorTemperature, light.TargetLightState.Brightness, transistionTime)
			if err != nil {
				return true, err
			}
			log.Debugf("💡 Light %s - Updated light state to %vK at %v%% brightness (Scene detection)", light.Name, light.TargetLightState.ColorTemperature, light.TargetLightState.Brightness)
			return true, nil
		}

		// Light was changed manually and does not conform to scene detection
		return false, nil
	}

	// Keep adjusting the light state for 10 seconds after the light appeared
	if light.Initializing {
		log.Debugf("💡 Light %s - Light in initialization for %v (TargetColorTemperature: %d, CurrentColorTemperature: %d, TargetColor: %v, CurrentColor: %v, TargetBrightness: %d, CurrentBrightness: %d)", light.Name, time.Since(light.Appearance), light.HueLight.TargetColorTemperature, light.HueLight.CurrentColorTemperature, light.HueLight.TargetColor, light.HueLight.CurrentColor, light.HueLight.TargetBrightness, light.HueLight.CurrentBrightness)
		hasChanged := light.HueLight.hasChanged()

		// Disable initialization phase if 10 seconds have passed and the light state has been adopted
		if time.Now().After(light.Appearance.Add(initializationDuration)) && !hasChanged {
			log.Debugf("💡 Light %s - Ending initialization phase after %v", light.Name, time.Since(light.Appearance))
			light.Initializing = false
		}

		if hasChanged {
			err := light.HueLight.setLightState(light.TargetLightState.ColorTemperature, light.TargetLightState.Brightness, transistionTime)
			if err != nil {
				return true, err
			}
			log.Debugf("💡 Light %s - Adjusting light state to %vK at %v%% brightness (Initialization)", light.Name, light.TargetLightState.ColorTemperature, light.TargetLightState.Brightness)
			return true, nil
		}

		return false, nil
	}

	// Did the user manually change the light state?
	if light.HueLight.hasChanged() {
		if log.GetLevel() == log.DebugLevel {
			log.Debugf("💡 Light %s - Light state has been changed manually after %v (TargetColorTemperature: %d, CurrentColorTemperature: %d, TargetColor: %v, CurrentColor: %v, TargetBrightness: %d, CurrentBrightness: %d)", light.Name, time.Since(light.Appearance), light.HueLight.TargetColorTemperature, light.HueLight.CurrentColorTemperature, light.HueLight.TargetColor, light.HueLight.CurrentColor, light.HueLight.TargetBrightness, light.HueLight.CurrentBrightness)
		} else {
			log.Printf("💡 Light %s - Light state has been changed manually. Disabling Kelvin...", light.Name)
		}
		light.Automatic = false
		return false, nil
	}

	// Update of lightstate needed?
	if light.HueLight.hasState(light.TargetLightState.ColorTemperature, light.TargetLightState.Brightness) {
		return false, nil
	}

	// Light is turned on and in automatic state. Set target lightstate.
	err := light.HueLight.setLightState(light.TargetLightState.ColorTemperature, light.TargetLightState.Brightness, transistionTime)
	if err != nil {
		return true, err
	}

	log.Printf("💡 Light %s - Updated light state to %vK at %v%% brightness", light.Name, light.TargetLightState.ColorTemperature, light.TargetLightState.Brightness)
	return true, nil
}

func (light *Light) updateSchedule(schedule Schedule) {
	light.Schedule = schedule
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
		log.Warningf("💡 Light %s - Could not determine interval for current schedule: %v", light.Name, err)
		return
	}
	if newInterval != light.Interval {
		light.Interval = newInterval
		log.Printf("💡 Light %s - Activating interval %v - %v", light.Name, light.Interval.Start.Time.Format("15:04"), light.Interval.End.Time.Format("15:04"))
	}
}

func (light *Light) updateTargetLightState() bool {
	if !light.Scheduled {
		log.Debugf("💡 Light %s - Light is not associated to any schedule. No target light state to update...", light.Name)
		return false
	}

	// Calculate the target lightstate from the interval
	newLightState := light.Interval.calculateLightStateInInterval(time.Now())

	// Did the target light state change?
	if newLightState.equals(light.TargetLightState) {
		return false
	}

	// First initialization of the TargetLightState?
	if light.TargetLightState.ColorTemperature == 0 && light.TargetLightState.Brightness == 0 {
		log.Debugf("💡 Light %s - Initialized target light state for the interval %v - %v to %+v", light.Name, light.Interval.Start.Time.Format("15:04"), light.Interval.End.Time.Format("15:04"), newLightState)
	} else {
		log.Debugf("💡 Light %s - Updated target light state for the interval %v - %v from %+v to %+v", light.Name, light.Interval.Start.Time.Format("15:04"), light.Interval.End.Time.Format("15:04"), light.TargetLightState, newLightState)
	}

	light.TargetLightState = newLightState
	return true
}
