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
import "net/http"
import "html/template"
import "github.com/gorilla/mux"
import "github.com/gorilla/handlers"
import "encoding/json"
import "fmt"
import "strings"
import "strconv"

func startInterface() {
	if !configuration.WebInterface.Enabled {
		return
	}

	r := mux.NewRouter()
	// html endpoints
	r.HandleFunc("/", dashboardHandler).Methods("GET")
	r.HandleFunc("/schedules.html", schedulesHandler).Methods("GET")
	r.HandleFunc("/configuration.html", configurationHandler).Methods("GET")

	// REST endpoints
	r.HandleFunc("/restart", restartHandler).Methods("PUT", "POST")
	r.HandleFunc("/schedules", updateSchedulesHandler).Methods("PUT", "POST")
	r.HandleFunc("/configuration", updateConfigurationHandler).Methods("PUT", "POST")
	r.HandleFunc("/lights", lightsHandler).Methods("GET")
	r.HandleFunc("/lights/{id}/automatic", automateLightHandler).Methods("PUT", "POST")
	r.HandleFunc("/lights/{id}/activate", activateLightHandler).Methods("PUT", "POST")

	// static files
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("gui/static"))))

	http.Handle("/", handlers.CompressHandler(r))
	port := configuration.WebInterface.Port
	log.Printf("Webinterface started on port %d", port)
	log.Warning(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}

func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	log.Debugf("Serving dashboard page to %s", r.RemoteAddr)
	if configuration.Bridge.IP == "" || configuration.Bridge.Username == "" {
		dashboardTemplate := template.Must(template.New("init.html").ParseGlob("gui/template/init.html"))
		err := dashboardTemplate.Execute(w, bridge)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		dashboardTemplate := template.Must(template.New("dashboard.html").ParseGlob("gui/template/dashboard.html"))
		err := dashboardTemplate.Execute(w, lights)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func configurationHandler(w http.ResponseWriter, r *http.Request) {
	log.Debugf("Serving configuration page to %s", r.RemoteAddr)
	configurationTemplate := template.Must(template.New("configuration.html").ParseGlob("gui/template/configuration.html"))
	err := configurationTemplate.Execute(w, configuration)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func schedulesHandler(w http.ResponseWriter, r *http.Request) {
	log.Debugf("Serving schedules page to %s", r.RemoteAddr)
	schedulesTemplate := template.Must(template.New("schedules.html").Funcs(template.FuncMap{"lightsToString": lightsToString}).ParseGlob("gui/template/schedules.html"))
	err := schedulesTemplate.Execute(w, configuration.Schedules)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func lightsToString(args ...interface{}) (string, error) {
	ok := false
	var s []int
	if len(args) == 1 {
		s, ok = args[0].([]int)
	} else {
		return "", fmt.Errorf("Input length != 1: %v", args)
	}
	if !ok {
		return "", fmt.Errorf("Not a []int: %v", args)
	}
	return strings.Trim(strings.Join(strings.Fields(fmt.Sprint(s)), ","), "[]"), nil
}

func updateSchedulesHandler(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var t []LightSchedule
	err := decoder.Decode(&t)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	log.Debugf("Received schedule update from %s: %+v", r.RemoteAddr, t)
	configuration.Schedules = t
	err = configuration.Write()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Update scenes
	updateScenes()

	// Update lights
	for _, light := range lights {
		light := light
		updateScheduleForLight(light)
	}
	w.Write([]byte("success"))
}

func updateConfigurationHandler(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var t Configuration
	err := decoder.Decode(&t)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	log.Debugf("Received configuration update from %s: %+v", r.RemoteAddr, t)
	configuration.Bridge = t.Bridge
	configuration.Location = t.Location
	configuration.WebInterface = t.WebInterface
	configuration.Write()
	log.Debugf("Updated configuration to: %+v", configuration)
	w.Write([]byte("success"))
}

func automateLightHandler(w http.ResponseWriter, r *http.Request) {
	lightID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	for _, l := range lights {
		if l.ID == lightID {
			log.Printf("💡 Light %s - Enabling automatic mode as requested by %s", l.Name, r.RemoteAddr)
			l.Tracking = false
		}
	}
	w.Write([]byte("success"))
}

func activateLightHandler(w http.ResponseWriter, r *http.Request) {
	log.Debugf("Received new light state by %s", r.RemoteAddr)
	defer r.Body.Close()
	lightID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	decoder := json.NewDecoder(r.Body)
	var t LightState
	err = decoder.Decode(&t)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !t.isValid() {
		log.Warningf("Received invalid light state from %s: %+v", r.RemoteAddr, t)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	for _, l := range lights {
		if l.ID == lightID {
			log.Printf("💡 Light %s - Activating light state %+v as requested by %s", l.Name, t, r.RemoteAddr)
			l.Automatic = false
			l.HueLight.setLightState(t.ColorTemperature, t.Brightness, 0)
		}
	}
	w.Write([]byte("success"))
}

func lightsHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Serving lights to %s", r.RemoteAddr)
	ls := []Light{}
	for _, l := range lights {
		ls = append(ls, *l)
	}
	data, err := json.Marshal(ls)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(data)
}

func restartHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Restart requested by %s", r.RemoteAddr)
	r.Body.Close()
	w.Write([]byte("success"))
	Restart()
}
