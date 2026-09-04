package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// withTestConfiguration swaps the global configuration for the test and
// restores it afterwards. The web handlers operate on the global.
func withTestConfiguration(t *testing.T, c Configuration) {
	t.Helper()
	previous := configuration
	configuration = &c
	t.Cleanup(func() { configuration = previous })
}

func protectedRequest(target string, mutate ...func(*http.Request)) *httptest.ResponseRecorder {
	handler := protect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	request := httptest.NewRequest("GET", target, nil)
	for _, m := range mutate {
		m(request)
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func TestProtectRequiresAuthentication(t *testing.T) {
	withTestConfiguration(t, Configuration{WebInterface: WebInterface{Password: "secret"}})

	if code := protectedRequest("http://192.168.1.2:8080/").Code; code != http.StatusUnauthorized {
		t.Errorf("request without credentials: got %d, want 401", code)
	}
	wrong := protectedRequest("http://192.168.1.2:8080/", func(r *http.Request) { r.SetBasicAuth("", "wrong") })
	if wrong.Code != http.StatusUnauthorized {
		t.Errorf("request with wrong password: got %d, want 401", wrong.Code)
	}
	right := protectedRequest("http://192.168.1.2:8080/", func(r *http.Request) { r.SetBasicAuth("", "secret") })
	if right.Code != http.StatusOK {
		t.Errorf("request with correct password: got %d, want 200", right.Code)
	}
}

func TestProtectFailsClosedWithoutPassword(t *testing.T) {
	withTestConfiguration(t, Configuration{WebInterface: WebInterface{Password: ""}})

	empty := protectedRequest("http://192.168.1.2:8080/", func(r *http.Request) { r.SetBasicAuth("", "") })
	if empty.Code != http.StatusUnauthorized {
		t.Errorf("empty configured password must deny all requests: got %d, want 401", empty.Code)
	}
}

func TestProtectRejectsForeignOrigin(t *testing.T) {
	withTestConfiguration(t, Configuration{WebInterface: WebInterface{Password: "secret"}})

	foreign := protectedRequest("http://192.168.1.2:8080/", func(r *http.Request) {
		r.SetBasicAuth("", "secret")
		r.Header.Set("Origin", "http://evil.example")
	})
	if foreign.Code != http.StatusForbidden {
		t.Errorf("foreign origin: got %d, want 403", foreign.Code)
	}
	same := protectedRequest("http://192.168.1.2:8080/", func(r *http.Request) {
		r.SetBasicAuth("", "secret")
		r.Header.Set("Origin", "http://192.168.1.2:8080")
	})
	if same.Code != http.StatusOK {
		t.Errorf("same origin: got %d, want 200", same.Code)
	}
}

func TestProtectRejectsDNSHostNames(t *testing.T) {
	withTestConfiguration(t, Configuration{WebInterface: WebInterface{Password: "secret", ListenAddress: "kelvin.home"}})
	auth := func(r *http.Request) { r.SetBasicAuth("", "secret") }

	rebind := protectedRequest("http://attacker.example:8080/", auth)
	if rebind.Code != http.StatusForbidden {
		t.Errorf("DNS name host: got %d, want 403", rebind.Code)
	}
	for _, host := range []string{"http://localhost:8080/", "http://localhost.:8080/", "http://192.168.1.2:8080/", "http://[::1]:8080/", "http://kelvin.home:8080/", "http://raspberrypi.local:8080/"} {
		if code := protectedRequest(host, auth).Code; code != http.StatusOK {
			t.Errorf("host %s: got %d, want 200", host, code)
		}
	}
}

// TestProtectAllowsSingleLabelHostNames pins the carve-out of issue #137:
// no dotless public name is attacker-registrable (TLD apexes belong to
// registries), so rebinding cannot be driven onto a single-label host,
// and basic auth backstops the residual hostile-search-domain case.
// Multi-label names stay rejected.
func TestProtectAllowsSingleLabelHostNames(t *testing.T) {
	withTestConfiguration(t, Configuration{WebInterface: WebInterface{Password: "secret"}})
	auth := func(r *http.Request) { r.SetBasicAuth("", "secret") }

	for _, host := range []string{"http://homeserver:8080/", "http://homeserver.:8080/", "http://nas/"} {
		if code := protectedRequest(host, auth).Code; code != http.StatusOK {
			t.Errorf("single-label host %s: got %d, want 200", host, code)
		}
	}
	if code := protectedRequest("http://attacker.example:8080/", auth).Code; code != http.StatusForbidden {
		t.Errorf("multi-label host must stay rejected: got %d, want 403", code)
	}
}

// TestEnsureWebInterfacePassword pins the synchronous password setup: the
// secret is generated, persisted with owner-only permissions, and never
// regenerated once present (issue #128 review).
func TestEnsureWebInterfacePassword(t *testing.T) {
	c := Configuration{WebInterface: WebInterface{Enabled: true, Port: 8080}}
	c.ConfigurationFile = filepath.Join(t.TempDir(), "config.json")
	withTestConfiguration(t, c)

	ensureWebInterfacePassword()
	first := configuration.WebInterface.Password
	if first == "" {
		t.Fatal("no password generated for enabled interface")
	}
	info, err := os.Stat(configuration.ConfigurationFile)
	if err != nil {
		t.Fatalf("configuration not persisted: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("configuration with secrets has mode %v, want 0600", info.Mode().Perm())
	}
	ensureWebInterfacePassword()
	if configuration.WebInterface.Password != first {
		t.Errorf("existing password was regenerated")
	}
}

// TestListenAddressOverrideIsEphemeral pins the flag contract: the
// override changes the effective address but never enters the stored
// configuration (issue #128 review).
func TestListenAddressOverrideIsEphemeral(t *testing.T) {
	c := Configuration{WebInterface: WebInterface{Enabled: true, Port: 8080, ListenAddress: "127.0.0.1", Password: "secret"}}
	c.ConfigurationFile = filepath.Join(t.TempDir(), "config.json")
	withTestConfiguration(t, c)
	listenAddressOverride = "0.0.0.0"
	t.Cleanup(func() { listenAddressOverride = "" })

	if effectiveListenAddress() != "0.0.0.0" {
		t.Errorf("override not effective: %q", effectiveListenAddress())
	}
	if err := configuration.Write(); err != nil {
		t.Fatalf("could not write configuration: %v", err)
	}
	data, err := os.ReadFile(configuration.ConfigurationFile)
	if err != nil {
		t.Fatalf("could not read configuration: %v", err)
	}
	if strings.Contains(string(data), "0.0.0.0") {
		t.Errorf("run-time override was persisted to the configuration file")
	}
}

func TestProtectAllowsHealthUnauthenticated(t *testing.T) {
	withTestConfiguration(t, Configuration{WebInterface: WebInterface{Password: "secret"}})

	if code := protectedRequest("http://192.168.1.2:8080/health").Code; code != http.StatusOK {
		t.Errorf("health endpoint must not require authentication: got %d, want 200", code)
	}
}

func TestUpdateConfigurationKeepsMaskedAndOmittedSecrets(t *testing.T) {
	c := Configuration{
		Bridge:       Bridge{IP: "192.168.1.10", Username: "realusername"},
		WebInterface: WebInterface{Enabled: true, Port: 8080, ListenAddress: "0.0.0.0", Password: "secret"},
	}
	c.ConfigurationFile = filepath.Join(t.TempDir(), "config.json")
	withTestConfiguration(t, c)

	body := `{"bridge":{"ip":"192.168.1.20","username":"` + maskedValue + `"},"location":{"latitude":1,"longitude":2},"webinterface":{"enabled":true,"port":8080}}`
	request := httptest.NewRequest("PUT", "http://localhost:8080/configuration", strings.NewReader(body))
	recorder := httptest.NewRecorder()
	updateConfigurationHandler(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("update failed: got %d, body %s", recorder.Code, recorder.Body.String())
	}
	if configuration.Bridge.Username != "realusername" {
		t.Errorf("masked username overwrote the real one: %q", configuration.Bridge.Username)
	}
	if configuration.Bridge.IP != "192.168.1.20" {
		t.Errorf("bridge IP not updated: %q", configuration.Bridge.IP)
	}
	if configuration.WebInterface.Password != "secret" {
		t.Errorf("omitted password was wiped: %q", configuration.WebInterface.Password)
	}
	if configuration.WebInterface.ListenAddress != "0.0.0.0" {
		t.Errorf("omitted listen address was wiped: %q", configuration.WebInterface.ListenAddress)
	}
}

func TestConfigurationPageMasksUsername(t *testing.T) {
	withTestConfiguration(t, Configuration{
		Bridge:       Bridge{IP: "192.168.1.10", Username: "realusername"},
		WebInterface: WebInterface{Enabled: true, Port: 8080, Password: "secret"},
	})

	request := httptest.NewRequest("GET", "http://localhost:8080/configuration.html", nil)
	recorder := httptest.NewRecorder()
	configurationHandler(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("configuration page failed: got %d", recorder.Code)
	}
	page := recorder.Body.String()
	if strings.Contains(page, "realusername") {
		t.Errorf("configuration page leaks the bridge username")
	}
	if strings.Contains(page, "secret") {
		t.Errorf("configuration page leaks the web interface password")
	}
}
