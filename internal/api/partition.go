package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (a *Api) GetPartitions(c *gin.Context) {
	contextId := c.Param("contextId")
	uuid, _ := uuid.Parse(contextId)
	partitions, _ := a.Backend.GetPartitions(uuid)
	c.IndentedJSON(http.StatusOK, partitions)
}

func (a *Api) PostPartition(c *gin.Context) {
	contextId := c.Param("contextId")
	uuid, _ := uuid.Parse(contextId)

	createdPartition, _ := a.Backend.AddPartition(uuid)
	c.IndentedJSON(http.StatusCreated, createdPartition)
}

func (a *Api) DeletePartition(c *gin.Context) {
	contextId := c.Param("contextId")
	partitionId := c.Param("partitionId")
	contextUuid, _ := uuid.Parse(contextId)
	partitionUuid, _ := uuid.Parse(partitionId)
	a.Backend.DeletePartition(contextUuid, partitionUuid)
	c.Status(http.StatusOK)
}
