package backend

import (
	"github.com/bespinian/lando/internal/model"
	"github.com/google/uuid"
)

type Backend interface {
	GetContexts() ([]model.Context, error)
	GetContext(id uuid.UUID) (model.Context, bool, error)
	CreateContext(name string, leaseTimeSeconds, maxLeasesPerPartition int) (model.Context, error)
	DeleteContext(id uuid.UUID) error

	GetPartitions(contextId uuid.UUID) ([]model.Partition, error)
	AddPartition(contextId uuid.UUID) (model.Partition, error)
	DeletePartition(contextId, partitionId uuid.UUID) error

	CreateLease(contextId uuid.UUID) (model.Lease, bool, error)
	RefreshLease(contextId, partitionId, leaseId uuid.UUID) (model.Lease, bool, error)
	DeleteLease(contextId, partitionId, leaseId uuid.UUID) error
}
