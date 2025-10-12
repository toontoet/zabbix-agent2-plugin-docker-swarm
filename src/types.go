package main

// Service represents a Docker Swarm service.
type Service struct {
	ID   string      `json:"ID"`
	Spec ServiceSpec `json:"Spec"`
}

// ServiceSpec represents the specification of a service.
type ServiceSpec struct {
	Name   string            `json:"Name"`
	Mode   ServiceMode       `json:"Mode"`
	Labels map[string]string `json:"Labels"`
}

// ServiceMode represents the mode of a service (replicated or global).
type ServiceMode struct {
	Replicated *ReplicatedService `json:"Replicated"`
	Global     *GlobalService     `json:"Global"`
}

// ReplicatedService is for a replicated service.
type ReplicatedService struct {
	Replicas *uint64 `json:"Replicas"`
}

// GlobalService is for a global service.
type GlobalService struct{}

// Task represents a task running as part of a service.
type Task struct {
	ID           string     `json:"ID"`
	ServiceID    string     `json:"ServiceID"`
	Status       TaskStatus `json:"Status"`
	DesiredState string     `json:"DesiredState"`
}

// Node represents a node running in the swarm cluster.
type Node struct {
	ID           string     `json:"ID"`
}

// TaskStatus represents the status of a task.
type TaskStatus struct {
	State           string               `json:"State"`
	Timestamp       string               `json:"Timestamp"`
	ContainerStatus *TaskContainerStatus `json:"ContainerStatus,omitempty"`
}

// TaskContainerStatus contains container-specific status information.
type TaskContainerStatus struct {
	ContainerID string `json:"ContainerID"`
	ExitCode    int    `json:"ExitCode"`
}

// StackHealth represents the health status of a Docker Compose stack.
type StackHealth struct {
	StackName         string  `json:"{#STACK.NAME}"`
	TotalServices     int     `json:"total_services"`
	HealthyServices   int     `json:"healthy_services"`
	UnhealthyServices int     `json:"unhealthy_services"`
	HealthPercentage  float64 `json:"health_percentage"`
}

// ErrorMessage represents the API error message from Docker.
type ErrorMessage struct {
	Message string `json:"message"`
}
