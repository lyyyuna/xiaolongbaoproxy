package internal

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

type httpMessage struct {
	Host     string
	Port     string
	Method   string
	Path     string
	Size     int64
	Duration int64
	Time     int64
	Scheme   string
}

type MongoClient struct {
	client       *mongo.Client
	host         string
	port         int
	database     string
	collection   string
	interval     int
	httptHistory chan *httpMessage
}

func NewMongoClient(host string, port int, database string, collection string, interval int) *MongoClient {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	mongoUri := fmt.Sprintf("mongodb://%v:%v", host, port)
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoUri))
	if err != nil {
		panic("Cannot connect to mongodb, the error is " + err.Error())
	}

	return &MongoClient{
		client:       client,
		host:         host,
		port:         port,
		database:     database,
		collection:   collection,
		interval:     interval,
		httptHistory: make(chan *httpMessage),
	}
}

func (mc *MongoClient) PersistLoop() {
	var httpMessages []*httpMessage
	// two cases won't interfere with each other
	for {
		select {
		case mes := <-mc.httptHistory:
			httpMessages = append(httpMessages, mes)

		case <-time.After(1 * time.Second):
			if len(httpMessages) == 0 {
				continue
			}
			// as it executes 1 second duration
			// we can use batch insert to mongo
			mc.saveToMongo(httpMessages)
			// clear the slices
			httpMessages = httpMessages[:0]
		}
	}
}

func (mc *MongoClient) saveToMongo(messages []*httpMessage) {
	collection := mc.client.Database(mc.database).Collection(mc.collection)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var data []interface{}
	for _, t := range messages {
		data = append(data, t)
	}
	_, err := collection.InsertMany(ctx, data)
	if err != nil {
		zap.S().Errorf("Error while saving data to mongodb: %v", err)
	}
}
