package main

import (
	"os"

	"github.com/bespinian/lando/internal/api"
	"github.com/bespinian/lando/internal/backend"
	"github.com/gin-gonic/gin"
)

func main() {
	redisBackend := backend.RedisBackend{}
	redisBackend.Initialize(os.Getenv("REDIS_CONNECTION_STRING"), "", 0)
	api := api.Api{Backend: &redisBackend}

	router := gin.Default()
	router.GET("/", api.GetInfo)
	router.GET("/contexts", api.GetContexts)
	router.POST("/contexts", api.PostContext)
	router.GET("/contexts/:contextId", api.GetContext)
	router.DELETE("/contexts/:contextId", api.DeleteContext)
	router.GET("/contexts/:contextId/partitions", api.GetPartitions)
	router.POST("/contexts/:contextId/partitions", api.PostPartition)
	router.DELETE("/contexts/:contextId/partitions/:partitionId", api.DeletePartition)

	router.POST("/contexts/:contextId/leases", api.PostLease)
	router.PUT("/contexts/:contextId/partitions/:partitionId/leases/:leaseId", api.PutLease)
	router.DELETE("/contexts/:contextId/partitions/:partitionId/leases/:leaseId", api.DeleteLease)
	router.Run("localhost:8080")
}
