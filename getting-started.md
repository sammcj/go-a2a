# Getting Started with go-a2a

This guide provides practical examples for using the go-a2a standalone applications. These examples demonstrate common use cases and workflows to help you get started quickly.

## Prerequisites

- Go 1.18 or later (for building from source)
- Docker (optional, for containerized deployment)

## Building the Applications

### From Source

```bash
# Clone the repository
git clone https://github.com/sammcj/go-a2a.git
cd go-a2a

# Build the server and client
make build
```

### Using Docker

```bash
# Build the Docker image
docker build -t go-a2a .
```

## Example 1: Simple Echo Server

This example demonstrates setting up a basic A2A server that echoes back messages sent to it.

### Step 1: Create a Server Configuration

Create a file named `config/echo-server.json`:

```json
{
  "listenAddress": ":8080",
  "agentCardPath": "/.well-known/agent.json",
  "a2aPathPrefix": "/a2a",
  "logLevel": "info",
  "pluginPath": "./plugins",
  "agentCard": {
    "a2aVersion": "1.0",
    "id": "echo-agent",
    "name": "Echo Agent",
    "description": "A simple agent that echoes back messages",
    "provider": {
      "name": "go-a2a Example",
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
    }
  }
}
```

### Step 2: Build the Echo Plugin

The echo plugin is already included in the `plugins` directory. Make sure it's built:

```bash
make build-plugin
```

### Step 3: Start the Server

```bash
# Using the binary
./bin/a2a-server --config config/echo-server.json

# Or using Docker
docker run -p 8080:8080 -v $(pwd)/config:/app/config -v $(pwd)/plugins:/app/plugins go-a2a --config /app/config/echo-server.json
```

### Step 4: Interact with the Server

```bash
# Get the agent card
./bin/a2a-client --url http://localhost:8080 card

# Send a message and get a response
./bin/a2a-client --url http://localhost:8080 send --message "Hello, A2A!"

# Send a message with streaming updates
./bin/a2a-client --url http://localhost:8080 send --message "Hello with streaming" --stream
```

## Example 2: LLM-Powered Assistant

This example demonstrates setting up an A2A server that uses a local LLM (via Ollama) to respond to queries.

### Step 1: Install Ollama

Follow the instructions at [ollama.ai](https://ollama.ai) to install Ollama on your system.

### Step 2: Pull a Model

```bash
ollama pull llama3
```

### Step 3: Create a Server Configuration

Create a file named `config/llm-server.json`:

```json
{
  "listenAddress": ":8081",
  "agentCardPath": "/.well-known/agent.json",
  "a2aPathPrefix": "/a2a",
  "logLevel": "info",
  "llmConfig": {
    "provider": "ollama",
    "model": "llama3",
    "systemPrompt": "You are a helpful assistant that responds to user queries."
  },
  "agentCard": {
    "a2aVersion": "1.0",
    "id": "llm-assistant",
    "name": "LLM Assistant",
    "description": "An intelligent assistant powered by a local LLM",
    "provider": {
      "name": "go-a2a Example",
      "uri": "https://github.com/sammcj/go-a2a"
    },
    "skills": [
      {
        "id": "general-assistance",
        "name": "General Assistance",
        "description": "Provides helpful responses to general queries"
      }
    ],
    "capabilities": {
      "supportsStreaming": true,
      "supportsSessions": true,
      "supportsPushNotification": true
    }
  }
}
```

### Step 4: Start the Server

```bash
# Using the binary
./bin/a2a-server --config config/llm-server.json

# Or using Docker (ensure Ollama is accessible from Docker)
docker run -p 8081:8081 -v $(pwd)/config:/app/config go-a2a --config /app/config/llm-server.json
```

### Step 5: Interact with the LLM Assistant

```bash
# Get the agent card
./bin/a2a-client --url http://localhost:8081 card

# Ask a question with streaming response
./bin/a2a-client --url http://localhost:8081 send --message "What are the benefits of using the A2A protocol?" --stream

# Have a conversation by continuing a task
TASK_ID=$(./bin/a2a-client --url http://localhost:8081 send --message "Tell me about Go programming" --output json | jq -r .id)
./bin/a2a-client --url http://localhost:8081 send --message "What are some popular Go web frameworks?" --task $TASK_ID
```

## Example 3: Setting Up Push Notifications

This example demonstrates how to configure push notifications for task updates.

### Step 1: Start a Server

Use either the echo server or LLM server from the previous examples.

### Step 2: Set Up a Webhook Receiver

For testing purposes, you can use a service like [webhook.site](https://webhook.site) to get a temporary webhook URL.

### Step 3: Send a Task and Configure Push Notifications

```bash
# Send a task
TASK_ID=$(./bin/a2a-client --url http://localhost:8080 send --message "This is a test message" --output json | jq -r .id)

# Configure push notifications
./bin/a2a-client --url http://localhost:8080 push --task $TASK_ID --url "https://webhook.site/your-unique-url" --include-task true

# Verify the push configuration
./bin/a2a-client --url http://localhost:8080 push --task $TASK_ID --get
```

### Step 4: Send Another Message to the Same Task

```bash
./bin/a2a-client --url http://localhost:8080 send --message "This should trigger a push notification" --task $TASK_ID
```

Check your webhook receiver to see the push notification.

## Example 4: Using Docker Compose

This example demonstrates how to use Docker Compose to run both the server and client.

### Step 1: Create a Docker Compose Configuration

The project already includes a `docker-compose.yml` file. You can use it as is or customize it for your needs.

### Step 2: Start the Services

```bash
docker-compose up -d
```

### Step 3: Use the Client to Interact with the Server

```bash
# Using the client container to interact with the server
docker-compose exec a2a-client /app/bin/a2a-client --url http://a2a-server:8080 card
docker-compose exec a2a-client /app/bin/a2a-client --url http://a2a-server:8080 send --message "Hello from Docker Compose!"
```

### Step 4: Stop the Services

```bash
docker-compose down
```

## Example 5: Authentication

This example demonstrates how to set up and use authentication.

### Step 1: Create a Server Configuration with Authentication

Create a file named `config/auth-server.json`:

```json
{
  "listenAddress": ":8082",
  "agentCardPath": "/.well-known/agent.json",
  "a2aPathPrefix": "/a2a",
  "logLevel": "info",
  "pluginPath": "./plugins",
  "authentication": {
    "required": true,
    "bearerToken": "secret-token-for-testing"
  },
  "agentCard": {
    "a2aVersion": "1.0",
    "id": "secure-agent",
    "name": "Secure Agent",
    "description": "An agent that requires authentication",
    "provider": {
      "name": "go-a2a Example",
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
    "authentication": [
      {
        "type": "bearer",
        "scheme": "Bearer"
      }
    ]
  }
}
```

### Step 2: Start the Server

```bash
./bin/a2a-server --config config/auth-server.json
```

### Step 3: Attempt to Access Without Authentication

```bash
# This should fail with an authentication error
./bin/a2a-client --url http://localhost:8082 card
```

### Step 4: Access with Authentication

```bash
# Create a client configuration file with authentication
cat > config/auth-client.json << EOF
{
  "defaultAgentUrl": "http://localhost:8082",
  "outputFormat": "pretty",
  "authentication": {
    "Authorization": "Bearer secret-token-for-testing"
  }
}
EOF

# Use the client with the configuration file
./bin/a2a-client --config config/auth-client.json card

# Or specify the authentication header directly
./bin/a2a-client --url http://localhost:8082 --auth "Authorization: Bearer secret-token-for-testing" card
```

## Next Steps

Now that you've seen some basic examples, you can:

1. Explore the `examples` directory for more advanced usage patterns
2. Create your own plugins to implement custom agent behavior
3. Integrate with LLMs for more sophisticated AI capabilities
4. Connect to MCP servers to leverage additional tools and resources
5. Build your own applications using the go-a2a library

For more information, refer to the [README.md](README.md) and [cmd/README.md](cmd/README.md) files.
