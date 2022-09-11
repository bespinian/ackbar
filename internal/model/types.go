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
	ID                    uuid.UUID `json:"id"`
	Name                  string    `json:"name"`
	LeaseTimeSeconds      int       `json:"leaseTimeSeconds"`
	MaxLeasesPerPartition int       `json:"maxLeasesPerPartition"`
}

func (c Context) GetID() uuid.UUID {
	return c.ID
}

type Partition struct {
	ID        uuid.UUID `json:"id"`
	ContextID uuid.UUID `json:"contextID"`
	Leases    []Lease
}

func (c Partition) GetID() uuid.UUID {
	return c.ID
}

type Lease struct {
	ID          uuid.UUID `json:"id"`
	ContextID   uuid.UUID `json:"contextID"`
	PartitionID uuid.UUID `json:"partitionID"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (c Lease) GetID() uuid.UUID {
	return c.ID
}
