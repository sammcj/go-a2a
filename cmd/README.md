# Go A2A Standalone Applications

This directory contains standalone applications for the Go A2A library:

- `a2a-server`: A standalone A2A server application
- `a2a-client`: A standalone A2A client application

## A2A Server

The A2A server is a standalone application that implements the A2A protocol. It can be used to host an A2A agent that can receive and process tasks from A2A clients.

### Building

```bash
go build -o a2a-server ./cmd/a2a-server
```

### Usage

```bash
./a2a-server [options]
```

#### Options

- `--config`: Path to configuration file (JSON or YAML)
- `--listen`: Address to listen on (default: ":8080")
- `--agent-card`: Path to agent card file (JSON or YAML)
- `--agent-card-path`: Path to serve agent card at (default: "/.well-known/agent.json")
- `--a2a-path-prefix`: Path prefix for A2A endpoints (default: "/a2a")
- `--log-level`: Log level (debug, info, warn, error, fatal) (default: "info")
- `--plugin-path`: Path to plugin directory

### Configuration

The server can be configured using a JSON or YAML file. Here's an example configuration:

```json
{
  "listenAddress": ":8080",
  "agentCardPath": "/.well-known/agent.json",
  "a2aPathPrefix": "/a2a",
  "logLevel": "info",
  "pluginPath": "/app/plugins",
  "agentCard": {
    "a2aVersion": "1.0",
    "id": "go-a2a-server",
    "name": "Go A2A Server",
    "description": "A standalone A2A server implemented in Go",
    "provider": {
      "name": "Go A2A",
      "uri": "https://github.com/sammcj/go-a2a"
    },
    "skills": [
      {
        "id": "echo",
        "name": "Echo",
        "description": "Echoes back the input message"
      }
    ],
    "capabilities": {
      "supportsStreaming": true,
      "supportsSessions": true,
      "supportsPushNotification": true
    },
    "contactEmail": "example@example.com",
    "homepageUri": "https://github.com/sammcj/go-a2a",
    "documentationUri": "https://github.com/sammcj/go-a2a/blob/main/README.md"
  }
}
```

### Plugins

The server supports plugins for task handling. Plugins are Go plugins that implement the `TaskHandlerPlugin` interface. Plugins are loaded from the directory specified by the `--plugin-path` flag or the `pluginPath` configuration option.

See the `cmd/common/plugin_example.go` file for examples of how to implement plugins.

## A2A Client

The A2A client is a standalone application that can be used to interact with A2A servers.

### Building

```bash
go build -o a2a-client ./cmd/a2a-client
```

### Usage

```bash
./a2a-client [options] <command> [command options]
```

#### Options

- `--config`: Path to configuration file (JSON or YAML)
- `--url`: URL of the A2A agent
- `--output`: Output format (json, pretty) (default: "pretty")
- `--auth`: Authentication header (format: 'Name: Value')
- `--timeout`: Request timeout (default: 30s)
- `--interactive`: Interactive mode (not implemented yet)

#### Commands

- `send`: Send a task to an agent
- `get`: Get a task from an agent
- `cancel`: Cancel a task
- `subscribe`: Subscribe to task updates
- `push`: Configure push notifications
- `card`: Get agent card information

### Configuration

The client can be configured using a JSON or YAML file. Here's an example configuration:

```json
{
  "defaultAgentUrl": "http://localhost:8080",
  "outputFormat": "pretty",
  "authentication": {
    "Authorization": "Bearer your-token-here"
  }
}
```

### Examples

#### Get Agent Card

```bash
./a2a-client --url http://localhost:8080 card
```

#### Send a Task

```bash
./a2a-client --url http://localhost:8080 send --message "Hello, world!"
```

#### Send a Task with Streaming

```bash
./a2a-client --url http://localhost:8080 send --message "Hello, world!" --stream
```

#### Get a Task

```bash
./a2a-client --url http://localhost:8080 get --task task_123456789
```

#### Cancel a Task

```bash
./a2a-client --url http://localhost:8080 cancel --task task_123456789
```

#### Subscribe to Task Updates

```bash
./a2a-client --url http://localhost:8080 subscribe --task task_123456789
```

#### Configure Push Notifications

```bash
./a2a-client --url http://localhost:8080 push --task task_123456789 --url https://example.com/webhook --auth bearer:your-token-here
```

## Docker

Both applications can be run using Docker. A Dockerfile and docker-compose.yml file are provided in the root directory of the project.

### Building the Docker Image

```bash
docker build -t go-a2a .
```

### Running the Server

```bash
docker run -p 8080:8080 -v ./config:/app/config -v ./plugins:/app/plugins go-a2a
```

### Running the Client

```bash
docker run --entrypoint /app/bin/a2a-client go-a2a --url http://localhost:8080 card
```

### Using Docker Compose

```bash
docker-compose up
