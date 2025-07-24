package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/mux"
)

// Endpoint represents a mock API endpoint configuration
type Endpoint struct {
	Path       string            `json:"path"`
	Method     string            `json:"method"`
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers,omitempty"`
	Response   interface{}       `json:"response"`
	Delay      int               `json:"delay,omitempty"` // delay in milliseconds
}

// Plugin represents a plugin configuration
type Plugin struct {
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	Enabled     bool       `json:"enabled"`
	Endpoints   []Endpoint `json:"endpoints"`
}

// Config represents the entire mock server configuration
type Config struct {
	Port       string     `json:"port,omitempty"`
	PluginsDir string     `json:"plugins_dir,omitempty"`
	Endpoints  []Endpoint `json:"endpoints"`
}

// MockServer represents the mock server
type MockServer struct {
	router     *mux.Router
	config     *Config
	plugins    map[string]*Plugin
	configPath string
	pluginsDir string
	mutex      sync.RWMutex
	watcher    *fsnotify.Watcher
}

// NewMockServer creates a new mock server instance
func NewMockServer(configPath string) *MockServer {
	return &MockServer{
		router:     mux.NewRouter(),
		plugins:    make(map[string]*Plugin),
		configPath: configPath,
	}
}

// LoadPlugins loads all plugins from the plugins directory
func (ms *MockServer) LoadPlugins() error {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()

	// Clear existing plugins
	ms.plugins = make(map[string]*Plugin)

	// Check if plugins directory exists
	if _, err := os.Stat(ms.pluginsDir); os.IsNotExist(err) {
		log.Printf("Plugins directory %s does not exist, skipping plugin loading", ms.pluginsDir)
		return nil
	}

	files, err := ioutil.ReadDir(ms.pluginsDir)
	if err != nil {
		return fmt.Errorf("failed to read plugins directory: %v", err)
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") {
			pluginPath := filepath.Join(ms.pluginsDir, file.Name())
			if err := ms.loadSinglePlugin(pluginPath); err != nil {
				log.Printf("Failed to load plugin %s: %v", file.Name(), err)
			}
		}
	}

	log.Printf("Loaded %d plugins", len(ms.plugins))
	return nil
}

// loadSinglePlugin loads a single plugin from file
func (ms *MockServer) loadSinglePlugin(pluginPath string) error {
	data, err := ioutil.ReadFile(pluginPath)
	if err != nil {
		return fmt.Errorf("failed to read plugin file: %v", err)
	}

	var plugin Plugin
	if err := json.Unmarshal(data, &plugin); err != nil {
		return fmt.Errorf("failed to parse plugin file: %v", err)
	}

	if plugin.Name == "" {
		plugin.Name = strings.TrimSuffix(filepath.Base(pluginPath), ".json")
	}

	ms.plugins[plugin.Name] = &plugin
	log.Printf("Loaded plugin: %s (enabled: %t, endpoints: %d)", plugin.Name, plugin.Enabled, len(plugin.Endpoints))
	return nil
}

// LoadConfig loads configuration from JSON file
func (ms *MockServer) LoadConfig() error {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()

	data, err := ioutil.ReadFile(ms.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %v", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config file: %v", err)
	}

	// Set default values
	if config.Port == "" {
		config.Port = "9000"
	}
	if config.PluginsDir == "" {
		config.PluginsDir = "plugins"
	}

	ms.config = &config
	ms.pluginsDir = config.PluginsDir

	// Ensure plugins directory exists
	if err := os.MkdirAll(ms.pluginsDir, 0755); err != nil {
		log.Printf("Warning: Failed to create plugins directory: %v", err)
	}

	return nil
}

// SetupRoutes sets up HTTP routes based on configuration and plugins
func (ms *MockServer) SetupRoutes() {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()

	// Clear existing routes
	ms.router = mux.NewRouter()

	// Add management API endpoints
	ms.setupManagementAPI()

	// Add health check endpoint
	ms.router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}).Methods("GET")

	// Add configured endpoints from main config
	for _, endpoint := range ms.config.Endpoints {
		ms.addEndpoint(endpoint, "main")
	}

	// Add endpoints from enabled plugins
	for pluginName, plugin := range ms.plugins {
		if plugin.Enabled {
			for _, endpoint := range plugin.Endpoints {
				ms.addEndpoint(endpoint, pluginName)
			}
		}
	}

	// Add a catch-all handler for undefined routes
	ms.router.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Endpoint not found",
			"path":  r.URL.Path,
		})
		log.Printf("%s %s - 404 (Not Found)", r.Method, r.URL.Path)
	})
}

// addEndpoint adds a single endpoint to the router
func (ms *MockServer) addEndpoint(endpoint Endpoint, source string) {
	// Create a closure to capture the endpoint configuration
	ep := endpoint // Important: create a copy to avoid closure issues

	ms.router.HandleFunc(ep.Path, func(w http.ResponseWriter, r *http.Request) {
		// Add delay if specified
		if ep.Delay > 0 {
			time.Sleep(time.Duration(ep.Delay) * time.Millisecond)
		}

		// Set custom headers
		if ep.Headers != nil {
			for key, value := range ep.Headers {
				w.Header().Set(key, value)
			}
		}

		// Set content type to JSON if not specified
		if w.Header().Get("Content-Type") == "" {
			w.Header().Set("Content-Type", "application/json")
		}

		// Set status code
		statusCode := ep.StatusCode
		if statusCode == 0 {
			statusCode = http.StatusOK
		}
		w.WriteHeader(statusCode)

		// Write response
		if ep.Response != nil {
			if responseStr, ok := ep.Response.(string); ok {
				fmt.Fprint(w, responseStr)
			} else {
				json.NewEncoder(w).Encode(ep.Response)
			}
		}

		log.Printf("%s %s - %d [%s]", r.Method, r.URL.Path, statusCode, source)
	}).Methods(strings.ToUpper(ep.Method))
}

// setupManagementAPI sets up management API endpoints
func (ms *MockServer) setupManagementAPI() {
	// List all plugins
	ms.router.HandleFunc("/_admin/plugins", func(w http.ResponseWriter, r *http.Request) {
		ms.mutex.RLock()
		defer ms.mutex.RUnlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ms.plugins)
	}).Methods("GET")

	// Get specific plugin
	ms.router.HandleFunc("/_admin/plugins/{name}", func(w http.ResponseWriter, r *http.Request) {
		ms.mutex.RLock()
		defer ms.mutex.RUnlock()

		vars := mux.Vars(r)
		name := vars["name"]

		plugin, exists := ms.plugins[name]
		if !exists {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "Plugin not found"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(plugin)
	}).Methods("GET")

	// Enable/disable plugin
	ms.router.HandleFunc("/_admin/plugins/{name}/toggle", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		name := vars["name"]

		ms.mutex.Lock()
		plugin, exists := ms.plugins[name]
		if !exists {
			ms.mutex.Unlock()
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "Plugin not found"})
			return
		}

		plugin.Enabled = !plugin.Enabled
		ms.mutex.Unlock()

		// Save plugin state to file
		ms.savePlugin(name, plugin)

		// Reload routes
		ms.SetupRoutes()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": fmt.Sprintf("Plugin %s %s", name, map[bool]string{true: "enabled", false: "disabled"}[plugin.Enabled]),
			"enabled": plugin.Enabled,
		})
		log.Printf("Plugin %s %s", name, map[bool]string{true: "enabled", false: "disabled"}[plugin.Enabled])
	}).Methods("POST")

	// Reload all plugins
	ms.router.HandleFunc("/_admin/reload", func(w http.ResponseWriter, r *http.Request) {
		if err := ms.LoadPlugins(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		ms.SetupRoutes()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "Plugins reloaded successfully"})
		log.Println("Plugins reloaded via admin API")
	}).Methods("POST")
} // savePlugin saves a plugin to file
func (ms *MockServer) savePlugin(name string, plugin *Plugin) error {
	pluginPath := filepath.Join(ms.pluginsDir, name+".json")
	data, err := json.MarshalIndent(plugin, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(pluginPath, data, 0644)
}

// WatchConfig watches for configuration file changes and reloads
func (ms *MockServer) WatchConfig() {
	var err error
	ms.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		log.Printf("Failed to create file watcher: %v", err)
		return
	}
	defer ms.watcher.Close()

	// Watch config file directory
	configDir := filepath.Dir(ms.configPath)
	err = ms.watcher.Add(configDir)
	if err != nil {
		log.Printf("Failed to watch config directory: %v", err)
		return
	}

	// Watch plugins directory
	if _, err := os.Stat(ms.pluginsDir); err == nil {
		err = ms.watcher.Add(ms.pluginsDir)
		if err != nil {
			log.Printf("Failed to watch plugins directory: %v", err)
		}
	}

	for {
		select {
		case event, ok := <-ms.watcher.Events:
			if !ok {
				return
			}

			// Check if the modified file is our config file
			if event.Name == ms.configPath && (event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create) {
				log.Println("Config file changed, reloading...")
				if err := ms.LoadConfig(); err != nil {
					log.Printf("Failed to reload config: %v", err)
				} else {
					if err := ms.LoadPlugins(); err != nil {
						log.Printf("Failed to reload plugins: %v", err)
					}
					ms.SetupRoutes()
					log.Println("Configuration reloaded successfully")
				}
			}

			// Check if a plugin file was modified
			if strings.HasPrefix(event.Name, ms.pluginsDir) && strings.HasSuffix(event.Name, ".json") &&
				(event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create || event.Op&fsnotify.Remove == fsnotify.Remove) {
				log.Printf("Plugin file changed: %s", event.Name)
				if err := ms.LoadPlugins(); err != nil {
					log.Printf("Failed to reload plugins: %v", err)
				} else {
					ms.SetupRoutes()
					log.Println("Plugins reloaded successfully")
				}
			}
		case err, ok := <-ms.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("File watcher error: %v", err)
		}
	}
}

// Start starts the mock server
func (ms *MockServer) Start() error {
	// Load initial configuration
	if err := ms.LoadConfig(); err != nil {
		return err
	}

	// Load plugins
	if err := ms.LoadPlugins(); err != nil {
		log.Printf("Warning: Failed to load plugins: %v", err)
	}

	// Setup routes
	ms.SetupRoutes()

	// Start watching for config changes
	go ms.WatchConfig()

	port := ms.config.Port
	log.Printf("Starting mock server on port :%s", port)
	log.Printf("Health check available at: http://localhost:%s/health", port)
	log.Printf("Admin API available at: http://localhost:%s/_admin/", port)
	log.Printf("Config file: %s", ms.configPath)
	log.Printf("Plugins directory: %s", ms.pluginsDir)

	return http.ListenAndServe(":"+port, ms.router)
}

func main() {
	// Check for config file argument
	configPath := "config.json"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Printf("Config file %s does not exist, creating example config...", configPath)
		if err := createExampleConfig(configPath); err != nil {
			log.Fatalf("Failed to create example config: %v", err)
		}
		log.Printf("Example config created at %s", configPath)
	}

	// Create and start mock server
	server := NewMockServer(configPath)
	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// createExampleConfig creates an example configuration file
func createExampleConfig(configPath string) error {
	exampleConfig := Config{
		Port:       "9000",
		PluginsDir: "plugins",
		Endpoints: []Endpoint{
			{
				Path:       "/api/users",
				Method:     "GET",
				StatusCode: 200,
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Response: []map[string]interface{}{
					{
						"id":    1,
						"name":  "John Doe",
						"email": "john@example.com",
					},
					{
						"id":    2,
						"name":  "Jane Smith",
						"email": "jane@example.com",
					},
				},
			},
			{
				Path:       "/api/users/{id}",
				Method:     "GET",
				StatusCode: 200,
				Response: map[string]interface{}{
					"id":    1,
					"name":  "John Doe",
					"email": "john@example.com",
				},
			},
		},
	}

	data, err := json.MarshalIndent(exampleConfig, "", "  ")
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(configPath, data, 0644); err != nil {
		return err
	}

	// Create example plugin
	return createExamplePlugin("plugins")
}

// createExamplePlugin creates an example plugin
func createExamplePlugin(pluginsDir string) error {
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		return err
	}

	examplePlugin := Plugin{
		Name:        "example-plugin",
		Description: "Example plugin demonstrating various API endpoints",
		Enabled:     true,
		Endpoints: []Endpoint{
			{
				Path:       "/api/products",
				Method:     "GET",
				StatusCode: 200,
				Response: []map[string]interface{}{
					{
						"id":    1,
						"name":  "Product A",
						"price": 99.99,
					},
					{
						"id":    2,
						"name":  "Product B",
						"price": 149.99,
					},
				},
			},
			{
				Path:       "/api/products/{id}",
				Method:     "GET",
				StatusCode: 200,
				Response: map[string]interface{}{
					"id":    1,
					"name":  "Product A",
					"price": 99.99,
				},
			},
			{
				Path:       "/api/products",
				Method:     "POST",
				StatusCode: 201,
				Response: map[string]interface{}{
					"id":      3,
					"message": "Product created successfully",
				},
				Delay: 300,
			},
		},
	}

	data, err := json.MarshalIndent(examplePlugin, "", "  ")
	if err != nil {
		return err
	}

	pluginPath := filepath.Join(pluginsDir, "example-plugin.json")
	return ioutil.WriteFile(pluginPath, data, 0644)
}
