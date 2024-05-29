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
	RootPath             = "/"
	ContextsPath         = fmt.Sprintf("%s%s", RootPath, "contexts")
	ContextIdPlaceholder = ":contextId"
	ContextPath          = fmt.Sprintf("%s/%s", ContextsPath, ContextIdPlaceholder)
)

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

	createdContext, _ := a.Backend.CreateContext(newContext.Name, newContext.LivenessIntervalSeconds, newContext.MaxPartitionsPerWorker)
	c.Header(headers.Location, a.replaceIDPlaceholders(ContextPath, ContextIdPlaceholder, createdContext.ID))
	c.IndentedJSON(http.StatusCreated, createdContext)
}

func (a *Api) DeleteContext(c *gin.Context) {
	id := c.Param("contextId")
	uuid, _ := uuid.Parse(id)
	err := a.Backend.DeleteContext(uuid)
	if err != nil {
		log.Printf("Error deleting context %s: %e", uuid, err)
	}
	c.Status(http.StatusOK)
}
