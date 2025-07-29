package main

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestNewMockServer tests the creation of a new mock server
func TestNewMockServer(t *testing.T) {
	configPath := "test-config.json"
	server := NewMockServer(configPath)

	if server == nil {
		t.Fatal("Expected server to be created, got nil")
	}

	if server.configPath != configPath {
		t.Errorf("Expected configPath to be %s, got %s", configPath, server.configPath)
	}

	if server.router == nil {
		t.Error("Expected router to be initialized")
	}

	if server.plugins == nil {
		t.Error("Expected plugins map to be initialized")
	}
}

// TestLoadConfig tests loading configuration from file
func TestLoadConfig(t *testing.T) {
	// Create temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.json")

	config := Config{
		Port:       "8080",
		PluginsDir: "test-plugins",
		Endpoints: []Endpoint{
			{
				Path:       "/test",
				Method:     "GET",
				StatusCode: 200,
				Response:   map[string]string{"message": "test"},
			},
		},
	}

	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	err = os.WriteFile(configPath, data, 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Test loading config
	server := NewMockServer(configPath)
	err = server.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if server.config.Port != "8080" {
		t.Errorf("Expected port to be 8080, got %s", server.config.Port)
	}

	if server.config.PluginsDir != "test-plugins" {
		t.Errorf("Expected pluginsDir to be test-plugins, got %s", server.config.PluginsDir)
	}

	if len(server.config.Endpoints) != 1 {
		t.Errorf("Expected 1 endpoint, got %d", len(server.config.Endpoints))
	}
}

// TestLoadConfigWithDefaults tests loading config with default values
func TestLoadConfigWithDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.json")

	// Create config without port and pluginsDir
	config := Config{
		Endpoints: []Endpoint{},
	}

	data, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	err = os.WriteFile(configPath, data, 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	server := NewMockServer(configPath)
	err = server.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if server.config.Port != "9000" {
		t.Errorf("Expected default port to be 9000, got %s", server.config.Port)
	}

	if server.config.PluginsDir != "plugins" {
		t.Errorf("Expected default pluginsDir to be plugins, got %s", server.config.PluginsDir)
	}
}

// TestLoadPlugins tests loading plugins from directory
func TestLoadPlugins(t *testing.T) {
	tmpDir := t.TempDir()
	pluginsDir := filepath.Join(tmpDir, "plugins")
	err := os.MkdirAll(pluginsDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create plugins directory: %v", err)
	}

	// Create test plugin
	plugin := Plugin{
		Name:        "test-plugin",
		Description: "Test plugin",
		Enabled:     true,
		Endpoints: []Endpoint{
			{
				Path:       "/test-plugin",
				Method:     "GET",
				StatusCode: 200,
				Response:   map[string]string{"message": "from plugin"},
			},
		},
	}

	pluginData, err := json.Marshal(plugin)
	if err != nil {
		t.Fatalf("Failed to marshal plugin: %v", err)
	}

	pluginPath := filepath.Join(pluginsDir, "test-plugin.json")
	err = os.WriteFile(pluginPath, pluginData, 0644)
	if err != nil {
		t.Fatalf("Failed to write plugin file: %v", err)
	}

	// Test loading plugins
	server := NewMockServer("")
	server.pluginsDir = pluginsDir
	err = server.LoadPlugins()
	if err != nil {
		t.Fatalf("Failed to load plugins: %v", err)
	}

	if len(server.plugins) != 1 {
		t.Errorf("Expected 1 plugin, got %d", len(server.plugins))
	}

	loadedPlugin, exists := server.plugins["test-plugin"]
	if !exists {
		t.Error("Expected test-plugin to be loaded")
	}

	if !loadedPlugin.Enabled {
		t.Error("Expected plugin to be enabled")
	}

	if len(loadedPlugin.Endpoints) != 1 {
		t.Errorf("Expected 1 endpoint in plugin, got %d", len(loadedPlugin.Endpoints))
	}
}

// TestSetupRoutes tests route setup functionality
func TestSetupRoutes(t *testing.T) {
	server := NewMockServer("")
	server.config = &Config{
		Port:       "9000",
		PluginsDir: "plugins",
		Endpoints: []Endpoint{
			{
				Path:       "/api/test",
				Method:     "GET",
				StatusCode: 200,
				Response:   map[string]string{"message": "test"},
			},
		},
	}

	server.plugins = map[string]*Plugin{
		"test-plugin": {
			Name:    "test-plugin",
			Enabled: true,
			Endpoints: []Endpoint{
				{
					Path:       "/plugin/test",
					Method:     "POST",
					StatusCode: 201,
					Response:   map[string]string{"message": "plugin"},
				},
			},
		},
	}

	server.SetupRoutes()

	// Test main config endpoint
	req := httptest.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["message"] != "test" {
		t.Errorf("Expected message 'test', got '%s'", response["message"])
	}

	// Test plugin endpoint
	req = httptest.NewRequest("POST", "/plugin/test", nil)
	w = httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	if w.Code != 201 {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	err = json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["message"] != "plugin" {
		t.Errorf("Expected message 'plugin', got '%s'", response["message"])
	}
}

// TestHealthEndpoint tests the health check endpoint
func TestHealthEndpoint(t *testing.T) {
	server := NewMockServer("")
	server.config = &Config{Port: "9000", PluginsDir: "plugins"}
	server.SetupRoutes()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["status"] != "ok" {
		t.Errorf("Expected status 'ok', got '%s'", response["status"])
	}
}

// TestAdminPluginsEndpoint tests the admin plugins listing endpoint
func TestAdminPluginsEndpoint(t *testing.T) {
	server := NewMockServer("")
	server.config = &Config{Port: "9000", PluginsDir: "plugins"}
	server.plugins = map[string]*Plugin{
		"test-plugin": {
			Name:        "test-plugin",
			Description: "Test plugin",
			Enabled:     true,
			Endpoints:   []Endpoint{},
		},
	}
	server.SetupRoutes()

	req := httptest.NewRequest("GET", "/_admin/plugins", nil)
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]*Plugin
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	plugin, exists := response["test-plugin"]
	if !exists {
		t.Error("Expected test-plugin to be in response")
	}

	if plugin.Name != "test-plugin" {
		t.Errorf("Expected plugin name 'test-plugin', got '%s'", plugin.Name)
	}
}

// TestAdminPluginToggle tests the plugin toggle functionality
func TestAdminPluginToggle(t *testing.T) {
	tmpDir := t.TempDir()
	pluginsDir := filepath.Join(tmpDir, "plugins")
	err := os.MkdirAll(pluginsDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create plugins directory: %v", err)
	}

	server := NewMockServer("")
	server.config = &Config{Port: "9000", PluginsDir: pluginsDir}
	server.pluginsDir = pluginsDir
	server.plugins = map[string]*Plugin{
		"test-plugin": {
			Name:    "test-plugin",
			Enabled: true,
		},
	}
	server.SetupRoutes()

	req := httptest.NewRequest("POST", "/_admin/plugins/test-plugin/toggle", nil)
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	enabled, ok := response["enabled"].(bool)
	if !ok {
		t.Error("Expected enabled field to be boolean")
	}

	if enabled {
		t.Error("Expected plugin to be disabled after toggle")
	}

	// Check that plugin state was updated
	if server.plugins["test-plugin"].Enabled {
		t.Error("Expected plugin to be disabled in server state")
	}
}

// TestNotFoundHandler tests the 404 handler
func TestNotFoundHandler(t *testing.T) {
	server := NewMockServer("")
	server.config = &Config{Port: "9000", PluginsDir: "plugins"}
	server.SetupRoutes()

	req := httptest.NewRequest("GET", "/nonexistent", nil)
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Errorf("Expected status 404, got %d", w.Code)
	}

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["error"] != "Endpoint not found" {
		t.Errorf("Expected error 'Endpoint not found', got '%s'", response["error"])
	}

	if response["path"] != "/nonexistent" {
		t.Errorf("Expected path '/nonexistent', got '%s'", response["path"])
	}
}

// TestEndpointWithDelay tests endpoint with delay
func TestEndpointWithDelay(t *testing.T) {
	server := NewMockServer("")
	server.config = &Config{
		Port:       "9000",
		PluginsDir: "plugins",
		Endpoints: []Endpoint{
			{
				Path:       "/delayed",
				Method:     "GET",
				StatusCode: 200,
				Response:   map[string]string{"message": "delayed"},
				Delay:      100, // 100ms delay
			},
		},
	}
	server.SetupRoutes()

	start := time.Now()
	req := httptest.NewRequest("GET", "/delayed", nil)
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)
	elapsed := time.Since(start)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check that delay was applied (allow some tolerance)
	if elapsed < 90*time.Millisecond {
		t.Errorf("Expected delay of at least 90ms, got %v", elapsed)
	}
}

// TestEndpointWithCustomHeaders tests endpoint with custom headers
func TestEndpointWithCustomHeaders(t *testing.T) {
	server := NewMockServer("")
	server.config = &Config{
		Port:       "9000",
		PluginsDir: "plugins",
		Endpoints: []Endpoint{
			{
				Path:       "/custom-headers",
				Method:     "GET",
				StatusCode: 200,
				Response:   map[string]string{"message": "test"},
				Headers: map[string]string{
					"X-Custom-Header": "custom-value",
					"X-Another":       "another-value",
				},
			},
		},
	}
	server.SetupRoutes()

	req := httptest.NewRequest("GET", "/custom-headers", nil)
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if w.Header().Get("X-Custom-Header") != "custom-value" {
		t.Errorf("Expected X-Custom-Header to be 'custom-value', got '%s'", w.Header().Get("X-Custom-Header"))
	}

	if w.Header().Get("X-Another") != "another-value" {
		t.Errorf("Expected X-Another to be 'another-value', got '%s'", w.Header().Get("X-Another"))
	}
}

// TestParseHeaders tests header parsing functionality
func TestParseHeaders(t *testing.T) {
	tests := []struct {
		input    string
		expected map[string]string
	}{
		{
			input:    "",
			expected: map[string]string{},
		},
		{
			input: "Content-Type:application/json",
			expected: map[string]string{
				"Content-Type": "application/json",
			},
		},
		{
			input: "Content-Type:application/json,X-Custom:value",
			expected: map[string]string{
				"Content-Type": "application/json",
				"X-Custom":     "value",
			},
		},
		{
			input: "Content-Type: application/json, X-Custom: value",
			expected: map[string]string{
				"Content-Type": "application/json",
				"X-Custom":     "value",
			},
		},
	}

	for _, test := range tests {
		result := parseHeaders(test.input)
		if len(result) != len(test.expected) {
			t.Errorf("Expected %d headers, got %d for input '%s'", len(test.expected), len(result), test.input)
			continue
		}

		for key, expectedValue := range test.expected {
			if result[key] != expectedValue {
				t.Errorf("Expected header '%s' to be '%s', got '%s' for input '%s'", key, expectedValue, result[key], test.input)
			}
		}
	}
}

// TestParseResponse tests response parsing functionality
func TestParseResponse(t *testing.T) {
	tests := []struct {
		input    string
		expected interface{}
	}{
		{
			input:    `{"message": "test"}`,
			expected: map[string]interface{}{"message": "test"},
		},
		{
			input:    `[1, 2, 3]`,
			expected: []interface{}{float64(1), float64(2), float64(3)},
		},
		{
			input:    `"simple string"`,
			expected: "simple string",
		},
		{
			input:    `plain text`,
			expected: "plain text",
		},
	}

	for _, test := range tests {
		result := parseResponse(test.input)

		// For JSON objects and arrays, we need to compare differently
		switch expected := test.expected.(type) {
		case map[string]interface{}:
			resultMap, ok := result.(map[string]interface{})
			if !ok {
				t.Errorf("Expected map for input '%s', got %T", test.input, result)
				continue
			}
			for key, value := range expected {
				if resultMap[key] != value {
					t.Errorf("Expected map key '%s' to be %v, got %v for input '%s'", key, value, resultMap[key], test.input)
				}
			}
		case []interface{}:
			resultArray, ok := result.([]interface{})
			if !ok {
				t.Errorf("Expected array for input '%s', got %T", test.input, result)
				continue
			}
			if len(resultArray) != len(expected) {
				t.Errorf("Expected array length %d, got %d for input '%s'", len(expected), len(resultArray), test.input)
				continue
			}
			for i, value := range expected {
				if resultArray[i] != value {
					t.Errorf("Expected array index %d to be %v, got %v for input '%s'", i, value, resultArray[i], test.input)
				}
			}
		default:
			if result != expected {
				t.Errorf("Expected %v, got %v for input '%s'", expected, result, test.input)
			}
		}
	}
}

// TestAddEndpointToConfig tests adding endpoint to configuration
func TestAddEndpointToConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.json")

	cmdEndpoint := &CommandLineEndpoint{
		Path:       "/api/test",
		Method:     "POST",
		StatusCode: 201,
		Response:   `{"message": "created"}`,
		Headers:    "Content-Type:application/json",
		Delay:      100,
	}

	err := AddEndpointToConfig(configPath, cmdEndpoint)
	if err != nil {
		t.Fatalf("Failed to add endpoint to config: %v", err)
	}

	// Verify the config was created correctly
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	if len(config.Endpoints) != 1 {
		t.Errorf("Expected 1 endpoint, got %d", len(config.Endpoints))
	}

	endpoint := config.Endpoints[0]
	if endpoint.Path != "/api/test" {
		t.Errorf("Expected path '/api/test', got '%s'", endpoint.Path)
	}

	if endpoint.Method != "POST" {
		t.Errorf("Expected method 'POST', got '%s'", endpoint.Method)
	}

	if endpoint.StatusCode != 201 {
		t.Errorf("Expected status code 201, got %d", endpoint.StatusCode)
	}

	if endpoint.Delay != 100 {
		t.Errorf("Expected delay 100, got %d", endpoint.Delay)
	}
}

// TestStringResponse tests endpoint with string response
func TestStringResponse(t *testing.T) {
	server := NewMockServer("")
	server.config = &Config{
		Port:       "9000",
		PluginsDir: "plugins",
		Endpoints: []Endpoint{
			{
				Path:       "/string-response",
				Method:     "GET",
				StatusCode: 200,
				Response:   "Hello, World!",
			},
		},
	}
	server.SetupRoutes()

	req := httptest.NewRequest("GET", "/string-response", nil)
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	body := strings.TrimSpace(w.Body.String())
	if body != "Hello, World!" {
		t.Errorf("Expected response 'Hello, World!', got '%s'", body)
	}
}
