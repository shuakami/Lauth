package repository

import (
	"context"
	"time"

	"lauth/internal/model"
	"lauth/pkg/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	fileCollection = "files"
)

// FileRepository 文件仓储接口
type FileRepository interface {
	// Create 创建文件记录
	Create(ctx context.Context, file *model.File) error
	// Update 更新文件记录
	Update(ctx context.Context, id primitive.ObjectID, file *model.File) error
	// Delete 删除文件记录
	Delete(ctx context.Context, id primitive.ObjectID) error
	// GetByID 通过ID获取文件记录
	GetByID(ctx context.Context, id primitive.ObjectID) (*model.File, error)
	// List 获取文件列表
	List(ctx context.Context, userID, appID string, offset, limit int64) ([]*model.File, error)
	// Count 获取文件总数
	Count(ctx context.Context, userID, appID string) (int64, error)
	// ListByTags 通过标签获取文件列表
	ListByTags(ctx context.Context, userID, appID string, tags []string, offset, limit int64) ([]*model.File, error)
	// UpdateCustomData 更新自定义数据
	UpdateCustomData(ctx context.Context, id primitive.ObjectID, data map[string]interface{}) error
}

// fileRepository 文件仓储实现
type fileRepository struct {
	mongo *database.MongoClient
}

// NewFileRepository 创建文件仓储实例
func NewFileRepository(mongo *database.MongoClient) FileRepository {
	return &fileRepository{mongo: mongo}
}

// Create 创建文件记录
func (r *fileRepository) Create(ctx context.Context, file *model.File) error {
	now := time.Now()
	file.CreatedAt = now
	file.UpdatedAt = now

	collection := r.mongo.Collection(fileCollection)
	_, err := collection.InsertOne(ctx, file)
	return err
}

// Update 更新文件记录
func (r *fileRepository) Update(ctx context.Context, id primitive.ObjectID, file *model.File) error {
	file.UpdatedAt = time.Now()

	collection := r.mongo.Collection(fileCollection)
	_, err := collection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": file},
	)
	return err
}

// Delete 删除文件记录
func (r *fileRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	collection := r.mongo.Collection(fileCollection)
	_, err := collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

// GetByID 通过ID获取文件记录
func (r *fileRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*model.File, error) {
	collection := r.mongo.Collection(fileCollection)
	var file model.File
	err := collection.FindOne(ctx, bson.M{"_id": id}).Decode(&file)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &file, err
}

// List 获取文件列表
func (r *fileRepository) List(ctx context.Context, userID, appID string, offset, limit int64) ([]*model.File, error) {
	collection := r.mongo.Collection(fileCollection)

	filter := bson.M{
		"user_id": userID,
		"app_id":  appID,
	}

	opts := options.Find().
		SetSkip(offset).
		SetLimit(limit).
		SetSort(bson.M{"created_at": -1})

	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var files []*model.File
	if err := cursor.All(ctx, &files); err != nil {
		return nil, err
	}
	return files, nil
}

// Count 获取文件总数
func (r *fileRepository) Count(ctx context.Context, userID, appID string) (int64, error) {
	collection := r.mongo.Collection(fileCollection)
	filter := bson.M{
		"user_id": userID,
		"app_id":  appID,
	}
	return collection.CountDocuments(ctx, filter)
}

// ListByTags 通过标签获取文件列表
func (r *fileRepository) ListByTags(ctx context.Context, userID, appID string, tags []string, offset, limit int64) ([]*model.File, error) {
	collection := r.mongo.Collection(fileCollection)

	filter := bson.M{
		"user_id": userID,
		"app_id":  appID,
		"tags":    bson.M{"$in": tags},
	}

	opts := options.Find().
		SetSkip(offset).
		SetLimit(limit).
		SetSort(bson.M{"created_at": -1})

	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var files []*model.File
	if err := cursor.All(ctx, &files); err != nil {
		return nil, err
	}
	return files, nil
}

// UpdateCustomData 更新自定义数据
func (r *fileRepository) UpdateCustomData(ctx context.Context, id primitive.ObjectID, data map[string]interface{}) error {
	collection := r.mongo.Collection(fileCollection)
	_, err := collection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{
			"$set": bson.M{
				"custom_data": data,
				"updated_at":  time.Now(),
			},
		},
	)
	return err
}
