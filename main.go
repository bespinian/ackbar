package main

import (
	"log"
	"os"

	apimodule "github.com/bespinian/lando/internal/api"
	"github.com/bespinian/lando/internal/backend"
	"github.com/gin-gonic/gin"
)

func main() {
	redisBackend := backend.RedisBackend{}
	redisBackend.Initialize(os.Getenv("REDIS_CONNECTION_STRING"), "", 0)
	api := apimodule.Api{Backend: &redisBackend}

	router := gin.Default()
	router.GET(apimodule.RootPath, api.GetInfo)
	router.GET(apimodule.ContextsPath, api.GetContexts)
	router.POST(apimodule.ContextsPath, api.PostContext)
	router.GET(apimodule.ContextPath, api.GetContext)
	router.DELETE(apimodule.ContextPath, api.DeleteContext)
	router.GET(apimodule.PartitionsPath, api.GetPartitions)
	router.POST(apimodule.PartitionsPath, api.PostPartition)
	router.DELETE(apimodule.PartitionPath, api.DeletePartition)

	router.POST(apimodule.WorkersPath, api.PostWorker)
	router.GET(apimodule.WorkersPath, api.GetWorkers)
	router.PUT(apimodule.WorkerPath, api.PutWorker)
	router.DELETE(apimodule.WorkerPath, api.DeleteWorker)
	err := router.Run("localhost:8080")
	if err != nil {
		log.Fatalf("Error running http router: %e", err)
	}
}
