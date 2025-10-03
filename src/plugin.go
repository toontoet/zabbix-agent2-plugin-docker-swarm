/*
** Copyright (C) 2005 Toon Toetenel
**
** Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated
** documentation files (the "Software"), to deal in the Software without restriction, including without limitation the
** rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to
** permit persons to whom the Software is furnished to do so, subject to the following conditions:
**
** The above copyright notice and this permission notice shall be included in all copies or substantial portions
** of the Software.
**
** THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE
** WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
** COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
** TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
** SOFTWARE.
**/

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"golang.zabbix.com/sdk/errs"
	"golang.zabbix.com/sdk/metric"
	"golang.zabbix.com/sdk/plugin"
	"golang.zabbix.com/sdk/plugin/container"
)

const (
	// Name of the plugin.
	Name = "DockerSwarm"

	serviceDiscoveryMetric = swarmMetricKey("swarm.services.discovery")
	serviceReplicasDesired = swarmMetricKey("swarm.service.replicas_desired")
	serviceReplicasRunning = swarmMetricKey("swarm.service.replicas_running")
	serviceRestartCount    = swarmMetricKey("swarm.service.restarts")
	stackDiscoveryMetric   = swarmMetricKey("swarm.stacks.discovery")
	stackHealthMetric      = swarmMetricKey("swarm.stack.health")
)

var (
	_ plugin.Exporter = (*swarmPlugin)(nil)
	_ plugin.Runner   = (*swarmPlugin)(nil)
)

type swarmMetricKey string

type swarmMetric struct {
	metric  *metric.Metric
	handler func(ctx context.Context, params []string) (any, error)
}

type swarmPlugin struct {
	plugin.Base
	client  *client
	metrics map[swarmMetricKey]*swarmMetric
}

// Launch launches the DockerSwarm plugin. Blocks until plugin execution has finished.
func Launch() error {
	p := &swarmPlugin{
		client: newClient("/var/run/docker.sock", 30),
	}

	err := p.registerMetrics()
	if err != nil {
		return err
	}

	h, err := container.NewHandler(Name)
	if err != nil {
		return errs.Wrap(err, "failed to create new handler")
	}

	p.Logger = h

	err = h.Execute()
	if err != nil {
		return errs.Wrap(err, "failed to execute plugin handler")
	}

	return nil
}

// Start starts the Docker Swarm plugin. Required for plugin to match runner interface.
func (p *swarmPlugin) Start() {
	p.Infof("DockerSwarm plugin started")
}

// Stop stops the Docker Swarm plugin. Required for plugin to match runner interface.
func (p *swarmPlugin) Stop() {
	p.Infof("DockerSwarm plugin stopped")
}

// Export collects all the metrics.
func (p *swarmPlugin) Export(key string, rawParams []string, _ plugin.ContextProvider) (any, error) {
	m, ok := p.metrics[swarmMetricKey(key)]
	if !ok {
		return nil, errs.New("unknown metric " + key)
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		30*time.Second,
	)
	defer cancel()

	res, err := m.handler(ctx, rawParams)
	if err != nil {
		return nil, errs.Wrap(err, "failed to execute handler")
	}

	return res, nil
}

func (p *swarmPlugin) registerMetrics() error {
	p.metrics = map[swarmMetricKey]*swarmMetric{
		serviceDiscoveryMetric: {
			metric: metric.New(
				"Discover Docker Swarm services with stack information.",
				nil,
				false,
			),
			handler: p.discoverServices,
		},
		serviceReplicasDesired: {
			metric: metric.New(
				"Returns the desired number of replicas for a service.",
				nil,
				false,
			),
			handler: p.getDesiredReplicas,
		},
		serviceReplicasRunning: {
			metric: metric.New(
				"Returns the number of running tasks for a service.",
				nil,
				false,
			),
			handler: p.getRunningTasks,
		},
		serviceRestartCount: {
			metric: metric.New(
				"Returns the number of task restarts for a service.",
				nil,
				false,
			),
			handler: p.getServiceRestarts,
		},
		stackDiscoveryMetric: {
			metric: metric.New(
				"Discover Docker Compose stacks.",
				nil,
				false,
			),
			handler: p.discoverStacks,
		},
		stackHealthMetric: {
			metric: metric.New(
				"Returns health status for a Docker Compose stack.",
				nil,
				false,
			),
			handler: p.getStackHealth,
		},
	}

	metricSet := metric.MetricSet{}

	for k, m := range p.metrics {
		metricSet[string(k)] = m.metric
	}

	err := plugin.RegisterMetrics(p, Name, metricSet.List()...)
	if err != nil {
		return errs.Wrap(err, "failed to register metrics")
	}

	return nil
}

func (p *swarmPlugin) getServices() ([]Service, error) {
	body, err := p.client.Query("services", nil)
	if err != nil {
		return nil, err
	}

	var services []Service
	if err = json.Unmarshal(body, &services); err != nil {
		return nil, errs.Wrap(err, "cannot unmarshal JSON")
	}

	return services, nil
}

func (p *swarmPlugin) discoverServices(_ context.Context, params []string) (any, error) {
	if len(params) != 0 {
		return nil, errs.New("expected no parameters for service discovery")
	}

	services, err := p.getServices()
	if err != nil {
		return nil, err
	}

	type LLDService struct {
		ID        string `json:"{#SERVICE.ID}"`
		Name      string `json:"{#SERVICE.NAME}"`
		StackName string `json:"{#STACK.NAME}"`
	}

	lldServices := make([]LLDService, 0, len(services))
	for _, s := range services {
		stackName := "standalone"
		if s.Spec.Labels != nil {
			if namespace, exists := s.Spec.Labels["com.docker.stack.namespace"]; exists {
				stackName = namespace
			}
		}

		lldServices = append(lldServices, LLDService{
			ID:        s.ID,
			Name:      s.Spec.Name,
			StackName: stackName,
		})
	}

	jsonData, err := json.Marshal(lldServices)
	if err != nil {
		return nil, errs.Wrap(err, "cannot marshal JSON")
	}

	return string(jsonData), nil
}

func (p *swarmPlugin) discoverStacks(_ context.Context, params []string) (any, error) {
	if len(params) != 0 {
		return nil, errs.New("expected no parameters for stack discovery")
	}

	services, err := p.getServices()
	if err != nil {
		return nil, err
	}

	stacksMap := make(map[string]bool)
	for _, s := range services {
		stackName := "standalone"
		if s.Spec.Labels != nil {
			if namespace, exists := s.Spec.Labels["com.docker.stack.namespace"]; exists {
				stackName = namespace
			}
		}
		stacksMap[stackName] = true
	}

	type LLDStack struct {
		StackName string `json:"{#STACK.NAME}"`
	}

	lldStacks := make([]LLDStack, 0, len(stacksMap))
	for stackName := range stacksMap {
		lldStacks = append(lldStacks, LLDStack{StackName: stackName})
	}

	jsonData, err := json.Marshal(lldStacks)
	if err != nil {
		return nil, errs.Wrap(err, "cannot marshal JSON")
	}

	return string(jsonData), nil
}

func (p *swarmPlugin) getStackHealth(_ context.Context, params []string) (any, error) {
	if len(params) != 1 {
		return nil, errs.New("expected 1 parameter for stack health")
	}

	stackName := params[0]
	services, err := p.getServices()
	if err != nil {
		return nil, err
	}

	// Filter services for this stack
	var stackServices []Service
	for _, s := range services {
		serviceStackName := "standalone"
		if s.Spec.Labels != nil {
			if namespace, exists := s.Spec.Labels["com.docker.stack.namespace"]; exists {
				serviceStackName = namespace
			}
		}
		if serviceStackName == stackName {
			stackServices = append(stackServices, s)
		}
	}

	if len(stackServices) == 0 {
		return nil, errs.New("stack not found: " + stackName)
	}

	totalServices := len(stackServices)
	healthyServices := 0

	// Check health of each service
	for _, service := range stackServices {
		desired, dErr := p.getServiceDesiredReplicas(service)
		if dErr != nil {
			continue // Skip services we can't evaluate
		}

		running, rErr := p.getServiceRunningTasks(service.ID)
		if rErr != nil {
			continue // Skip services we can't evaluate
		}

		if running >= desired {
			healthyServices++
		}
	}

	unhealthyServices := totalServices - healthyServices
	healthPercentage := float64(healthyServices) / float64(totalServices) * 100

	result := map[string]interface{}{
		"total_services":     totalServices,
		"healthy_services":   healthyServices,
		"unhealthy_services": unhealthyServices,
		"health_percentage":  healthPercentage,
	}

	jsonData, err := json.Marshal(result)
	if err != nil {
		return nil, errs.Wrap(err, "cannot marshal JSON")
	}

	return string(jsonData), nil
}

func (p *swarmPlugin) getDesiredReplicas(_ context.Context, params []string) (any, error) {
	if len(params) != 1 {
		return nil, errs.New("expected 1 parameter for desired replicas")
	}

	serviceID := params[0]
	body, err := p.client.Query(fmt.Sprintf("services/%s", serviceID), nil)
	if err != nil {
		return nil, err
	}

	var service Service
	if err = json.Unmarshal(body, &service); err != nil {
		return nil, errs.Wrap(err, "cannot unmarshal JSON")
	}

	return p.getServiceDesiredReplicas(service)
}

func (p *swarmPlugin) getServiceDesiredReplicas(service Service) (int, error) {
	if service.Spec.Mode.Replicated != nil && service.Spec.Mode.Replicated.Replicas != nil {
		replicas := *service.Spec.Mode.Replicated.Replicas
		// #nosec G115 - Docker Swarm replica counts are reasonable values, overflow extremely unlikely
		return int(replicas), nil
	}

	if service.Spec.Mode.Global != nil {
		// For global services, return 1 as a placeholder
		// This could be enhanced to count actual nodes
		return 1, nil
	}

	return 0, errs.New("could not determine desired replicas for service " + service.ID)
}

func (p *swarmPlugin) getRunningTasks(_ context.Context, params []string) (any, error) {
	if len(params) != 1 {
		return nil, errs.New("expected 1 parameter for running tasks")
	}

	serviceID := params[0]
	return p.getServiceRunningTasks(serviceID)
}

func (p *swarmPlugin) getServiceRunningTasks(serviceID string) (int, error) {
	filters := map[string][]string{
		"service":       {serviceID},
		"desired-state": {"running"},
	}

	body, err := p.client.Query("tasks", filters)
	if err != nil {
		return 0, err
	}

	var tasks []Task
	if err = json.Unmarshal(body, &tasks); err != nil {
		return 0, errs.Wrap(err, "cannot unmarshal JSON")
	}

	count := 0
	for _, task := range tasks {
		if task.Status.State == "running" {
			count++
		}
	}

	return count, nil
}

func (p *swarmPlugin) getServiceRestarts(_ context.Context, params []string) (any, error) {
	if len(params) != 1 {
		return nil, errs.New("expected 1 parameter for service restarts")
	}

	serviceID := params[0]

	// Get all tasks for the service (not just running ones)
	filters := map[string][]string{
		"service": {serviceID},
	}

	body, err := p.client.Query("tasks", filters)
	if err != nil {
		return 0, err
	}

	var tasks []Task
	if err = json.Unmarshal(body, &tasks); err != nil {
		return 0, errs.Wrap(err, "cannot unmarshal JSON")
	}

	// Count tasks that have failed/shutdown state with exit code != 0
	// This indicates the container crashed and was restarted
	restartCount := 0
	for _, task := range tasks {
		// Count tasks that were shutdown/failed with non-zero exit code
		// These indicate restarts due to crashes
		if task.Status.State == "failed" || task.Status.State == "shutdown" {
			if task.Status.ContainerStatus != nil && task.Status.ContainerStatus.ExitCode != 0 {
				restartCount++
			}
		}
	}

	return restartCount, nil
}
