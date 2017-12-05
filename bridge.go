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
	"io/ioutil"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

// HueBridge represents the Philips Hue bridge in
// your system.
// It is used to communicate with all devices.
type HueBridge struct {
	bridge   hue.Bridge
	BridgeIP string
	Username string
	Version  int
}

const hueBridgeAppName = "kelvin"

// InitializeBridge creates and returns an initialized HueBridge.
// If you have a valid configuration this will be used. Otherwise a local
// discovery will be started, followed by a user registration on your bridge.
func (bridge *HueBridge) InitializeBridge(configuration *Configuration) error {
	err := bridge.discover(configuration.Bridge.IP)
	if err != nil {
		return err
	}
	configuration.Bridge.IP = bridge.BridgeIP

	if configuration.Bridge.Username != "" {
		log.Debugf("⌘ Found bridge username in configuration: %s", configuration.Bridge.Username)
		bridge.Username = configuration.Bridge.Username
	} else {
		log.Debugf("⌘ No username found in bridge configuration. Starting registration...")
		err := bridge.register()
		if err != nil {
			return err
		}
		log.Debugf("⌘ Saving new username in bridge configuration: %s", bridge.Username)
		configuration.Bridge.Username = bridge.Username
	}

	log.Debugf("⌘ Connecting to bridge %s with username %s", bridge.BridgeIP, bridge.Username)
	err = bridge.connect()
	if err != nil {
		return err
	}
	log.Println("⌘ Connection to bridge established")
	go bridge.validateSofwareVersion()
	err = bridge.printDevices()
	if err != nil {
		return err
	}

	err = bridge.populateSchedule(configuration)
	return err
}

// Lights return all known lights on your bridge.
func (bridge *HueBridge) Lights() ([]*Light, error) {
	var lights []*Light
	hueLights, err := bridge.bridge.GetAllLights()
	if err != nil {
		return lights, err
	}

	for _, hueLight := range hueLights {
		var light Light
		light.ID, err = strconv.Atoi(hueLight.Id)
		if err != nil {
			return lights, err
		}

		light.HueLight = *hueLight
		light.initialize()

		lights = append(lights, &light)
	}

	sort.Slice(lights, func(i, j int) bool { return lights[i].ID < lights[j].ID })
	return lights, nil
}

func (bridge *HueBridge) printDevices() error {
	lights, err := bridge.Lights()
	if err != nil {
		return err
	}

	log.Printf("⌘ Devices found on current bridge:")
	log.Printf("| %-20s | %3v | %-9v | %-5v | %-8v | %-11v | %-5v | %-9v | %-8v |", "Name", "ID", "Reachable", "On", "Dimmable", "Temperature", "Color", "Cur. Temp", "Cur. Bri")
	for _, light := range lights {
		var temp string
		if !light.SupportsColorTemperature && !light.SupportsXYColor {
			temp = "-"
		} else {
			temp = strings.Join([]string{strconv.Itoa(light.CurrentLightState.ColorTemperature), "K"}, "")
		}
		log.Printf("| %-20s | %3v | %-9v | %-5v | %-8v | %-11v | %-5v | %9v | %8v |", light.Name, light.ID, light.Reachable, light.On, light.Dimmable, light.SupportsColorTemperature, light.SupportsXYColor, temp, light.CurrentLightState.Brightness)
	}
	return nil
}

func (bridge *HueBridge) discover(ip string) error {
	if ip != "" {
		// we have a known IP address. Validate if it points to a reachable bridge
		bridge.BridgeIP = ip
		err := bridge.validateBridge()
		if err == nil {
			return nil
		}
	}
	log.Debugf("⌘ Starting bridge discovery")
	bridges, err := hue.DiscoverBridges(false)
	if err != nil {
		bridge.BridgeIP = ""
		return err
	}
	if len(bridges) == 0 {
		bridge.BridgeIP = ""
		return errors.New("Bridge discovery failed. Please configure manually in config.json.")
	}
	for _, candidate := range bridges {
		bridge.BridgeIP = candidate.IpAddr
		err := bridge.validateBridge()
		if err == nil {
			log.Printf("⌘ Found bridge at %s", bridge.BridgeIP)
			return nil
		}
	}
	bridge.BridgeIP = ""
	return errors.New("Bridge discovery failed. Please configure manually in config.json.")
}

func (bridge *HueBridge) register() error {
	if bridge.BridgeIP == "" {
		return errors.New("Registration at bridge not possible because no IP is configured. Start discovery first or enter manually.")
	}

	bridge.bridge = *hue.NewBridge(bridge.BridgeIP, "")
	log.Printf("⌘ Starting user registration.")
	log.Warningf("⌘ PLEASE PUSH THE BLUE BUTTON ON YOUR HUE BRIDGE")
	for {
		time.Sleep(5 * time.Second)

		// try user creation, will fail if the button wasn't pressed.
		err := bridge.bridge.CreateUser(hueBridgeAppName)
		if err != nil {
			return err
		}

		if bridge.bridge.Username != "" {
			// registration successful
			bridge.Username = bridge.bridge.Username
			log.Printf("⌘ User registration successful.")
			return nil
		}
	}
}

func (bridge *HueBridge) connect() error {
	if bridge.BridgeIP == "" {
		return errors.New("No bridge IP configured")
	}

	if bridge.Username == "" {
		return errors.New("No username on bridge configured")
	}
	bridge.bridge = *hue.NewBridge(bridge.BridgeIP, bridge.Username)

	// Test bridge
	_, err := bridge.bridge.Search()
	return err
}

func (bridge *HueBridge) populateSchedule(configuration *Configuration) error {
	if len(configuration.Schedules) == 0 {
		return errors.New("Configuration does not contain any schedules to populate")
	}

	// Do we have associated lights?
	for _, schedule := range configuration.Schedules {
		if len(schedule.AssociatedDeviceIDs) > 0 {
			log.Debugf("⌘ Configuration contains at least one schedule with associated lights.")
			return nil // At least one schedule is configured
		}
	}

	// No schedule has associated lights
	log.Debugf("⌘ Configuration contains no schedule with associated lights. Initializing first schedule with all lights.")
	lights, err := bridge.Lights()
	if err != nil {
		return err
	}
	var lightIDs []int
	for _, light := range lights {
		lightIDs = append(lightIDs, light.ID)
	}
	configuration.Schedules[0].AssociatedDeviceIDs = lightIDs
	return nil
}

func (bridge *HueBridge) validateSofwareVersion() {
	configuration, err := bridge.bridge.Configuration()
	if err != nil {
		log.Warningf("⌘ Could not validate bridge software version: %v", err)
		return
	}

	swversion, err := strconv.Atoi(configuration.SoftwareVersion)
	if err != nil {
		log.Warningf("⌘ Could not validate bridge software version: %v", err)
		return
	}
	log.Debugf("⌘ Bridge is running software version %s", configuration.SoftwareVersion)

	if (bridge.Version == 1 && swversion < 1038802) || (bridge.Version == 2 && swversion < 1711151408) {
		log.Warningf("⌘ Your hue bridge is running an old software version. Please update using the hue app to ensure Kelvin will run smoothly.")
	} else {
		log.Debugf("⌘ Bridge software is up to date")
	}
}

func (bridge *HueBridge) validateBridge() error {
	if bridge.BridgeIP == "" {
		return errors.New("No bridge configured. Could not validate")
	}
	resp, err := http.Get("http://" + bridge.BridgeIP + "/description.xml")
	if err != nil {
		return fmt.Errorf("Could not read bridge description: %v", err)
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("Could not read bridge description: %v", err)
	}
	if strings.Contains(string(data), "<modelNumber>929000226503</modelNumber>") {
		bridge.Version = 1
		return nil
	}
	if strings.Contains(string(data), "<modelNumber>BSB002</modelNumber>") {
		bridge.Version = 2
		return nil
	}
	return fmt.Errorf("Bridge validation failed.")
}
