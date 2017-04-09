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
import "os/signal"
import "syscall"
import "os"
import "sync"

var applicationVersion = "development"

func main() {
	log.Printf("Kelvin %v starting up... 🚀\n", applicationVersion)
	go CheckForUpdate(applicationVersion)
	go handleSIGHUP()

	// validate local clock as it forms the basis for all time calculations.
	log.Printf("Validating local system time...\n")
	valid, err := IsLocalTimeValid()
	if err != nil {
		log.Fatal(err)
	}
	if !valid {
		log.Printf("WARNING: Your local system time seems to be more than one minute off. Timings may be inaccurate.\n")
	} else {
		log.Printf("Local system time seems to be valid.\n")
	}

	// load configuration or create a new one
	configuration, err := InitializeConfiguration()
	if err != nil {
		log.Fatal(err)
	}

	// find bridge
	bridge, err := InitializeBridge(configuration.Bridge.IP, configuration.Bridge.Username, configuration.IgnoredDeviceIDs)
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

	// start routine for every light
	hueLights, err := bridge.Lights()
	if err != nil {
		log.Fatal(err)
	}
	var wg sync.WaitGroup
	for _, hueLight := range hueLights {
		hueLight := hueLight
		wg.Add(1)
		go func() {
			hueLight.updateCyclic(configuration)
			wg.Done()
		}()
	}
	wg.Wait()
	log.Printf("All routines ended...\n")
}

func handleSIGHUP() {
	sighup := make(chan os.Signal, 1)
	signal.Notify(sighup, syscall.SIGHUP)
	<-sighup // wait for signal
	log.Printf("Received signal SIGHUP. Restarting...\n")
	Restart()
}
