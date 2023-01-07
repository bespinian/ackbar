package backend

import (
	"github.com/bespinian/lando/internal/model"
	"github.com/google/uuid"
)

type Backend interface {
	GetContexts() ([]model.Context, error)
	GetContext(id uuid.UUID) (model.Context, bool, error)
	CreateContext(name string, livenessIntervalSeconds, maxPartitionsPerWorker int) (model.Context, error)
	DeleteContext(id uuid.UUID) error

	GetPartitions(contextId uuid.UUID) ([]model.Partition, error)
	AddPartition(contextId uuid.UUID, configuration string) (model.Partition, error)
	DeletePartition(contextId, partitionId uuid.UUID) error

	RegisterWorker(contextId uuid.UUID) (model.Worker, error)
	GetWorkers(contextId uuid.UUID) ([]model.Worker, error)
	GetWorker(contextId, workerId uuid.UUID) (model.Worker, bool, error)
	RefreshWorker(contextId, workerId uuid.UUID) (model.Worker, bool, error)
	DeleteWorker(contextId, workerId uuid.UUID) error
}
