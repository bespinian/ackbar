package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (a *Api) PostLease(c *gin.Context) {
	contextId := c.Param("contextId")
	uuid, _ := uuid.Parse(contextId)

	createdLease, _, _ := a.Backend.CreateLease(uuid)
	c.IndentedJSON(http.StatusCreated, createdLease)
}

func (a *Api) PutLease(c *gin.Context) {
	contextId := c.Param("contextId")
	partitionId := c.Param("partitionId")
	leaseId := c.Param("leaseId")
	contextUuid, _ := uuid.Parse(contextId)
	partitionUuid, _ := uuid.Parse(partitionId)
	leaseUuid, _ := uuid.Parse(leaseId)

	refreshedLease, _, _ := a.Backend.RefreshLease(contextUuid, partitionUuid, leaseUuid)
	c.IndentedJSON(http.StatusCreated, refreshedLease)
}

func (a *Api) DeleteLease(c *gin.Context) {
	contextId := c.Param("contextId")
	partitionId := c.Param("partitionId")
	leaseId := c.Param("leaseId")
	contextUuid, _ := uuid.Parse(contextId)
	partitionUuid, _ := uuid.Parse(partitionId)
	leaseUuid, _ := uuid.Parse(leaseId)
	a.Backend.DeleteLease(contextUuid, partitionUuid, leaseUuid)
	c.Status(http.StatusOK)
}
