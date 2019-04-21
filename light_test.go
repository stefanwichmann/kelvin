package main

import (
	"errors"
	"io/ioutil"
	"testing"
	"time"

	log "github.com/Sirupsen/logrus"
)

func mockSetLightState(int, int, time.Duration) error {
	return nil
}

func mockSetLightStateReturnsError(int, int, time.Duration) error {
	return errors.New("Test Error")
}

func mockHasChanged() bool {
	return true
}

func mockHasNotChanged() bool {
	return false
}

func mockHasState(int, int) bool {
	return true
}

func mockHasNoState(int, int) bool {
	return false
}

func TestLightUpdateFalseReturns(t *testing.T) {
	log.SetOutput(ioutil.Discard)
	outOfTrackLights := map[string]Light{
		"Not scheduled": Light{
			Scheduled:  false,
			hasChanged: mockHasChanged,
		},
		"Scheduled and not reachable": Light{
			Scheduled:  true,
			Reachable:  false,
			Tracking:   true,
			hasChanged: mockHasChanged,
		},
		"Not tracking": Light{
			Scheduled:  true,
			Reachable:  false,
			Tracking:   false,
			hasChanged: mockHasChanged,
		},
		"Off light": Light{
			Scheduled:  true,
			Reachable:  true,
			Tracking:   true,
			On:         false,
			hasChanged: mockHasChanged,
		},
		"Off and not tracking": Light{
			Scheduled:  true,
			Reachable:  true,
			Tracking:   false,
			On:         false,
			hasChanged: mockHasChanged,
		},
		"Automatic, not initializing": Light{
			Scheduled:  true,
			Reachable:  true,
			Tracking:   true,
			On:         true,
			Automatic:  true,
			hasChanged: mockHasChanged,
		},
		"Not Automatic, no state": Light{
			Scheduled:  true,
			Reachable:  true,
			Tracking:   true,
			On:         true,
			Automatic:  false,
			hasState:   mockHasNoState,
			hasChanged: mockHasChanged,
		},
		"Automatic, not changed, with state": Light{
			Scheduled:  true,
			Reachable:  true,
			Tracking:   true,
			On:         true,
			Automatic:  true,
			hasChanged: mockHasNotChanged,
			hasState:   mockHasState,
		},
		"Lightstate minus one": Light{
			Scheduled: true,
			Reachable: true,
			Tracking:  true,
			On:        true,
			TargetLightState: LightState{
				ColorTemperature: -1,
				Brightness:       -1,
			},
		},

		//		if light.TargetLightState.ColorTemperature == -1 && light.TargetLightState.Brightness == -1 {

	}

	transistionTime := time.Duration(60.0)
	for description, l := range outOfTrackLights {
		l.setLightState = mockSetLightState
		if ok, _ := l.update(transistionTime); ok != false {
			t.Fatalf("Light update should have returned false : %v", description)
		}
	}
}

func TestLightUpdateTrueReturns(t *testing.T) {
	log.SetOutput(ioutil.Discard)
	transistionTime := time.Duration(1.0)

	okLights := map[string]Light{
		"Not tracked, light on": Light{
			Scheduled:        true,
			Reachable:        true,
			Tracking:         false,
			On:               true,
			Schedule:         Schedule{enableWhenLightsAppear: true},
			TargetLightState: LightState{ColorTemperature: 2000, Brightness: 60},
			hasState:         mockHasState,
			hasChanged:       mockHasChanged,
		},
		"Not Automatic": Light{
			Scheduled:  true,
			Reachable:  true,
			Tracking:   true,
			On:         true,
			Automatic:  false,
			hasState:   mockHasState,
			hasChanged: mockHasChanged,
		},
		"Automatic, initializing": Light{
			Scheduled:    true,
			Reachable:    true,
			Tracking:     true,
			On:           true,
			Automatic:    true,
			Initializing: true,
			hasChanged:   mockHasChanged,
		},
		"Automatic, not changed, no state": Light{
			Scheduled:  true,
			Reachable:  true,
			Tracking:   true,
			On:         true,
			Automatic:  true,
			hasChanged: mockHasNotChanged,
			hasState:   mockHasNoState,
		},
	}

	for description, l := range okLights {
		l.setLightState = mockSetLightState

		if ok, _ := l.update(transistionTime); ok != true {
			t.Fatalf("Light update should have returned true: %v", description)
		}
	}
}

func TestLightUpdateErrorReturns(t *testing.T) {
	log.SetOutput(ioutil.Discard)
	transistionTime := time.Duration(1.0)

	errorLights := map[string]Light{
		"On, setLightState returns an error": Light{
			Scheduled:     true,
			Reachable:     true,
			Tracking:      true,
			On:            true,
			hasState:      mockHasState,
			setLightState: mockSetLightStateReturnsError,
		},
		"Automatic, initializing, when setLightState returns an error": Light{
			Scheduled:     true,
			Reachable:     true,
			Tracking:      true,
			On:            true,
			Initializing:  true,
			Automatic:     true,
			hasChanged:    mockHasChanged,
			hasState:      mockHasState,
			setLightState: mockSetLightStateReturnsError,
		},
		"Automatic, not changed, no state, when setLightState returns an error": Light{
			Scheduled:     true,
			Reachable:     true,
			Tracking:      true,
			On:            true,
			Automatic:     true,
			hasChanged:    mockHasNotChanged,
			hasState:      mockHasNoState,
			setLightState: mockSetLightStateReturnsError,
		},
	}

	for description, l := range errorLights {
		if _, err := l.update(transistionTime); err == nil {
			t.Fatalf("Light update should have returned an error: %v", description)
		}
	}
}
