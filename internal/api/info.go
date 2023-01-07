package api

import (
	"net/http"
	"strings"

	"github.com/bespinian/lando/internal/backend"
	"github.com/bespinian/lando/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Api struct {
	Backend backend.Backend
}

var version = model.Info{Name: "lando", Version: "0.0.1"}

func (a *Api) GetInfo(c *gin.Context) {
	c.IndentedJSON(http.StatusOK, version)
}

func (a *Api) replaceIDPlaceholders(path, placeholder string, id uuid.UUID) string {
	return strings.Replace(path, placeholder, id.String(), -1)
}
