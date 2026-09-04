// MIT License
//
// # Copyright (c) 2019 Stefan Wichmann
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
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

// maskedValue replaces secrets in rendered pages. The update handler
// treats a submitted masked value as "unchanged".
const maskedValue = "********"

// listenAddressOverride rebinds the interface for this run only. It is a
// separate variable, never written into the configuration, so the flag
// stays ephemeral (issue #128 review).
var listenAddressOverride string

// effectiveListenAddress resolves the address the interface binds to and
// accepts as a host name: the run-time override, the configured address,
// or loopback.
func effectiveListenAddress() string {
	if listenAddressOverride != "" {
		return listenAddressOverride
	}
	if configuration.WebInterface.ListenAddress != "" {
		return configuration.WebInterface.ListenAddress
	}
	return "127.0.0.1"
}

// ensureWebInterfacePassword generates and stores the password before the
// interface starts. It runs synchronously in main so its configuration
// write never races the update loop's writes (issue #128 review). On any
// failure the password stays empty and the interface denies all requests.
func ensureWebInterfacePassword() {
	if !configuration.WebInterface.Enabled || configuration.WebInterface.Password != "" {
		return
	}
	password, err := generatePassword()
	if err != nil {
		log.Warningf("Could not generate web interface password: %v. The interface will deny all requests.", err)
		return
	}
	configuration.WebInterface.Password = password
	if err := configuration.Write(); err != nil {
		log.Warningf("Could not store web interface password: %v. The interface will deny all requests.", err)
		configuration.WebInterface.Password = ""
		return
	}
	log.Printf("Web interface password generated: %s (any username; stored in %s)", password, configuration.ConfigurationFile)
}

// protect wraps every route in HTTP basic auth plus a Host and Origin
// check. The Host check rejects DNS names (except localhost and the
// configured listen address), which defeats DNS rebinding; the Origin
// check rejects cross-site browser requests, which defeats CSRF — the
// browser attaches cached basic auth credentials on its own, so auth
// alone does not stop either attack. The health endpoint stays open for
// container probes.
func protect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}
		if !hostAllowed(r.Host) {
			log.Warningf("Rejected request with host header %q from %s. Access the interface by IP address or a .local name.", r.Host, r.RemoteAddr)
			http.Error(w, "unknown host", http.StatusForbidden)
			return
		}
		if origin := r.Header.Get("Origin"); origin != "" && !originAllowed(origin, r.Host) {
			log.Warningf("Rejected cross-origin request from %s (origin %q)", r.RemoteAddr, origin)
			http.Error(w, "cross-origin request rejected", http.StatusForbidden)
			return
		}
		// An empty configured password denies everything: the interface
		// must never fall open because a config edit removed the secret.
		_, password, ok := r.BasicAuth()
		if configuration.WebInterface.Password == "" || !ok ||
			subtle.ConstantTimeCompare([]byte(password), []byte(configuration.WebInterface.Password)) != 1 {
			w.Header().Set("WWW-Authenticate", `Basic realm="kelvin"`)
			http.Error(w, "authentication required", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func hostAllowed(hostport string) bool {
	host := hostport
	if h, _, err := net.SplitHostPort(hostport); err == nil {
		host = h
	}
	host = strings.TrimSuffix(host, ".")
	if strings.EqualFold(host, "localhost") || net.ParseIP(strings.Trim(host, "[]")) != nil {
		return true
	}
	// mDNS names resolve only on the local network; public DNS cannot
	// rebind them, so they are safe to serve.
	if strings.HasSuffix(strings.ToLower(host), ".local") {
		return true
	}
	// Dotless names can reach public DNS (bare TLD apexes, resolver
	// search-domain expansion), but rebinding needs a name the attacker
	// controls authoritatively and no dotless public name is registrable,
	// so rebinding cannot be driven onto this branch. Basic auth
	// backstops the residual hostile-search-domain case (issue #137).
	if host != "" && !strings.Contains(host, ".") {
		return true
	}
	return strings.EqualFold(host, effectiveListenAddress())
}

func originAllowed(origin, requestHost string) bool {
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return strings.EqualFold(parsed.Host, requestHost)
}

func startInterface() {
	if !configuration.WebInterface.Enabled {
		return
	}

	r := mux.NewRouter()
	// html endpoints
	r.HandleFunc("/", dashboardHandler).Methods("HEAD", "GET")
	r.HandleFunc("/schedules.html", schedulesHandler).Methods("GET")
	r.HandleFunc("/configuration.html", configurationHandler).Methods("GET")

	// REST endpoints
	r.HandleFunc("/restart", restartHandler).Methods("PUT", "POST")
	r.HandleFunc("/schedules", updateSchedulesHandler).Methods("PUT", "POST")
	r.HandleFunc("/configuration", updateConfigurationHandler).Methods("PUT", "POST")
	r.HandleFunc("/lights", lightsHandler).Methods("GET")
	r.HandleFunc("/lights/{id}/automatic", automateLightHandler).Methods("PUT", "POST")
	r.HandleFunc("/lights/{id}/activate", activateLightHandler).Methods("PUT", "POST")
	r.HandleFunc("/health", healthHandler).Methods("HEAD", "GET")

	// static files
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("gui/static"))))

	addr := net.JoinHostPort(effectiveListenAddress(), strconv.Itoa(configuration.WebInterface.Port))
	log.Printf("Webinterface started on %s", addr)
	log.Warning(http.ListenAndServe(addr, protect(handlers.CompressHandler(r))))
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
	// Render a copy with masked secrets: the bridge username is a bearer
	// credential for the whole Hue system (issue #128).
	view := *configuration
	if view.Bridge.Username != "" {
		view.Bridge.Username = maskedValue
	}
	view.WebInterface.Password = maskedValue
	err := configurationTemplate.Execute(w, view)
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

func healthHandler(w http.ResponseWriter, r *http.Request) {
	log.Debugf("Health request with method %s from %s", r.Method, r.RemoteAddr)
	// Assume Kelvin is healthy when is is running for now
	w.WriteHeader(200)
}

func lightsToString(args ...interface{}) (string, error) {
	ok := false
	var s []int
	if len(args) == 1 {
		s, ok = args[0].([]int)
	} else {
		return "", fmt.Errorf("input length != 1: %v", args)
	}
	if !ok {
		return "", fmt.Errorf("not a []int: %v", args)
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
	log.Debugf("Received configuration update from %s", r.RemoteAddr)
	// The form posts secrets masked or omitted; keep the stored values then.
	if t.Bridge.Username == "" || t.Bridge.Username == maskedValue {
		t.Bridge.Username = configuration.Bridge.Username
	}
	if t.WebInterface.ListenAddress == "" {
		t.WebInterface.ListenAddress = configuration.WebInterface.ListenAddress
	}
	if t.WebInterface.Password == "" || t.WebInterface.Password == maskedValue {
		t.WebInterface.Password = configuration.WebInterface.Password
	}
	configuration.Bridge = t.Bridge
	configuration.Location = t.Location
	configuration.WebInterface = t.WebInterface
	if err := configuration.Write(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
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
			w.Write([]byte("success"))
			return
		}
	}
	http.Error(w, "unknown light", http.StatusNotFound)
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
		http.Error(w, "Received invalid light state", http.StatusBadRequest)
		return
	}

	for _, l := range lights {
		if l.ID == lightID {
			log.Printf("💡 Light %s - Activating light state %+v as requested by %s", l.Name, t, r.RemoteAddr)
			l.Automatic = false
			if err := l.HueLight.setLightState(t.ColorTemperature, t.Brightness, 0); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Write([]byte("success"))
			return
		}
	}
	http.Error(w, "unknown light", http.StatusNotFound)
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
