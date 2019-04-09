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
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/Sirupsen/logrus"
)

var applicationVersion = "development"
var debug = flag.Bool("debug", false, "Enable debug logging")
var logFile = flag.String("log", "", "Redirect log output to specified file")
var configurationFile = flag.String("configuration", absolutePath("config.json"), "Specify the filename of the configuration to load")
var forceUpdate = flag.Bool("forceUpdate", false, "Update to new major version")
var enableWebInterface = flag.Bool("enableWebInterface", false, "Enable the web interface at startup")

var configuration *Configuration
var bridge = &HueBridge{}
var lights []*Light

const lightUpdateInterval = 1 * time.Second
const stateUpdateInterval = 1 * time.Minute
const timeBetweenCalls = 200 * time.Millisecond // see https://developers.meethue.com/develop/application-design-guidance/hue-system-performance/
const lightTransistionTime = 400 * time.Millisecond

func main() {
	flag.Parse()
	configureLogging()
	log.Printf("🤖 Kelvin %v starting up... 🚀", applicationVersion)
	log.Debugf("🤖 Current working directory: %v", workingDirectory())
	go CheckForUpdate(applicationVersion, *forceUpdate)
	go validateSystemTime()
	go handleSIGHUP()

	// Load configuration or create a new one
	conf, err := InitializeConfiguration(*configurationFile, *enableWebInterface)
	if err != nil {
		log.Fatal(err)
	}
	configuration = &conf

	// Start web interface
	go startInterface()

	// Find Hue bridge
	log.Printf("🤖 Initializing bridge connection...")
	for {
		err = bridge.InitializeBridge(configuration)
		if err != nil {
			log.Errorf("Could not initialze bridge: %v - Retrying...", err)
			time.Sleep(10 * time.Second)
		} else {
			break
		}
	}

	// Find geo location
	_, err = InitializeLocation(configuration)
	if err != nil {
		log.Warning(err)
	}

	// Save configuration
	err = configuration.Write()
	if err != nil {
		log.Fatal(err)
	}

	// Initialize lights
	l, err := bridge.Lights()
	if err != nil {
		log.Warning(err)
	}
	time.Sleep(timeBetweenCalls)
	printDevices(l)
	for _, light := range l {
		light := light

		// Filter devices we can't control
		if !light.HueLight.supportsColorTemperature() && !light.HueLight.supportsBrightness() {
			log.Printf("🤖 Light %s - This device doesn't support any functionality Kelvin uses. Ignoring...", light.Name)
		} else {
			lights = append(lights, light)
			updateScheduleForLight(light)
		}
	}

	// Initialize scenes
	updateScenes()
	time.Sleep(timeBetweenCalls)

	// Start cyclic update for all lights and scenes
	log.Debugf("🤖 Starting cyclic update...")
	lightUpdateTimer := time.NewTimer(lightUpdateInterval)
	stateUpdateTick := time.Tick(stateUpdateInterval)
	newDayTimer := time.After(durationUntilNextDay())
	for {
		select {
		case <-newDayTimer:
			// A new day has begun, calculate new schedule
			log.Printf("🤖 Calculating schedule for %v", time.Now().Format("Jan 2 2006"))
			for _, light := range lights {
				light := light
				updateScheduleForLight(light)
			}
			updateScenes()
			time.Sleep(timeBetweenCalls)
			newDayTimer = time.After(durationUntilNextDay())
		case <-stateUpdateTick:
			// update interval and color every minute
			updated := false
			for _, light := range lights {
				light := light
				light.updateInterval()
				if light.updateTargetLightState() {
					updated = true
				}
			}
			// update scenes
			if updated {
				updateScenes()
				time.Sleep(timeBetweenCalls)
			}
		case <-lightUpdateTimer.C:
			states, err := bridge.LightStates()
			if err != nil {
				log.Warningf("🤖 Failed to update light states: %v", err)
			}
			time.Sleep(timeBetweenCalls)

			for _, light := range lights {
				light := light
				currentLightState, found := states[light.ID]
				if found {
					light.updateCurrentLightState(currentLightState)
					updated, err := light.update(lightTransistionTime)
					if err != nil {
						log.Warningf("🤖 Light %s - Failed to update light: %v", light.Name, err)
					}
					if updated {
						log.Debugf("🤖 Light %s - Updated light state. Awaiting transition...", light.Name)
						time.Sleep(timeBetweenCalls)
					}
				} else {
					log.Warningf("🤖 Light %s - No current light state found", light.Name)
				}
			}

			lightUpdateTimer.Reset(lightUpdateInterval)
		}
	}
}

func updateScheduleForLight(light *Light) {
	schedule, err := configuration.lightScheduleForDay(light.ID, time.Now())
	if err != nil {
		log.Printf("🤖 Light %s - Light is not associated to any schedule. Ignoring...", light.Name)
		light.Schedule = schedule // Assign empty schedule
		light.Scheduled = false
	} else {
		light.updateSchedule(schedule)
		light.updateTargetLightState()
	}
}

func printDevices(l []*Light) {
	log.Printf("🤖 Devices found on current bridge:")
	log.Printf("| %-32s | %3v | %-5v | %-8v | %-11v | %-5v | %17v |", "Name", "ID", "On", "Dimmable", "Temperature", "Color", "Temperature range")
	for _, light := range l {
		ctRange := ""
		if light.HueLight.supportsColorTemperature() {
			ctRange = fmt.Sprintf("%dK - %dK", light.HueLight.MinimumColorTemperature, 6500)
		}
		log.Printf("| %-32s | %3v | %-5v | %-8v | %-11v | %-5v | %17v |", light.Name, light.ID, light.On, light.HueLight.Dimmable, light.HueLight.SupportsColorTemperature, light.HueLight.SupportsXYColor, ctRange)
	}
}

func handleSIGHUP() {
	sighup := make(chan os.Signal, 1)
	signal.Notify(sighup, syscall.SIGHUP)
	<-sighup // wait for signal
	log.Printf("🤖 Received signal SIGHUP. Restarting...")
	Restart()
}

func configureLogging() {
	formatter := new(log.TextFormatter)
	formatter.FullTimestamp = true
	formatter.TimestampFormat = "2006/02/01 15:04:05"
	log.SetFormatter(formatter)
	if *debug {
		log.SetLevel(log.DebugLevel)
	}
	if logFile != nil && *logFile != "" {
		file, err := os.OpenFile(*logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err == nil {
			log.SetOutput(file)
		} else {
			log.Info("🤖 Failed to log to file, using default stderr")
		}
	}
}

func validateSystemTime() {
	// validate local clock as it forms the basis for all time calculations.
	valid, err := IsLocalTimeValid()
	if err != nil {
		log.Errorf("🤖 ERROR: Could not validate system time: %v", err)
	}
	if !valid {
		log.Warningf("🤖 WARNING: Your local system time seems to be more than one minute off. Timings may be inaccurate.")
	} else {
		log.Debugf("🤖 Local system time validated.")
	}
}
