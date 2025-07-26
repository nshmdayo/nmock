# nmock

A mock server that can dynamically add API endpoints by reading JSON files. It supports plugin functionality to manage endpoints across multiple JSON files.

## Features

- Load API endpoint configurations from JSON files
- Dynamic endpoint management through plugin system
- Auto-reload by monitoring changes in configuration and plugin files
- Support for custom headers and status codes
- Configurable response delays
- Support for path variables (e.g., `/api/users/{id}`)
- Admin API (enable/disable plugins, list plugins)

## Usage

### Start and Build Server

```bash
make start
```

### Stop and Remove Server

```bash
make stop
```

### Development Mode

```bash
make dev
```

### Direct Execution

```bash
cd app
go run main.go [config_file]
```

By default, it uses the `config.json` file. You can specify a different configuration file:

```bash
go run main.go my-config.json
```

## Configuration File Format

The configuration file is in JSON format with the following structure:

```json
{
  "port": "9000",
  "plugins_dir": "plugins",
  "endpoints": [
    {
      "path": "/api/users",
      "method": "GET",
      "status_code": 200,
      "headers": {
        "Content-Type": "application/json"
      },
      "response": [
        {
          "id": 1,
          "name": "John Doe",
          "email": "john@example.com"
        }
      ],
      "delay": 100
    }
  ]
}
```

### Configuration Items

- `port` (optional): Server port number (default: 9000)
- `plugins_dir` (optional): Plugin directory path (default: plugins)
- `endpoints`: Array of endpoints

## Plugin System

Plugins are managed as JSON files within the `plugins` directory. Each plugin file has the following structure:

```json
{
  "name": "example-plugin",
  "description": "Example plugin demonstrating various API endpoints",
  "enabled": true,
  "endpoints": [
    {
      "path": "/api/products",
      "method": "GET",
      "status_code": 200,
      "response": [
        {
          "id": 1,
          "name": "Product A",
          "price": 99.99
        }
      ]
    }
  ]
}
```

### Plugin Configuration Items

- `name` (required): Plugin name
- `description` (optional): Plugin description
- `enabled` (required): Plugin enable/disable state
- `endpoints` (required): Array of endpoints

#### Endpoint Configuration

- `path` (required): API path (supports path variables: `/api/users/{id}`)
- `method` (required): HTTP method (GET, POST, PUT, DELETE, etc.)
- `status_code` (optional): HTTP status code (default: 200)
- `headers` (optional): Custom headers
- `response` (required): Response body (JSON object, array, or string)
- `delay` (optional): Response delay (milliseconds)

## Admin API

The server has built-in admin API functionality for plugin management:

### List Plugins

```bash
curl http://localhost:9000/_admin/plugins
```

### Get Plugin Details

```bash
curl http://localhost:9000/_admin/plugins/example-plugin
```

### Enable/Disable Plugin

```bash
curl -X POST http://localhost:9000/_admin/plugins/example-plugin/toggle
```

### Reload Plugins

```bash
curl -X POST http://localhost:9000/_admin/reload
```

## Built-in Endpoints

- `GET /health`: Health check endpoint
- `GET /_admin/plugins`: List all plugins
- `GET /_admin/plugins/{name}`: Get specific plugin details
- `POST /_admin/plugins/{name}/toggle`: Enable/disable plugin
- `POST /_admin/reload`: Reload plugins

## Examples

### Basic Usage

1. Start the server:
```bash
cd app && go run main.go
```

2. Test APIs:
```bash
# Get user list (main config)
curl http://localhost:9000/api/users

# Get product list (plugin)
curl http://localhost:9000/api/products

# Authentication endpoint (plugin)
curl -X POST http://localhost:9000/api/auth/login

# Plugin management
curl http://localhost:9000/_admin/plugins
```

### Adding a New Plugin

1. Create a new JSON file in the `plugins` directory:

```json
{
  "name": "my-plugin",
  "description": "My custom plugin",
  "enabled": true,
  "endpoints": [
    {
      "path": "/api/custom",
      "method": "GET",
      "status_code": 200,
      "response": {
        "message": "Hello from my plugin!"
      }
    }
  ]
}
```

2. Save the file and the server will automatically reload, making the new endpoint available.

### Plugin Hot Reload

- When configuration files or plugin files are modified, new settings are automatically applied without restarting the server
- Plugin enable/disable can be done dynamically using the admin API

## Development

```bash
# Install dependencies
go mod tidy

# Run application
go run main.go

# Build
go build -o nmock main.go
```
