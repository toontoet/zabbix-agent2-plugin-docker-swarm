# Zabbix Agent 2 Docker Swarm Plugin

A production-ready loadable plugin for Zabbix Agent 2 that provides comprehensive monitoring of Docker Swarm services.

## Overview

The standard Docker plugin in Zabbix is inadequate for 
Docker Swarm monitoring because containers get random suffixes 
when restarted, creating new Zabbix items and breaking historical data continuity. 
This plugin solves this by monitoring at the **service level** instead of container level, 
providing stable service discovery and tracking desired vs running replica counts.

## Features

- **Service Discovery**: Low-Level Discovery (LLD) for all Docker Swarm services with stack grouping
- **Stack Discovery**: Low-Level Discovery for Docker Compose stacks
- **Replica Monitoring**: Track desired vs running replica counts per service
- **Restart Detection**: Monitor and alert on service task restarts/crashes
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

# Or build both
make build
```

**Option B: Download pre-built binaries**

Download the latest release from the 
  [Releases](https://github.com/your-username/zabbix-agent2-plugin-docker-swarm/releases) page.

### 2. Install Plugin

```bash
# Copy the binary to Zabbix plugins directory
sudo cp docker-swarm-linux-x86_64 /var/lib/zabbix/plugins/docker-swarm
sudo chmod +x /var/lib/zabbix/plugins/docker-swarm
sudo chown zabbix:zabbix /var/lib/zabbix/plugins/docker-swarm
```

### 3. Configure Zabbix Agent 2

Add to `/etc/zabbix/zabbix_agent2.conf`:

```ini
Plugins.DockerSwarm.System.Path=/var/lib/zabbix/plugins/docker-swarm
Plugins.DockerSwarm.System.Timeout=30
```

### 4. Configure Docker Socket Access

```bash
# Add zabbix user to docker group
sudo usermod -aG docker zabbix

# Or set proper permissions on socket
sudo chmod 666 /var/run/docker.sock
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

After installation, test the monitoring features:

```bash
# Test service discovery with stack information
zabbix_get -s localhost -k "swarm.services.discovery"

# Test stack discovery
zabbix_get -s localhost -k "swarm.stacks.discovery"

# Test stack health (replace 'mystack' with actual stack name)
zabbix_get -s localhost -k "swarm.stack.health[mystack]"

# Test restart monitoring (replace 'service_id' with actual service ID)
zabbix_get -s localhost -k "swarm.service.restarts[service_id]"
```

For detailed examples and Zabbix template configuration, see [EXAMPLES.md](EXAMPLES.md).

## Supported Metrics

| Key | Description | Returns |
|-----|-------------|---------|
| `swarm.services.discovery` | Service discovery for LLD | JSON array with 
  `{#SERVICE.ID}`, `{#SERVICE.NAME}`, and `{#STACK.NAME}` macros |
| `swarm.service.replicas_desired[<service_id>]` | Configured replica count | Integer (desired replicas) |
| `swarm.service.replicas_running[<service_id>]` | Running task count | Integer (running tasks) |
| `swarm.service.restarts[<service_id>]` | Number of task restarts (crashed tasks) | Integer (restart count) |
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

3. **Restart Count**

   - **Name**: Service {#SERVICE.NAME} ({#STACK.NAME}) restart count
   - **Key**: `swarm.service.restarts[{#SERVICE.ID}]`
   - **Type**: Zabbix agent
   - **Store Value**: Delta (speed per second)
   - **Note**: Use Delta to track increase in restarts over time

#### Trigger Prototypes

1. **Replica Mismatch**

   - **Name**: Service {#SERVICE.NAME} ({#STACK.NAME}) replica mismatch
   - **Expression**: `last(/Template/swarm.service.replicas_running[{#SERVICE.ID}])<>last(/Template/swarm.service.replicas_desired[{#SERVICE.ID}])`
   - **Severity**: Warning

2. **Service Restarted**

   - **Name**: Service {#SERVICE.NAME} ({#STACK.NAME}) has restarted
   - **Expression**: `change(/Template/swarm.service.restarts[{#SERVICE.ID}])>0`
   - **Severity**: Warning
   - **Description**: A task for this service has crashed and been restarted

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

## How It Works

### Service Discovery

The plugin discovers all Docker Swarm services and groups them by Docker 
Compose stack using the `com.docker.stack.namespace` label. Services 
without this label are marked as "standalone".

### Stack Health Calculation

For each stack, the plugin:

1. Identifies all services belonging to the stack
2. Compares desired vs running replica counts for each service
3. Calculates health percentage: `(healthy_services / total_services) * 100`
4. Returns comprehensive health metrics

### Restart Detection

The plugin tracks tasks that have failed or shutdown with non-zero exit codes, 
indicating container crashes that triggered Docker Swarm restarts.

## Troubleshooting

### Common Issues

1. **Permission Denied**: Ensure Zabbix user has access to Docker socket
2. **No Services Found**: Verify Docker Swarm is running and services exist
3. **Stack Not Detected**: Check that services have `com.docker.stack.namespace` labels

### Debug Commands

```bash
# Test Docker API access
curl --unix-socket /var/run/docker.sock http://localhost/v1.41/services

# Check Zabbix Agent logs
sudo tail -f /var/log/zabbix/zabbix_agent2.log

# Test specific metrics
zabbix_get -s localhost -k "swarm.services.discovery"
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Zabbix team for the excellent Agent 2 plugin framework
- Docker team for the comprehensive Swarm API
- Community contributors and testers
