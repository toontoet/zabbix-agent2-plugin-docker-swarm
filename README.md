# Zabbix Agent 2 Docker Swarm Plugin

A production-ready loadable plugin for Zabbix Agent 2 that provides comprehensive monitoring of Docker Swarm services.

## Overview

The standard Docker plugin in Zabbix is inadequate for Docker Swarm monitoring because containers get random suffixes when restarted, creating new Zabbix items and breaking historical data continuity. This plugin solves this by monitoring at the **service level** instead of container level, providing stable service discovery and tracking desired vs running replica counts.

## Features

- **Service Discovery**: Low-Level Discovery (LLD) for all Docker Swarm services with stack grouping
- **Stack Discovery**: Low-Level Discovery for Docker Compose stacks
- **Replica Monitoring**: Track desired vs running replica counts per service
- **Stack Health Monitoring**: Aggregate health status by Docker Compose stack
- **Stable Monitoring**: Service-based monitoring prevents historical data fragmentation
- **Cross-Architecture**: Supports both x86_64 and ARM64 Linux systems

## Requirements

- **Zabbix Agent 2**: Version 6.0 or later
- **Go**: Version 1.21+ (for building from source)
- **Docker Swarm**: Linux environment with Docker Swarm mode enabled
- **Permissions**: Zabbix user must have access to Docker socket

## Installation

### 1. Download or Build

**Option A: Build from source**
```bash
git clone <repository-url>
cd zabbix-agent2-plugin-docker-swarm/src

# For x86_64 Linux (most common)
make build-x86_64

# For ARM64 Linux 
make build-arm64

# For both architectures
make build-all
```

**Option B: Download pre-built binaries**
Download the appropriate binary for your system architecture from the releases page.

### 2. Install Plugin

Copy the binary to your Docker Swarm node:

```bash
# For x86_64 systems (Intel/AMD):
sudo cp docker-swarm-linux-x86_64 /var/lib/zabbix/plugins/docker-swarm

# For ARM64 systems:
sudo cp docker-swarm-linux-arm64 /var/lib/zabbix/plugins/docker-swarm

# Set permissions:
sudo chmod +x /var/lib/zabbix/plugins/docker-swarm
sudo chown zabbix:zabbix /var/lib/zabbix/plugins/docker-swarm
```

### 3. Configure Zabbix Agent 2

Add the plugin configuration to your Zabbix Agent 2 configuration file:

```ini
# /etc/zabbix/zabbix_agent2.conf
Plugins.DockerSwarm.System.Path=/var/lib/zabbix/plugins/docker-swarm
```

### 4. Set Docker Permissions

Ensure the Zabbix user has access to the Docker socket:

```bash
sudo usermod -a -G docker zabbix
```

### 5. Restart Services

```bash
sudo systemctl restart zabbix-agent2
```

### 6. Verify Installation

Test the plugin functionality:

```bash
zabbix_get -s localhost -k "swarm.services.discovery"
```

## Quick Start

After installation, test the new stack monitoring features:

```bash
# Test service discovery with stack information
zabbix_get -s localhost -k "swarm.services.discovery"

# Test stack discovery
zabbix_get -s localhost -k "swarm.stacks.discovery"

# Test stack health (replace 'mystack' with actual stack name)
zabbix_get -s localhost -k "swarm.stack.health[mystack]"
```

For detailed examples and Zabbix template configuration, see [EXAMPLES.md](EXAMPLES.md).

## Supported Metrics

| Key | Description | Returns |
|-----|-------------|---------|
| `swarm.services.discovery` | Service discovery for LLD | JSON array with `{#SERVICE.ID}`, `{#SERVICE.NAME}`, and `{#STACK.NAME}` macros |
| `swarm.service.replicas_desired[<service_id>]` | Configured replica count | Integer (desired replicas) |
| `swarm.service.replicas_running[<service_id>]` | Running task count | Integer (running tasks) |
| `swarm.stacks.discovery` | Stack discovery for LLD | JSON array with `{#STACK.NAME}` macro |
| `swarm.stack.health[<stack_name>]` | Stack health status | JSON with health metrics |

## Zabbix Template Configuration

### Service-Level Monitoring

#### Discovery Rule
- **Name**: Docker Swarm Services
- **Key**: `swarm.services.discovery`
- **Update Interval**: 300s (5 minutes)

#### Item Prototypes
1. **Desired Replicas**
   - **Name**: Service {#SERVICE.NAME} ({#STACK.NAME}) desired replicas
   - **Key**: `swarm.service.replicas_desired[{#SERVICE.ID}]`
   - **Type**: Zabbix agent

2. **Running Replicas**
   - **Name**: Service {#SERVICE.NAME} ({#STACK.NAME}) running replicas
   - **Key**: `swarm.service.replicas_running[{#SERVICE.ID}]`
   - **Type**: Zabbix agent

#### Trigger Prototype
- **Name**: Service {#SERVICE.NAME} ({#STACK.NAME}) replica mismatch
- **Expression**: `last(/Template/swarm.service.replicas_running[{#SERVICE.ID}])<>last(/Template/swarm.service.replicas_desired[{#SERVICE.ID}])`
- **Severity**: Warning

### Stack-Level Monitoring

#### Discovery Rule
- **Name**: Docker Compose Stacks
- **Key**: `swarm.stacks.discovery`
- **Update Interval**: 600s (10 minutes)

#### Item Prototypes
1. **Stack Health**
   - **Name**: Stack {#STACK.NAME} health status
   - **Key**: `swarm.stack.health[{#STACK.NAME}]`
   - **Type**: Zabbix agent
   - **Value Type**: Text

2. **Stack Health Percentage** (Calculated Item)
   - **Name**: Stack {#STACK.NAME} health percentage
   - **Formula**: `jsonpath(last(/Template/swarm.stack.health[{#STACK.NAME}]),"$.health_percentage")`
   - **Units**: %

3. **Unhealthy Services Count** (Calculated Item)
   - **Name**: Stack {#STACK.NAME} unhealthy services
   - **Formula**: `jsonpath(last(/Template/swarm.stack.health[{#STACK.NAME}]),"$.unhealthy_services")`

#### Trigger Prototypes
1. **Stack Health Critical**
   - **Name**: Stack {#STACK.NAME} has unhealthy services
   - **Expression**: `jsonpath(last(/Template/swarm.stack.health[{#STACK.NAME}]),"$.unhealthy_services")>0`
   - **Severity**: High

2. **Stack Health Warning**
   - **Name**: Stack {#STACK.NAME} health below 100%
   - **Expression**: `jsonpath(last(/Template/swarm.stack.health[{#STACK.NAME}]),"$.health_percentage")<100`
   - **Severity**: Warning

## Architecture Support

This plugin supports the following architectures:

- **x86_64/AMD64**: Intel and AMD processors (most common)
- **ARM64/aarch64**: ARM processors (including Apple Silicon via cross-compilation)

Use `uname -m` to determine your system architecture.

## Example Outputs

### Service Discovery
```json
[
  {
    "{#SERVICE.ID}": "abc123def456",
    "{#SERVICE.NAME}": "web-frontend",
    "{#STACK.NAME}": "myapp"
  },
  {
    "{#SERVICE.ID}": "def456ghi789",
    "{#SERVICE.NAME}": "api-backend", 
    "{#STACK.NAME}": "myapp"
  },
  {
    "{#SERVICE.ID}": "ghi789jkl012",
    "{#SERVICE.NAME}": "nginx-proxy",
    "{#STACK.NAME}": "standalone"
  }
]
```

### Stack Discovery
```json
[
  {
    "{#STACK.NAME}": "myapp"
  },
  {
    "{#STACK.NAME}": "monitoring"
  },
  {
    "{#STACK.NAME}": "standalone"
  }
]
```

### Stack Health
```json
{
  "total_services": 5,
  "healthy_services": 4,
  "unhealthy_services": 1,
  "health_percentage": 80.0
}
```

## Development

### Building
```bash
# Install dependencies
make deps

# Check code quality
make check

# Build for production
make build-all

# Clean build artifacts
make clean
```

### Project Structure
```
zabbix-agent2-plugin-docker-swarm/
├── README.md         # This file
├── LICENSE          # MIT License
└── src/             # Source code directory
    ├── main.go      # Entry point and version handling
    ├── plugin.go    # Core plugin logic and metric handlers
    ├── client.go    # Docker API client
    ├── types.go     # Docker API data structures
    ├── go.mod       # Go module dependencies
    ├── Makefile     # Build automation
    ├── swarm.conf   # Configuration example
    └── .gitignore   # Git ignore rules
```

## Troubleshooting

### Architecture Mismatch
**Error**: `exec format error`
**Solution**: Use the correct binary for your system architecture (x86_64 vs ARM64).

### Permission Denied
**Error**: Plugin cannot access Docker socket
**Solution**: Ensure `zabbix` user is in the `docker` group and restart zabbix-agent2.

### Plugin Not Loading
**Error**: Plugin not found or fails to load
**Solution**: 
1. Verify the binary path in `zabbix_agent2.conf`
2. Check binary permissions and ownership
3. Review Zabbix Agent 2 logs for detailed error messages

### No Services Discovered
**Error**: Empty discovery results
**Solution**:
1. Verify Docker Swarm is running (`docker node ls`)
2. Ensure services exist (`docker service ls`)
3. Check Docker socket accessibility

### Stack Name Shows "standalone"
**Info**: Services not created with `docker stack deploy` will show `{#STACK.NAME}` as "standalone"
**Note**: This is expected behavior for services created with `docker service create` or similar commands

### Stack Health Returns Error
**Error**: "stack not found" when querying `swarm.stack.health[<stack_name>]`
**Solution**:
1. Verify the stack name exists in `swarm.stacks.discovery` output
2. Check that services in the stack have the `com.docker.stack.namespace` label
3. Ensure the stack name is spelled correctly (case-sensitive)

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Support

For issues and questions:
1. Check the troubleshooting section above
2. Review Zabbix Agent 2 logs
3. Open an issue on the project repository
