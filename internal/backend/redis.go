package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/bespinian/ackbar/internal/model"
	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
)

const (
	contextsKeyIndexName   = "contexts:index"
	contextKeyPattern      = "contexts:%s"
	partitionsKeyIndexName = "partitions:index"
	partitionKeyPattern    = "contexts:%s:partitions:%s"
	workersKeyIndexName    = "workers:index"
	workersKeyPattern      = "contexts:%s:workers:%s"
	workersKeyRegex        = `contexts:([a-f|A-F|\d|-]+):workers:([a-f|A-F|\d|-]+)`
)

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

	b.redis.WithTimeout(0)

	_, err := b.redis.Do(context.Background(), "CONFIG", "SET", "notify-keyspace-events", "KEA").Result()
	if err != nil {
		log.Fatalf("Could not initialize pub/sub in Redis backend: %e", err)
	}

	// tracking worker liveness requires listening to key expirations
	pubsub := b.redis.PSubscribe(context.Background(), "__keyevent@0__:expired")

	// workers are expired by Redis, so expirations might have been missed during downtime. Catch up now.
	b.handleMissedWorkerExpirations()

	go func(*redis.PubSub) {
		expirations := pubsub.Channel()
		for {
			message := <-expirations
			b.handleWorkerExpiration(message.String())
		}
	}(pubsub)
}

func (b *RedisBackend) handleWorkerExpiration(workerKey string) {
	r, err := regexp.Compile(workersKeyRegex)
	if err != nil {
		log.Printf("Could not compile worker key regex: %e", err)
	}
	contextId := r.FindStringSubmatch(workerKey)[1]
	contextIdUuid, err := uuid.Parse(contextId)
	if err != nil {
		log.Printf("Error parsing context id %s as a UUID: %e", contextId, err)
	}
	workerId := r.FindStringSubmatch(workerKey)[2]
	workerIdUuid, err := uuid.Parse(workerId)
	if err != nil {
		log.Printf("Error parsing worker id %s as a UUID: %e", contextId, err)
	}
	log.Printf("Expiration of worker %s in context %s", workerId, contextId)
	err = b.unassignWorkerFromAllPartitions(contextIdUuid, workerIdUuid)
	if err != nil {
		log.Printf("Error unsassigning worker %s from all partitions in context %s", workerId, contextId)
	}
}

func (b *RedisBackend) GetContexts() ([]model.Context, error) {
	contexts, err := getModelList[model.Context](contextsKeyIndexName, b, "")
	if err != nil {
		return []model.Context{}, err
	}
	updatedContexts := []model.Context{}
	for _, context := range contexts {
		err := b.addPartitionToWorkerRatio(&context, context.MaxPartitionsPerWorker)
		if err != nil {
			return []model.Context{}, err
		}
		updatedContexts = append(updatedContexts, context)
	}
	return updatedContexts, nil
}

func (b *RedisBackend) addPartitionToWorkerRatio(context *model.Context, maxPartitionsPerWorker int) error {
	ratio, err := b.partitionToWorkerRatio(context.ID, maxPartitionsPerWorker)
	if err != nil {
		return err
	}
	context.PartitionToWorkerRatio = ratio
	return nil
}

func (b *RedisBackend) GetContext(id uuid.UUID) (model.Context, bool, error) {
	context, found, err := getModel[model.Context](contextKeyPattern, b, id)
	if err == nil && found {
		err = b.addPartitionToWorkerRatio(&context, context.MaxPartitionsPerWorker)
	}
	return context, found, err
}

func (b *RedisBackend) CreateContext(name string, livenessIntervalSeconds, maxPartitionsPerWorker int) (model.Context, error) {
	context := model.Context{ID: uuid.New(), Name: name, LivenessIntervalSeconds: livenessIntervalSeconds, MaxPartitionsPerWorker: maxPartitionsPerWorker}
	return createOrUpdateModel(context, contextsKeyIndexName, contextKeyPattern, 0, b, context.ID)
}

func (b *RedisBackend) DeleteContext(id uuid.UUID) error {
	partitions, err := b.GetPartitions(id)
	if err != nil {
		return err
	}
	for _, partition := range partitions {
		err = b.DeletePartition(id, partition.ID)
		if err != nil {
			log.Printf("Error deleting partition %s in context %s: %e", partition.ID, id, err)
		}
	}
	return deleteModel(contextsKeyIndexName, contextKeyPattern, b, id)
}

func (b *RedisBackend) GetPartitions(contextId uuid.UUID) ([]model.Partition, error) {
	partitionsKeyPrefix := fmt.Sprintf(contextKeyPattern, contextId)
	partitions, err := getModelList[model.Partition](partitionsKeyIndexName, b, partitionsKeyPrefix)
	if err != nil {
		return partitions, err
	}
	return partitions, nil
}

func (b *RedisBackend) GetPartition(contextId, partitionId uuid.UUID) (model.Partition, bool, error) {
	context, found, err := getModel[model.Partition](partitionKeyPattern, b, contextId, partitionId)
	return context, found, err
}

func (b *RedisBackend) AddPartition(contextId uuid.UUID, configuration string) (model.Partition, error) {
	_, _, err := b.GetContext(contextId)
	if err != nil {
		log.Printf("error getting context with id %s", contextId)
		return model.Partition{}, err
	}
	partition := model.Partition{ID: uuid.New(), ContextID: contextId, Configuration: configuration}
	partition, err = b.assignWorkerIfPossible(contextId, partition)
	if err != nil {
		log.Printf("error assigning a worker with free capacity for partition %s context with id %s", partition.ID, contextId)
		return model.Partition{}, err
	}
	return createOrUpdateModel(partition, partitionsKeyIndexName, partitionKeyPattern, 0, b, contextId, partition.ID)
}

func (b *RedisBackend) assignWorkerIfPossible(contextId uuid.UUID, partition model.Partition) (model.Partition, error) {
	unassignedWorkers, err := b.getWorkersWithFreeCapacity(contextId)
	if err != nil {
		log.Printf("error getting workers with free capacity for context with id %s", contextId)
		return model.Partition{}, err
	}
	if len(unassignedWorkers) > 0 {
		// there are workers with free capacity. Assign the first one that was found.
		worker := unassignedWorkers[0]
		partition.AssignedWorker = worker.ID
	}
	return partition, nil
}

func (b *RedisBackend) DeletePartition(contextId, partitionId uuid.UUID) error {
	_, _, err := b.GetContext(contextId)
	if err != nil {
		log.Printf("error getting context with id %s", contextId)
		return err
	}
	return deleteModel(partitionsKeyIndexName, partitionKeyPattern, b, contextId, partitionId)
}

func (b *RedisBackend) RegisterWorker(contextId uuid.UUID) (model.Worker, error) {
	context, _, err := b.GetContext(contextId)
	if err != nil {
		log.Printf("error getting context with id %s", contextId)
		return model.Worker{}, err
	}

	now := time.Now()
	worker := model.Worker{ID: uuid.New(), ContextID: contextId, RegisteredAt: now, RefreshedAt: now}
	partitionConfigurations := []string{}
	for i := 0; i < context.MaxPartitionsPerWorker; i++ {
		partitions, err := b.getUnassignedPartitions(contextId)
		if err != nil {
			log.Printf("error getting unassigned partitions for context with id %s", contextId)
			return model.Worker{}, err
		}
		if len(partitions) > 0 {
			// there are unassigned partitions. Assign the new worker to the first one that was found
			partitions[0].AssignedWorker = worker.ID
			partitionConfigurations = append(partitionConfigurations, partitions[0].Configuration)
			_, err := createOrUpdateModel(partitions[0], partitionsKeyIndexName, partitionKeyPattern, 0, b, contextId, partitions[0].ID)
			if err != nil {
				return model.Worker{}, err
			}
		}
	}
	createdWorker, err := createOrUpdateModel(worker, workersKeyIndexName, workersKeyPattern, context.LivenessIntervalSeconds, b, contextId, worker.ID)
	createdWorker.PartitionConfigurations = partitionConfigurations
	if err != nil {
		return model.Worker{}, err
	}
	return createdWorker, nil
}

func (b *RedisBackend) RefreshWorker(contextId, workerId uuid.UUID) (model.Worker, bool, error) {
	context, _, err := b.GetContext(contextId)
	if err != nil {
		return model.Worker{}, false, err
	}
	worker, found, err := getModel[model.Worker](workersKeyPattern, b, contextId, workerId)
	if err != nil {
		return model.Worker{}, false, err
	}
	var refreshedWorker model.Worker
	refreshed := false
	if found {
		worker.RefreshedAt = time.Now()
		refreshedWorker, err = createOrUpdateModel(worker, workersKeyIndexName, workersKeyPattern, context.LivenessIntervalSeconds, b, contextId, worker.ID)
		refreshed = true
		if err != nil {
			return model.Worker{}, false, err
		}
	} else {
		refreshedWorker, err = b.RegisterWorker(contextId)
		if err != nil {
			return model.Worker{}, false, err
		}
	}
	updatedWorkers, err := b.addAssignedPartitionConfigurations(contextId, []model.Worker{refreshedWorker})
	if err != nil {
		log.Printf("Failed to get partition configurations for worker %s in context %s", workerId, contextId)
		return model.Worker{}, false, err
	}
	return updatedWorkers[0], refreshed, nil
}

func (b *RedisBackend) GetWorkers(contextId uuid.UUID) ([]model.Worker, error) {
	workersKeyPrefix := fmt.Sprintf(contextKeyPattern, contextId)
	workers, err := getModelList[model.Worker](workersKeyIndexName, b, workersKeyPrefix)
	if err != nil {
		return workers, err
	}
	workers, err = b.addAssignedPartitionConfigurations(contextId, workers)
	if err != nil {
		return workers, err
	}
	return workers, nil
}

func (b *RedisBackend) GetWorker(contextId, workerId uuid.UUID) (model.Worker, bool, error) {
	worker, found, err := getModel[model.Worker](partitionKeyPattern, b, contextId, workerId)
	if err != nil {
		log.Printf("Error when looking up worker %s in context %s: %e", workerId, contextId, err)
	}
	updatedWorkers, err := b.addAssignedPartitionConfigurations(contextId, []model.Worker{worker})
	return updatedWorkers[0], found, err
}

func (b *RedisBackend) DeleteWorker(contextId, workerId uuid.UUID) error {
	err := deleteModel(workersKeyIndexName, workersKeyPattern, b, contextId, workerId)
	if err != nil {
		log.Printf("error deleting worker %s in context %s", workerId, contextId)
		return err
	}
	return b.unassignWorkerFromAllPartitions(contextId, workerId)
}

func (b *RedisBackend) unassignWorkerFromAllPartitions(contextId, workerId uuid.UUID) error {
	assignedPartitions, err := b.getAssignedPartitions(contextId, workerId)
	if err != nil {
		log.Printf("error getting partitions assigned to worker %s in context %s", workerId, contextId)
		return err
	}
	// unassign the worker from all partitions
	for _, partition := range assignedPartitions {
		partition.AssignedWorker = uuid.Nil
		partition, err = b.assignWorkerIfPossible(contextId, partition)
		if err != nil {
			log.Printf("error unsassigning worker %s from partition %s in context %s", workerId, partition.ID, contextId)
			return err
		}
		_, err := createOrUpdateModel(partition, partitionsKeyIndexName, partitionKeyPattern, 0, b, contextId, partition.ID)
		if err != nil {
			log.Printf("error saving partition %s in context %s", partition.ID, contextId)
			return err
		}
	}
	return nil
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
				log.Printf("removing stale key %s from index %s", key, indexName)
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

func (b *RedisBackend) handleMissedWorkerExpirations() []string {
	result := []string{}
	keys, err := b.redis.SMembers(ctx, workersKeyIndexName).Result()
	if err != nil {
		log.Printf("Could not load keys from index %s", workersKeyIndexName)
		keys = []string{}
	}
	for _, key := range keys {
		_, err := b.redis.Get(ctx, key).Result()
		if err != nil {
			log.Printf("Missed worker expiration for %s, expiring worker now", key)
			b.handleWorkerExpiration(key)
			b.redis.SRem(ctx, workersKeyIndexName, key)
		}
	}

	return result
}

func (b *RedisBackend) getAssignedWorker(contextId, partitionId uuid.UUID) (uuid.UUID, error) {
	partition, found, err := b.GetPartition(contextId, partitionId)
	if !found {
		return uuid.Nil, fmt.Errorf("Partition with id %s does not exists", partitionId)
	}
	if err != nil {
		return uuid.Nil, err
	}
	return partition.AssignedWorker, nil
}

func (b *RedisBackend) getUnassignedPartitions(contextId uuid.UUID) ([]model.Partition, error) {
	result := []model.Partition{}
	partitionsKeyPrefix := fmt.Sprintf(contextKeyPattern, contextId)
	partitions, err := getModelList[model.Partition](partitionsKeyIndexName, b, partitionsKeyPrefix)
	if err != nil {
		return result, err
	}
	for _, partition := range partitions {
		worker, err := b.getAssignedWorker(contextId, partition.ID)
		if err != nil {
			return result, err
		}
		if worker == uuid.Nil {
			result = append(result, partition)
		}
	}
	return result, nil
}

func (b *RedisBackend) getWorkersWithFreeCapacity(contextId uuid.UUID) ([]model.Worker, error) {
	result := []model.Worker{}
	context, found, err := b.GetContext(contextId)
	if !found {
		return nil, fmt.Errorf("Context with id %s does not exists", contextId)
	}
	if err != nil {
		return nil, err
	}
	workersKeyPrefix := fmt.Sprintf(contextKeyPattern, contextId)
	workers, err := getModelList[model.Worker](workersKeyIndexName, b, workersKeyPrefix)
	if err != nil {
		return nil, err
	}
	for _, worker := range workers {
		assignedPartitions, err := b.getAssignedPartitions(contextId, worker.ID)
		if err != nil {
			return nil, err
		}
		if len(assignedPartitions) < context.MaxPartitionsPerWorker {
			result = append(result, worker)
		}
	}
	return result, nil
}

func (b *RedisBackend) getAssignedPartitions(contextId, workerId uuid.UUID) ([]model.Partition, error) {
	result := []model.Partition{}
	partitionsKeyPrefix := fmt.Sprintf(contextKeyPattern, contextId)
	partitions, err := getModelList[model.Partition](partitionsKeyIndexName, b, partitionsKeyPrefix)
	if err != nil {
		return nil, err
	}
	for _, partition := range partitions {
		if partition.AssignedWorker == workerId {
			result = append(result, partition)
		}
	}
	return result, nil
}

func (b *RedisBackend) getAssignedPartitionConfigurations(contextId, workerId uuid.UUID) ([]string, error) {
	configs := []string{}
	assignedPartitions, err := b.getAssignedPartitions(contextId, workerId)
	for _, partition := range assignedPartitions {
		configs = append(configs, partition.Configuration)
	}
	return configs, err
}

func (b *RedisBackend) addAssignedPartitionConfigurations(contextId uuid.UUID, workers []model.Worker) ([]model.Worker, error) {
	updatedWorkers := []model.Worker{}
	for _, worker := range workers {
		assignedPartitionConfigs, err := b.getAssignedPartitionConfigurations(contextId, worker.ID)
		if err != nil {
			log.Printf("Failed getting assigned partition configs for worker %s in context %s: %e", worker.ID, contextId, err)
			return updatedWorkers, err
		}
		worker.PartitionConfigurations = assignedPartitionConfigs
		updatedWorkers = append(updatedWorkers, worker)
	}
	return updatedWorkers, nil
}

func (b *RedisBackend) countWorkers(contextId uuid.UUID) (int, error) {
	workersKeyPrefix := fmt.Sprintf(contextKeyPattern, contextId)
	workers, err := getModelList[model.Worker](workersKeyIndexName, b, workersKeyPrefix)
	if err != nil {
		return 0, err
	}
	return len(workers), nil
}

func (b *RedisBackend) countPartitions(contextId uuid.UUID) (int, error) {
	partitions, err := b.GetPartitions(contextId)
	if err != nil {
		return 0, err
	}
	return len(partitions), nil
}

func (b *RedisBackend) partitionToWorkerRatio(contextId uuid.UUID, maxPartitionsPerWorker int) (float64, error) {
	result := 0.0
	workerCount, err := b.countWorkers(contextId)
	if err != nil {
		return result, err
	}
	partitionCount, err := b.countPartitions(contextId)
	if err != nil {
		return result, err
	}
	if workerCount == 0 {
		result = float64(partitionCount)
	} else {
		result = float64(partitionCount) / float64(workerCount*maxPartitionsPerWorker)
	}
	return result, nil
}

func getModel[M model.Model](keyPattern string, b *RedisBackend, ids ...uuid.UUID) (M, bool, error) {
	var result M
	idStrings := uuidsToStrings(ids)
	stringRep, err := b.redis.Get(ctx, fmt.Sprintf(keyPattern, idStrings...)).Result()
	if err != nil {
		log.Printf("error getting model with key %s", fmt.Sprintf(keyPattern, idStrings...))
		return result, false, err
	}
	result, err = stringToModel[M](stringRep)
	if err != nil {
		return result, false, err
	}
	return result, true, nil
}

func createOrUpdateModel[M model.Model](model M, indexName, keyPattern string, ttlSeconds int, b *RedisBackend, ids ...uuid.UUID) (M, error) {
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
	result, err := json.Marshal(context)
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
