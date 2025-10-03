# Zabbix Docker Swarm Plugin - Usage Examples

This document provides practical examples of how to use the Docker Swarm plugin features.

## Testing the Plugin

### 1. Service Discovery

Test basic service discovery to see all services with their stack information:

```bash
zabbix_get -s localhost -k "swarm.services.discovery"
```

Expected output (formatted for readability):

```json
[
  {
    "{#SERVICE.ID}": "abc123def456",
    "{#SERVICE.NAME}": "web-frontend",
    "{#STACK.NAME}": "myapp"
  },
  {
    "{#SERVICE.ID}": "def456ghi789", 
    "{#SERVICE.NAME}": "standalone-nginx",
    "{#STACK.NAME}": "standalone"
  }
]
```

### 2. Stack Discovery

Discover all Docker Compose stacks:

```bash
zabbix_get -s localhost -k "swarm.stacks.discovery"
```

Expected output:

```json
[
  {"{#STACK.NAME}": "myapp"},
  {"{#STACK.NAME}": "monitoring"},
  {"{#STACK.NAME}": "standalone"}
]
```

### 3. Stack Health Monitoring

Get health status for a specific stack:

```bash
zabbix_get -s localhost -k "swarm.stack.health[myapp]"
```

Expected output:

```json
{
  "total_services": 3,
  "healthy_services": 3,
  "unhealthy_services": 0,
  "health_percentage": 100.0
}
```

### 4. Individual Service Metrics

Get metrics for individual services using flexible identifiers:

**Service Identifier Types:**
- **Service Name**: `web`, `database` (simple and readable)
- **Service Key**: `mystack_web`, `mystack_database` (stable across redeploys)
- **Service ID**: `abc123def456...` (Docker's internal ID)

```bash
# Get desired replica count (using service name)
zabbix_get -s localhost -k "swarm.service.replicas_desired[web]"

# Get running replica count (using service key for stack service)
zabbix_get -s localhost -k "swarm.service.replicas_running[mystack_web]"

# Get restart count (using service ID - all methods work)
zabbix_get -s localhost -k "swarm.service.restarts[abc123def456]"
```

## Docker Compose Stack Example

To test stack functionality, create a simple Docker Compose stack:

### docker-compose.yml

```yaml
version: '3.8'
services:
  web:
    image: nginx:alpine
    deploy:
      replicas: 2
    ports:
      - "80:80"
      
  api:
    image: httpd:alpine
    deploy:
      replicas: 1
    ports:
      - "8080:80"
      
  worker:
    image: alpine:latest
    command: sleep infinity
    deploy:
      replicas: 3
```

### Deploy the Stack

```bash
# Deploy the stack
docker stack deploy -c docker-compose.yml teststack

# Verify deployment
docker service ls
docker stack ps teststack
```

### Test Stack Monitoring

```bash
# Discover the new stack
zabbix_get -s localhost -k "swarm.stacks.discovery"

# Check stack health
zabbix_get -s localhost -k "swarm.stack.health[teststack]"

# Test individual services
SERVICE_ID=$(docker service ls --filter name=teststack_web --format "{{.ID}}")
zabbix_get -s localhost -k "swarm.service.replicas_desired[$SERVICE_ID]"
zabbix_get -s localhost -k "swarm.service.replicas_running[$SERVICE_ID]"
zabbix_get -s localhost -k "swarm.service.restarts[$SERVICE_ID]"
```

## Zabbix Template Import

### Create Template Items

1. **Stack Discovery Rule**:
   - Name: `Docker Swarm Stacks`
   - Key: `swarm.stacks.discovery`
   - Update interval: `600s`

2. **Stack Health Item Prototype**:
   - Name: `Stack {#STACK.NAME} health`
   - Key: `swarm.stack.health[{#STACK.NAME}]`
   - Type: `Zabbix agent`
   - Value type: `Text`

3. **Calculated Items for Stack Health**:

   ```text
   # Health Percentage
   Name: Stack {#STACK.NAME} health percentage
   Formula: jsonpath(last(/Template/swarm.stack.health[{#STACK.NAME}]),"$.health_percentage")
   Units: %
   
   # Unhealthy Services Count
   Name: Stack {#STACK.NAME} unhealthy services
   Formula: jsonpath(last(/Template/swarm.stack.health[{#STACK.NAME}]),"$.unhealthy_services")
   ```

### Trigger Examples

```text
# Critical: Stack has unhealthy services
Name: Stack {#STACK.NAME} has unhealthy services
Expression: jsonpath(last(/Template/swarm.stack.health[{#STACK.NAME}]),"$.unhealthy_services")>0
Severity: High

# Warning: Stack health not 100%
Name: Stack {#STACK.NAME} health degraded
Expression: jsonpath(last(/Template/swarm.stack.health[{#STACK.NAME}]),"$.health_percentage")<100
Severity: Warning
```

## Monitoring Scenarios

### Scenario 1: Service Scaling

When you scale a service, the plugin detects the change:

```bash
# Scale up web service
docker service scale teststack_web=5

# Check updated metrics
zabbix_get -s localhost -k "swarm.stack.health[teststack]"
```

### Scenario 2: Service Failure Simulation

Simulate a service failure to test alerting:

```bash
# Force stop some tasks
docker service update --replicas 1 teststack_web

# Monitor health degradation
zabbix_get -s localhost -k "swarm.stack.health[teststack]"
```

### Scenario 3: Stack vs Standalone Services

Compare stack services with standalone services:

```bash
# Create standalone service
docker service create --name standalone-redis redis:alpine

# Compare discovery output
zabbix_get -s localhost -k "swarm.services.discovery"
```

### Scenario 4: Monitor Service Restarts

Test restart detection when a service crashes:

```bash
# Get a service ID
SERVICE_ID=$(docker service ls --filter name=teststack_web --format "{{.ID}}")

# Check initial restart count
echo "Initial restart count:"
zabbix_get -s localhost -k "swarm.service.restarts[$SERVICE_ID]"

# Simulate a crash by updating the service with a failing command
docker service update --entrypoint '["sh","-c","exit 1"]' teststack_web

# Wait a few seconds for Docker to restart the task
sleep 10

# Check updated restart count
echo "Restart count after crash:"
zabbix_get -s localhost -k "swarm.service.restarts[$SERVICE_ID]"

# Restore the service
docker service update --entrypoint '["nginx","-g","daemon off;"]' teststack_web

# In Zabbix, configure a trigger with:
# Expression: change(/YourHost/swarm.service.restarts[{#SERVICE.KEY}])>0
# This will fire whenever the restart count increases
```

### Scenario 5: Stable Service Monitoring Across Redeploys

**Problem**: Service IDs change when you redeploy a stack, breaking Zabbix monitoring.

**Solution**: Use service keys for stable monitoring:

```bash
# Before redeploy - check service discovery
zabbix_get -s localhost -k "swarm.services.discovery"

# Example output shows service keys:
# [{"{#SERVICE.ID}":"abc123","{#SERVICE.NAME}":"web","{#STACK.NAME}":"mystack","{#SERVICE.KEY}":"mystack_web"}]

# Monitor using stable service key (survives redeploys)
zabbix_get -s localhost -k "swarm.service.replicas_running[mystack_web]"
zabbix_get -s localhost -k "swarm.service.restarts[mystack_web]"

# Redeploy the stack
docker stack deploy -c docker-compose.yml mystack

# Service ID changed, but service key remains the same!
# Monitoring continues without interruption
zabbix_get -s localhost -k "swarm.service.replicas_running[mystack_web]"
```

**Zabbix Template Configuration:**
- Use `{#SERVICE.KEY}` instead of `{#SERVICE.ID}` in item keys
- Service keys are stable across stack redeploys
- Monitoring items won't disappear when services are updated

## Troubleshooting Examples

### Debug Stack Labels

If services don't appear in the correct stack, check their labels:

```bash
# Inspect service labels
docker service inspect teststack_web --format '{{json .Spec.Labels}}'

# Should show: {"com.docker.stack.namespace":"teststack"}
```

### Verify Plugin Installation

```bash
# Test basic connectivity
zabbix_get -s localhost -k "agent.ping"

# Test plugin loading
zabbix_get -s localhost -k "swarm.services.discovery"

# Check agent logs
sudo tail -f /var/log/zabbix/zabbix_agent2.log
```

This should give you a comprehensive foundation for testing and implementing the Docker Swarm plugin with stack monitoring capabilities.
