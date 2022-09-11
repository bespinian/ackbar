package api

import (
	"net/http"

	"github.com/bespinian/lando/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var contexts = []model.Context{
	{ID: uuid.New(), Name: "Sensors", LeaseTimeSeconds: 1},
}

func (a *Api) GetContexts(c *gin.Context) {
	contexts, _ := a.Backend.GetContexts()
	c.IndentedJSON(http.StatusOK, contexts)
}

func (a *Api) GetContext(c *gin.Context) {
	id := c.Param("contextId")
	uuid, _ := uuid.Parse(id)
	context, _, _ := a.Backend.GetContext(uuid)
	c.IndentedJSON(http.StatusOK, context)
}

func (a *Api) PostContext(c *gin.Context) {
	var newContext model.Context

	if err := c.BindJSON(&newContext); err != nil {
		return
	}

	createdContext, _ := a.Backend.CreateContext(newContext.Name, newContext.LeaseTimeSeconds, newContext.MaxLeasesPerPartition)
	c.IndentedJSON(http.StatusCreated, createdContext)
}

func (a *Api) DeleteContext(c *gin.Context) {
	id := c.Param("contextId")
	uuid, _ := uuid.Parse(id)
	a.Backend.DeleteContext(uuid)
	c.Status(http.StatusOK)
}
