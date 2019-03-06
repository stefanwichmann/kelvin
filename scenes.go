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

import log "github.com/Sirupsen/logrus"
import hue "github.com/stefanwichmann/go.hue"
import "time"
import "strings"

func updateScenes() {
	log.Debugf("🎨 Updating scenes...")
	scenes, _ := bridge.bridge.AllScenes()
	for _, scene := range scenes {
		if strings.Contains(strings.ToLower(scene.Name), "kelvin") {
			for _, schedule := range configuration.Schedules {
				if strings.Contains(strings.ToLower(scene.Name), strings.ToLower(schedule.Name)) {
					log.Debugf("🎨 Updating scene \"%s\" for schedule \"%s\"...", scene.Name, schedule.Name)
					updateSceneForSchedule(scene, schedule)
				}
			}
		}
	}
}

func updateSceneForSchedule(scene *hue.Scene, lightSchedule LightSchedule) {
	// Updating lights
	var modifyScene hue.ModifyScene
	modifyScene.Lights = toStringArray(lightSchedule.AssociatedDeviceIDs)

	_, err := scene.Modify(modifyScene)
	if err != nil {
		log.Warningf("🎨 %v", err)
		return
	}

	// Updating light states
	light := lightSchedule.AssociatedDeviceIDs[0]
	schedule, err := configuration.lightScheduleForDay(light, time.Now())
	if err != nil {
		log.Warningf("🎨 %v", err)
		return
	}

	interval, err := schedule.currentInterval(time.Now())
	if err != nil {
		log.Warningf("🎨 %v", err)
		return
	}

	state := interval.calculateLightStateInInterval(time.Now())

	var modifyState hue.ModifyLightState
	modifyState.On = true // turn lights on when the scene is activated

	if state.ColorTemperature != -1 {
		modifyState.ColorTemperature = uint16(mapColorTemperature(state.ColorTemperature))
		modifyState.Xy = colorTemperatureToXYColor(state.ColorTemperature)
	}
	if state.Brightness != -1 {
		modifyState.Brightness = uint8(mapBrightness(state.Brightness))
	}

	_, err = scene.ModifyLightStates(modifyState)
	if err != nil {
		log.Warningf("🎨 %v", err)
	}

	log.Debugf("🎨 Successfully updated scene \"%s\"", scene.Name)
}
