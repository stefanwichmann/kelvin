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

import "log"

var applicationVersion = "development"

func main() {
	log.Printf("🚀 Kelvin %v starting up...\n", applicationVersion)
	go CheckForUpdate(applicationVersion)

	// load config or create new on
	configuration, err := InitializeConfiguration()
	if err != nil {
		log.Fatal(err)
	}

	// find bridge
	bridge, err := InitializeBridge(configuration.Bridge.IP, configuration.Bridge.Username)
	if err != nil {
		log.Fatal(err)
	} else {
		configuration.Bridge.IP = bridge.bridgeIP
		configuration.Bridge.Username = bridge.username
	}
	err = bridge.printDevices()
	if err != nil {
		log.Fatal(err)
	}

	// find location
	location, err := InitializeLocation(configuration.Location.Latitude, configuration.Location.Longitude)
	if err != nil {
		log.Fatal(err)
	} else {
		configuration.Location.Latitude = location.Latitude
		configuration.Location.Longitude = location.Longitude
	}

	// Save configuration
	err = configuration.Write()
	if err != nil {
		log.Fatal(err)
	}

	// Lights and their channels
	hueLights, err := bridge.Lights()
	if err != nil {
		log.Fatal(err.Error())
	}
	var lightChannels []chan LightState
	for _, hueLight := range hueLights {
		hueLight := hueLight
		// Ignore devices that don't support dimming and colors
		if !hueLight.dimmable && !hueLight.supportsXYColor && !hueLight.supportsColorTemperature {
			log.Printf("⌘ Device %v doesn't support any functionality we use. Exlude it from unnessesary polling.\n", hueLight.name)
			continue
		}
		lightChannel := make(chan LightState, 1)
		lightChannels = append(lightChannels, lightChannel)
		go hueLight.updateCyclic(lightChannel)
	}

	// time intervals off the day
	var day Day
	intervalChannel := make(chan Interval, 1)
	go day.updateCyclic(configuration, location, intervalChannel)

	// light state channel
	lightStateChannel := make(chan LightState, 5)

	for {
		select {
		case state := <-lightStateChannel:
			// Send new state to all lights
			for _, light := range lightChannels {
				state := state
				light <- state
			}
		case interval := <-intervalChannel:
			log.Printf("⌘ - New interval received: %v - %v\n", interval.Start.Time.Format("15:04"), interval.End.Time.Format("15:04"))
			interval = interval
			go interval.updateCyclic(lightStateChannel)
		}
	}
}
