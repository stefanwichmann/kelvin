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
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/stefanwichmann/go.hue"
	"strconv"
	"strings"
	"time"
)

// HueBridge represents the Philips Hue bridge in
// your system.
// It is used to communicate with all devices.
type HueBridge struct {
	bridge           hue.Bridge
	bridgeIP         string
	username         string
	ignoredDeviceIDs []int
}

const hueBridgeAppName = "kelvin"

// InitializeBridge creates and returns an initialized HueBridge.
// If you have a valid configuration this will be used. Otherwise a local
// discovery will be started, followed by a user registration on your bridge.
func InitializeBridge(configuration *Configuration) (HueBridge, error) {
	var bridge HueBridge
	bridge.ignoredDeviceIDs = configuration.IgnoredDeviceIDs

	if configuration.Bridge.IP != "" && configuration.Bridge.Username != "" {
		// known bridge configuration
		log.Println("⌘ Initializing bridge from configuration")
		bridge.bridgeIP = configuration.Bridge.IP
		bridge.username = configuration.Bridge.Username

		err := bridge.connect()
		if err != nil {
			return bridge, err
		}
		log.Println("⌘ Connection to bridge established")
		return bridge, nil
	}

	// no known bridge or username
	log.Println("⌘ No bridge configuration found. Starting local discovery...")
	err := bridge.discover()
	if err != nil {
		return bridge, err
	}

	configuration.Bridge.IP = bridge.bridgeIP
	configuration.Bridge.Username = bridge.username
	configuration.Modified = true
	return bridge, nil
}

// Lights return all known lights on your bridge.
func (bridge *HueBridge) Lights() ([]Light, error) {
	var lights []Light
	hueLights, err := bridge.bridge.GetAllLights()
	if err != nil {
		return lights, err
	}

	for _, hueLight := range hueLights {
		var light Light
		light.id, err = strconv.Atoi(hueLight.Id)
		if err != nil {
			return lights, err
		}

		light.hueLight = *hueLight
		light.initialize()

		// ignore current device?
		if containsInt(bridge.ignoredDeviceIDs, light.id) {
			light.ignored = true
		}
		lights = append(lights, light)
	}

	return lights, nil
}

func (bridge *HueBridge) printDevices() error {
	lights, err := bridge.Lights()
	if err != nil {
		return err
	}

	log.Printf("💡 Devices found on current bridge:")
	log.Printf("| %-20s | %3v | %-9v | %-5v | %-7v | %-8v | %-11v | %-5v | %-9v | %-8v |", "Name", "ID", "Reachable", "On", "Ignored", "Dimmable", "Temperature", "Color", "Cur. Temp", "Cur. Bri")
	for _, light := range lights {
		var temp string
		if light.supportsColorTemperature == false && light.supportsXYColor == false {
			temp = "-"
		} else {
			temp = strings.Join([]string{strconv.Itoa(light.currentLightState.colorTemperature), "K"}, "")
		}
		log.Printf("| %-20s | %3v | %-9v | %-5v | %-7v | %-8v | %-11v | %-5v | %9v | %8v |", light.name, light.id, light.reachable, light.on, light.ignored, light.dimmable, light.supportsColorTemperature, light.supportsXYColor, temp, light.currentLightState.brightness)
	}
	return nil
}

func (bridge *HueBridge) discover() error {
	bridges, err := hue.DiscoverBridges(false)
	if err != nil {
		return err
	}
	if len(bridges) == 0 {
		return errors.New("Bridge discovery failed. Please configure manually in config.json.")
	}
	if len(bridges) > 1 {
		log.Printf("Found multiple bridges. Using first one.")
	}
	hueBridge := bridges[0] // use the first locator

	log.Printf("⌘ Found bridge. Starting user registration.")
	fmt.Printf("PLEASE PUSH THE BLUE BUTTON ON YOUR HUE BRIDGE.")
	for index := 0; index < 30; index++ {
		time.Sleep(5 * time.Second)
		fmt.Printf(".")
		// try user creation, will fail if the button wasn't pressed.
		err := hueBridge.CreateUser(hueBridgeAppName)
		if err != nil {
			return err
		}

		if hueBridge.Username != "" {
			// registration successful
			fmt.Printf(" Success!\n")

			bridge.bridge = hueBridge
			bridge.username = hueBridge.Username
			bridge.bridgeIP = hueBridge.IpAddr
			return nil
		}
	}
	return errors.New("Registration at bridge timed out")
}

func (bridge *HueBridge) connect() error {
	if bridge.bridgeIP == "" {
		return errors.New("No bridge IP configured")
	}

	if bridge.username == "" {
		return errors.New("No username on bridge configured")
	}
	bridge.bridge = *hue.NewBridge(bridge.bridgeIP, bridge.username)

	// Test bridge
	_, err := bridge.bridge.Search()
	if err != nil {
		return err
	}

	return nil
}
