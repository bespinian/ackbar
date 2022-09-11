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
