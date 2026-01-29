# Context Mirroring and MCP Implementation

## Overview
The AOI Protocol project now includes Context Mirroring functionality and MCP (Model Context Protocol) bridge preparation.

## Architecture

### Context Package (`internal/context/`)

1. **types.go** - Core type definitions
   - `ContextEntry` - Single context entry with metadata
   - `ContextSummary` - Summary view of current context
   - `ContextQuery` - Parameters for querying context
   - `ContextHistory` - Paginated list of context entries
   - `WatchRequest/WatchResponse` - Directory watching configuration
   - `FileChangeEvent` - Detected file changes

2. **store.go** - Context storage and indexing
   - `ContextStore` - In-memory store with indexes by project/file/topic/type
   - Automatic cleanup of expired entries
   - Query interface with filtering and pagination
   - Thread-safe with RWMutex

3. **monitor.go** - File watching and activity tracking
   - `ContextMonitor` - Watches directories for file changes
   - Polling-based change detection using file hashes
   - Automatic project and topic inference from file paths
   - Activity recording support

4. **api.go** - REST and JSON-RPC endpoints
   - REST: `/api/v1/context`, `/api/v1/context/history`, `/api/v1/context/watch`, `/api/v1/context/stats`
   - JSON-RPC: `aoi.context`, `aoi.context.history`, `aoi.context.watch`, `aoi.context.activity`

### MCP Package (`internal/mcp/`)

1. **types.go** - MCP protocol types
   - JSON-RPC 2.0 types (Request, Response, Error)
   - MCP Initialize types and capabilities
   - Tool, Resource, Prompt types
   - Content blocks (text, image, resource)
   - Log levels and error codes

2. **client.go** - MCP client implementation
   - `MCPClient` - Connects to MCP servers via stdio or HTTP
   - Methods: `ListTools`, `CallTool`, `ListResources`, `ReadResource`, `ListPrompts`, `GetPrompt`
   - Protocol version: `2024-11-05`

3. **bridge.go** - AOI to MCP bridge
   - `MCPBridge` - Translates AOI queries to MCP tool calls
   - Resource caching with configurable timeout
   - Tool mappings for query pattern matching
   - Discovery of MCP server capabilities

## Configuration

### Context Configuration
```json
{
  "context": {
    "watch_paths": [".", "./docs", "./src"],
    "index_interval": "5m",
    "default_ttl": "24h",
    "poll_interval": "5s",
    "ignore_patterns": [".git", "node_modules"]
  }
}
```

### MCP Configuration
```json
{
  "mcp": {
    "enabled": false,
    "cache_timeout": "5m",
    "servers": [
      {
        "name": "filesystem",
        "transport": "stdio",
        "command": "npx",
        "args": ["-y", "@modelcontextprotocol/server-filesystem", "/path"],
        "auto_connect": false
      },
      {
        "name": "external-api",
        "transport": "http",
        "base_url": "http://localhost:3000",
        "auto_connect": false
      }
    ]
  }
}
```

## Integration Points

### main.go
- Initializes ContextStore with configurable TTL
- Creates ContextMonitor with poll interval
- Sets up watch paths from configuration
- Creates MCPBridge connected to ContextStore
- Configures MCP servers from configuration
- Passes context API and MCP bridge to protocol server

### transport.go
- Routes `aoi.context*` methods to ContextAPI
- Routes `aoi.mcp*` methods to MCPBridge
- Registers REST endpoints from ContextAPI

## JSON-RPC Methods

### Context Methods
- `aoi.context` - Get current context summary
- `aoi.context.history` - Query context history
- `aoi.context.watch` - Add directory to watch
- `aoi.context.activity` - Record manual activity

### MCP Methods
- `aoi.mcp.status` - Get MCP bridge status
- `aoi.mcp.discover` - Discover MCP server capabilities
- `aoi.mcp.tools` - List tools from MCP server
- `aoi.mcp.call` - Call MCP tool
- `aoi.mcp.resources` - List MCP resources
- `aoi.mcp.read` - Read MCP resource

## Testing
All packages have comprehensive unit tests:
- `store_test.go` - Store operations, indexing, expiration
- `monitor_test.go` - Watching, file change detection
- `api_test.go` - REST and JSON-RPC handlers
- `types_test.go` - JSON serialization
- `client_test.go` - HTTP transport, tool/resource operations
- `bridge_test.go` - Query translation, resource caching
