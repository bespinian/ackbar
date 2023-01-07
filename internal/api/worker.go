package api

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-http-utils/headers"
	"github.com/google/uuid"
)

var WorkersPath = fmt.Sprintf("%s/%s", ContextPath, "workers")
var WorkerIdPlaceholder = ":workerId"
var WorkerPath = fmt.Sprintf("%s/%s", WorkersPath, WorkerIdPlaceholder)

func (a *Api) PostWorker(c *gin.Context) {
	contextId := c.Param("contextId")
	uuid, _ := uuid.Parse(contextId)

	createdWorker, _ := a.Backend.RegisterWorker(uuid)
	c.Header(headers.Location, a.replaceIDPlaceholders(a.replaceIDPlaceholders(WorkerPath, ContextIdPlaceholder, uuid), WorkerIdPlaceholder, createdWorker.ID))
	c.IndentedJSON(http.StatusCreated, createdWorker)
}

func (a *Api) GetWorkers(c *gin.Context) {
	contextId := c.Param("contextId")
	uuid, _ := uuid.Parse(contextId)

	workers, _ := a.Backend.GetWorkers(uuid)
	c.IndentedJSON(http.StatusOK, workers)
}

func (a *Api) GetWorker(c *gin.Context) {
	contextId := c.Param("contextId")
	workerId := c.Param("workerId")
	contextUuid, _ := uuid.Parse(contextId)
	workerUuid, _ := uuid.Parse(workerId)

	worker, _, _ := a.Backend.GetWorker(contextUuid, workerUuid)
	c.IndentedJSON(http.StatusOK, worker)
}

func (a *Api) PutWorker(c *gin.Context) {
	contextId := c.Param("contextId")
	workerId := c.Param("workerId")
	contextUuid, _ := uuid.Parse(contextId)
	workerUuid, _ := uuid.Parse(workerId)

	refreshedWorker, _, _ := a.Backend.RefreshWorker(contextUuid, workerUuid)
	c.IndentedJSON(http.StatusOK, refreshedWorker)
}

func (a *Api) DeleteWorker(c *gin.Context) {
	contextId := c.Param("contextId")
	workerId := c.Param("workerId")
	contextUuid, _ := uuid.Parse(contextId)
	workerUuid, _ := uuid.Parse(workerId)
	err := a.Backend.DeleteWorker(contextUuid, workerUuid)
	if err != nil {
		log.Printf("Error deleting worker %s in context %s: %e", workerUuid, contextUuid, err)
	}
	c.Status(http.StatusOK)
}
