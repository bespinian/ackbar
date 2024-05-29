package api

import (
	"fmt"
	"log"
	"net/http"

	"github.com/bespinian/ackbar/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/go-http-utils/headers"
	"github.com/google/uuid"
)

var (
	PartitionsPath         = fmt.Sprintf("%s/%s", ContextPath, "partitions")
	PartitionIdPlaceholder = ":partitionId"
	PartitionPath          = fmt.Sprintf("%s/%s", PartitionsPath, PartitionIdPlaceholder)
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

	var newPartition model.Partition

	if err := c.BindJSON(&newPartition); err != nil {
		return
	}
	createdPartition, _ := a.Backend.AddPartition(uuid, newPartition.Configuration)
	c.Header(headers.Location, a.replaceIDPlaceholders(a.replaceIDPlaceholders(PartitionPath, ContextIdPlaceholder, uuid), PartitionIdPlaceholder, createdPartition.ID))
	c.IndentedJSON(http.StatusCreated, createdPartition)
}

func (a *Api) DeletePartition(c *gin.Context) {
	contextId := c.Param("contextId")
	partitionId := c.Param("partitionId")
	contextUuid, _ := uuid.Parse(contextId)
	partitionUuid, _ := uuid.Parse(partitionId)
	err := a.Backend.DeletePartition(contextUuid, partitionUuid)
	if err != nil {
		log.Printf("Error deleting partition %s in context %s: %e", partitionUuid, contextUuid, err)
	}
	c.Status(http.StatusOK)
}
