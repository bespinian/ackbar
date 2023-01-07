package model

import (
	"time"

	"github.com/google/uuid"
)

type Info struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Model interface {
	GetID() uuid.UUID
}

type Context struct {
	ID                      uuid.UUID `json:"id"`
	Name                    string    `json:"name"`
	LivenessIntervalSeconds int       `json:"livenessIntervalSeconds"`
	MaxPartitionsPerWorker  int       `json:"maxPartitionsPerWorker"`
	PartitionToWorkerRatio  float64   `json:"partitionToWorkerRatio"`
}

func (c Context) GetID() uuid.UUID {
	return c.ID
}

type Partition struct {
	ID             uuid.UUID `json:"id"`
	ContextID      uuid.UUID `json:"contextID"`
	AssignedWorker uuid.UUID `json:"assignedWorker"`
	Configuration  string    `json:"configuration"`
}

func (c Partition) GetID() uuid.UUID {
	return c.ID
}

type Worker struct {
	ID                      uuid.UUID `json:"id"`
	ContextID               uuid.UUID `json:"contextID"`
	RegisteredAt            time.Time `json:"registeredAt"`
	RefreshedAt             time.Time `json:"refreshedAt"`
	PartitionConfigurations []string  `json:"partitionConfiguration"`
}

func (w Worker) GetID() uuid.UUID {
	return w.ID
}
