package api

import (
	"net/http"

	"github.com/bespinian/lando/internal/backend"
	"github.com/bespinian/lando/internal/model"
	"github.com/gin-gonic/gin"
)

type Api struct {
	Backend backend.Backend
}

var version = model.Info{Name: "lando", Version: "0.0.1"}

func (a *Api) GetInfo(c *gin.Context) {
	c.IndentedJSON(http.StatusOK, version)
}
