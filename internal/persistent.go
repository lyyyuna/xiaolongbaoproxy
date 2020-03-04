package internal

import (
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

type httpMessage struct {
	Host     string
	Port     int
	Method   string
	Path     string
	Size     int
	Duration int
	Time     int
}

type mongoClient struct {
	client       *mongo.Client
	host         string
	port         int
	interval     int
	httptHistory chan httpMessage
}

func NewMongoClient(host string, port, interval int) *mongoClient {
	mongoUri := fmt.Sprintf("mongodb://%v:%v", host, port)
	client, err := mongo.NewClient(options.Client().ApplyURI(mongoUri))
	if err != nil {
		panic("Cannot connect to mongodb, the error is " + err.Error())
	}

	return &mongoClient{
		client:       client,
		host:         host,
		port:         port,
		interval:     interval,
		httptHistory: make(chan httpMessage),
	}
}

func (mc *mongoClient) persistLoop() {
	var httpMessages []httpMessage
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

func (mc *mongoClient) saveToMongo(messages []httpMessage) {

}
