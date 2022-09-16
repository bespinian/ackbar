package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bespinian/lando/internal/model"
	redis "github.com/go-redis/redis/v9"
	"github.com/google/uuid"
)

const contextsKeyIndexName = "contexts:index"
const contextKeyPattern = "contexts:%s"
const partitionsKeyIndexName = "partitions:index"
const partitionKeyPattern = "contexts:%s:partitions:%s"
const leasesKeyIndexName = "leases:index"
const leaseKeyPattern = "contexts:%s:partitions:%s:leases:%s"

var ctx = context.Background()

type RedisBackend struct {
	redis *redis.Client
}

func (b *RedisBackend) Initialize(address, password string, db int) {
	b.redis = redis.NewClient(&redis.Options{
		Addr:     address,
		Password: password,
		DB:       db,
	})
}

func (b *RedisBackend) GetContexts() ([]model.Context, error) {
	return getModelList[model.Context](contextsKeyIndexName, b, "")
}

func (b *RedisBackend) GetContext(id uuid.UUID) (model.Context, bool, error) {
	return getModel[model.Context](contextKeyPattern, b, id)
}

func (b *RedisBackend) CreateContext(name string, leaseTimeSeconds, maxLeasesPerPartition int) (model.Context, error) {
	context := model.Context{ID: uuid.New(), Name: name, LeaseTimeSeconds: leaseTimeSeconds, MaxLeasesPerPartition: maxLeasesPerPartition}
	return createModel(context, contextsKeyIndexName, contextKeyPattern, 0, b, context.ID)
}

func (b *RedisBackend) DeleteContext(id uuid.UUID) error {
	partitions, err := b.GetPartitions(id)
	if err != nil {
		return err
	}
	for _, partition := range partitions {
		b.DeletePartition(id, partition.ID)
	}
	return deleteModel(contextsKeyIndexName, contextKeyPattern, b, id)
}

func (b *RedisBackend) GetPartitions(contextId uuid.UUID) ([]model.Partition, error) {
	result := []model.Partition{}
	partitions, err := getModelList[model.Partition](partitionsKeyIndexName, b, "")
	if err != nil {
		return partitions, err
	}
	for _, partition := range partitions {
		leaseKeyPrefix := fmt.Sprintf(partitionKeyPattern, partition.ContextID, partition.GetID())
		leases, err := getModelList[model.Lease](leasesKeyIndexName, b, leaseKeyPrefix)
		if err != nil {
			log.Println(fmt.Sprintf("error getting leases with key prefix %s: %s", leaseKeyPrefix, err))
			return partitions, err
		}
		log.Println(fmt.Sprintf("%d leases found", len(leases)))
		partition.Leases = append(partition.Leases, leases...)
		result = append(result, partition)
	}
	return result, nil
}

func (b *RedisBackend) AddPartition(contextId uuid.UUID) (model.Partition, error) {
	partition := model.Partition{ID: uuid.New(), ContextID: contextId}
	return createModel(partition, partitionsKeyIndexName, partitionKeyPattern, 0, b, contextId, partition.ID)
}

func (b *RedisBackend) DeletePartition(contextId, partitionId uuid.UUID) error {
	leaseKeyPrefix := fmt.Sprintf(partitionKeyPattern, contextId, partitionId)
	leases, err := getModelList[model.Lease](leasesKeyIndexName, b, leaseKeyPrefix)
	if err != nil {
		return err
	}
	for _, lease := range leases {
		deleteModel(leasesKeyIndexName, leaseKeyPattern, b, contextId, partitionId, lease.ID)
	}
	return deleteModel(partitionsKeyIndexName, partitionKeyPattern, b, contextId, partitionId)
}

func (b *RedisBackend) CreateLease(contextId uuid.UUID) (model.Lease, bool, error) {
	context, _, err := b.GetContext(contextId)
	if err != nil {
		log.Println(fmt.Sprintf("error getting context with id %s", contextId))
		return model.Lease{}, false, err
	}
	partitions, err := b.GetPartitions(contextId)
	if err != nil {
		log.Println(fmt.Sprintf("error getting partitions for context with id %s", contextId))
		return model.Lease{}, false, err
	}
	var createdLease model.Lease
	created := false
	for _, partition := range partitions {
		if len(partition.Leases) < context.MaxLeasesPerPartition {
			now := time.Now()
			lease := model.Lease{ID: uuid.New(), ContextID: contextId, PartitionID: partition.ID, CreatedAt: now, UpdatedAt: now}
			createdLease, err = createModel(lease, leasesKeyIndexName, leaseKeyPattern, context.LeaseTimeSeconds, b, contextId, partition.ID, lease.ID)
			if err != nil {
				return model.Lease{}, false, err
			}
			created = true
			break
		}
	}
	return createdLease, created, nil
}

func (b *RedisBackend) RefreshLease(contextId, partitionId, leaseId uuid.UUID) (model.Lease, bool, error) {
	context, _, err := b.GetContext(contextId)
	if err != nil {
		return model.Lease{}, false, err
	}
	lease, found, err := getModel[model.Lease](leaseKeyPattern, b, contextId, partitionId, leaseId)
	if err != nil {
		return model.Lease{}, false, err
	}
	var refreshedLease model.Lease
	refreshed := false
	if found {
		lease.UpdatedAt = time.Now()
		refreshedLease, err = createModel(lease, leasesKeyIndexName, leaseKeyPattern, context.LeaseTimeSeconds, b, contextId, lease.PartitionID, lease.ID)
		if err != nil {
			return model.Lease{}, false, err
		}
	} else {
		refreshedLease, refreshed, err = b.CreateLease(contextId)
		if err != nil {
			return model.Lease{}, false, err
		}
	}
	return refreshedLease, refreshed, nil
}

func (b *RedisBackend) DeleteLease(contextId, partitionId, leaseId uuid.UUID) error {
	return deleteModel(leasesKeyIndexName, leaseKeyPattern, b, contextId, partitionId, leaseId)
}

func getModelList[M model.Model](indexName string, b *RedisBackend, keyPrefix string) ([]M, error) {
	result := []M{}
	keys, err := b.redis.SMembers(ctx, indexName).Result()
	if err != nil {
		keys = []string{}
	}
	for _, key := range keys {
		if strings.HasPrefix(key, keyPrefix) {
			stringRep, err := b.redis.Get(ctx, key).Result()
			if err != nil {
				log.Println(fmt.Sprintf("removing stale key %s from index %s", key, indexName))
				b.redis.SRem(ctx, indexName, key)
			} else {
				model, err := stringToModel[M](stringRep)
				if err != nil {
					return result, err
				}
				result = append(result, model)
			}
		}
	}
	return result, nil
}

func getModel[M model.Model](keyPattern string, b *RedisBackend, ids ...uuid.UUID) (M, bool, error) {
	var result M
	idStrings := uuidsToStrings(ids)
	stringRep, err := b.redis.Get(ctx, fmt.Sprintf(keyPattern, idStrings...)).Result()
	if err != nil {
		log.Println(fmt.Sprintf("error getting model with key %s", fmt.Sprintf(keyPattern, idStrings...)))
		return result, false, err
	}
	result, err = stringToModel[M](stringRep)
	if err != nil {
		return result, false, err
	}
	return result, true, nil

}

func createModel[M model.Model](model M, indexName, keyPattern string, ttlSeconds int, b *RedisBackend, ids ...uuid.UUID) (M, error) {
	idStrings := uuidsToStrings(ids)
	key := fmt.Sprintf(keyPattern, idStrings...)
	stringRep, err := modelToStr(model)
	if err != nil {
		log.Println("error when transforming model to string")
		return model, err
	}
	b.redis.SAdd(ctx, indexName, key)
	b.redis.Set(ctx, key, stringRep, time.Duration(ttlSeconds)*time.Second)
	return model, nil
}

func deleteModel(indexName, keyPattern string, b *RedisBackend, ids ...uuid.UUID) error {
	idStrings := uuidsToStrings(ids)
	key := fmt.Sprintf(keyPattern, idStrings...)
	b.redis.Del(ctx, key)
	b.redis.SRem(ctx, indexName, key)
	return nil
}

func stringToModel[M interface{}](str string) (M, error) {
	var result M
	if err := json.Unmarshal([]byte(str), &result); err != nil {
		return result, err
	} else {
		return result, nil
	}
}

func modelToStr[M interface{}](context M) (string, error) {
	var result, err = json.Marshal(context)
	if err != nil {
		return "", err
	}
	return string(result), nil
}

func uuidsToStrings(uuids []uuid.UUID) []any {
	result := []any{}
	for _, uuid := range uuids {
		result = append(result, uuid.String())
	}
	return result
}
