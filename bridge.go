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
	"github.com/dillonhafer/go.hue"
	"strconv"
	"time"
	"log"
	"strings"
)

type HueBridge struct {
	bridgeLocator hue.BridgeLocator
	bridge        hue.Bridge
	bridgeIP      string
	username      string
}

const hueBridgeAppName = "kelvin"

func InitializeBridge(ip string, username string) (HueBridge, error) {
	var bridge HueBridge
	if ip != "" {
		log.Println("⌘ Initializing bridge from configuration")
		bridge.bridgeIP = ip
		if username != "" {
			log.Println("⌘ Username found in configuration")
			bridge.username = username

			// connect to know bridge with known username
			log.Printf("⌘ Connecting to bridge on IP %s ...\n", bridge.bridgeIP)
			err := bridge.Connect()
			if err != nil {
				return bridge, err
			}
			log.Println("⌘ Connection to bridge established")
		} else {
			// connect to known brigde without user
			log.Printf("⌘ Registering new user at Hue Bridge")
			err := bridge.Register()
			if err != nil {
				return bridge, err
			}
			log.Println("⌘ Registered new user at bridge.")
		}
	} else {
		// no known bridge
		log.Println("⌘ No bridge found in configuration. Searching for new one...")
		err := bridge.Discover()
		if err != nil {
			return bridge, err
		}
		log.Println("⌘ Found bridge. Registering new user")
		err = bridge.Register()
		if err != nil {
			return bridge, err
		}
		err = bridge.Connect()
		if err != nil {
			return bridge, err
		}
	}
	return bridge, nil
}

func (self *HueBridge) Lights() ([]Light, error) {
	var lights []Light
	hueLights, err := self.bridge.GetAllLights()
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
		lights = append(lights, light)
	}

	return lights, nil
}

func (self *HueBridge) printDevices() error {
	lights, err := self.Lights()
	if err != nil {
		return err
	}

	log.Printf("💡 Devices found on current bridge:\n")
	log.Printf("| %-20s | %3v | %-9v | %-5v | %-8v | %-11v | %-5v | %-9v | %-8v |", "Name", "ID", "Reachable", "On", "Dimmable", "Temperature", "Color", "Cur. Temp", "Cur. Bri")
	for _, light := range lights {
		var temp string
		if light.supportsColorTemperature == false && light.supportsXYColor == false {
			temp = "-"
		} else {
			temp = strings.Join([]string{strconv.Itoa(light.currentLightState.colorTemperature), "K"}, "")
		}
		log.Printf("| %-20s | %3v | %-9v | %-5v | %-8v | %-11v | %-5v | %9v | %8v |", light.name, light.id, light.reachable, light.on, light.dimmable, light.supportsColorTemperature, light.supportsXYColor, temp, light.currentLightState.brightness)
	}
	return nil
}

func (self *HueBridge) Discover() error {
	locators, err := hue.DiscoverBridges(false)
	if err != nil {
		return err
	}

	self.bridgeLocator = locators[0] // use the first locator
	return nil
}

func (self *HueBridge) Register() error {
	fmt.Printf("PLEASE PUSH THE BUTTON ON THE BRIDGE .")
	for index := 0; index < 30; index++ {
		time.Sleep(5 * time.Second)
		fmt.Printf(".")
		// remember to push the button on your hue first
		bridge, err := self.bridgeLocator.CreateUser(hueBridgeAppName)
		if err != nil {
			return err
		}

		if bridge.Username != "" {
			// registration successfull
			self.username = bridge.Username
			return nil
		}
	}
	return errors.New("Registration at bridge timed out!")
}

func (self *HueBridge) Connect() error {
	if self.bridgeIP == "" {
		return errors.New("No bridge IP configured.")
	}

	if self.bridgeIP == "" {
		return errors.New("No username on bridge configured.")
	}
	self.bridge = *hue.NewBridge(self.bridgeIP, self.username)

	// Test bridge
	_, err := self.bridge.Search()
	if err != nil {
		return err
	}

	return nil
}
