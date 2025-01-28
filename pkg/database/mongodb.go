package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// MongoClient MongoDB客户端
type MongoClient struct {
	client   *mongo.Client
	database *mongo.Database
}

// NewMongoClient 创建MongoDB客户端实例
func NewMongoClient(config *MongoDBConfig) (*MongoClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 创建客户端选项
	clientOptions := options.Client().
		ApplyURI(config.URI).
		SetMaxPoolSize(config.MaxPoolSize).
		SetMinPoolSize(config.MinPoolSize)

	// 连接到MongoDB
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Printf("Failed to connect to MongoDB: %v\n", err)
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// 测试连接
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		log.Printf("Failed to ping MongoDB: %v\n", err)
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	// 获取数据库
	database := client.Database(config.Database)

	log.Printf("Successfully connected to MongoDB database: %s", config.Database)

	return &MongoClient{
		client:   client,
		database: database,
	}, nil
}

// Close 关闭MongoDB连接
func (c *MongoClient) Close(ctx context.Context) error {
	return c.client.Disconnect(ctx)
}

// Database 获取数据库实例
func (c *MongoClient) Database() *mongo.Database {
	return c.database
}

// Collection 获取集合
func (c *MongoClient) Collection(name string) *mongo.Collection {
	return c.database.Collection(name)
}

// MongoDBConfig MongoDB配置
type MongoDBConfig struct {
	URI         string
	Database    string
	MaxPoolSize uint64
	MinPoolSize uint64
}
