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
	profileCollection = "profiles"
)

// ProfileRepository Profile仓储接口
type ProfileRepository interface {
	// Create 创建用户档案
	Create(ctx context.Context, profile *model.Profile) error
	// Update 更新用户档案
	Update(ctx context.Context, id primitive.ObjectID, profile *model.Profile) error
	// Delete 删除用户档案
	Delete(ctx context.Context, id primitive.ObjectID) error
	// GetByID 通过ID获取用户档案
	GetByID(ctx context.Context, id primitive.ObjectID) (*model.Profile, error)
	// GetByUserID 通过用户ID获取用户档案
	GetByUserID(ctx context.Context, userID string) (*model.Profile, error)
	// List 获取用户档案列表
	List(ctx context.Context, appID string, offset, limit int64) ([]*model.Profile, error)
	// Count 获取用户档案总数
	Count(ctx context.Context, appID string) (int64, error)
	// UpdateCustomData 更新自定义数据
	UpdateCustomData(ctx context.Context, id primitive.ObjectID, data map[string]interface{}) error
}

// profileRepository Profile仓储实现
type profileRepository struct {
	mongo *database.MongoClient
}

// NewProfileRepository 创建Profile仓储实例
func NewProfileRepository(mongo *database.MongoClient) ProfileRepository {
	return &profileRepository{mongo: mongo}
}

// Create 创建用户档案
func (r *profileRepository) Create(ctx context.Context, profile *model.Profile) error {
	now := time.Now()
	profile.CreatedAt = now
	profile.UpdatedAt = now

	collection := r.mongo.Collection(profileCollection)
	_, err := collection.InsertOne(ctx, profile)
	return err
}

// Update 更新用户档案
func (r *profileRepository) Update(ctx context.Context, id primitive.ObjectID, profile *model.Profile) error {
	profile.UpdatedAt = time.Now()

	collection := r.mongo.Collection(profileCollection)
	_, err := collection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": profile},
	)
	return err
}

// Delete 删除用户档案
func (r *profileRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	collection := r.mongo.Collection(profileCollection)
	_, err := collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

// GetByID 通过ID获取用户档案
func (r *profileRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*model.Profile, error) {
	collection := r.mongo.Collection(profileCollection)
	var profile model.Profile
	err := collection.FindOne(ctx, bson.M{"_id": id}).Decode(&profile)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &profile, err
}

// GetByUserID 通过用户ID获取用户档案
func (r *profileRepository) GetByUserID(ctx context.Context, userID string) (*model.Profile, error) {
	collection := r.mongo.Collection(profileCollection)
	var profile model.Profile
	err := collection.FindOne(ctx, bson.M{"user_id": userID}).Decode(&profile)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &profile, err
}

// List 获取用户档案列表
func (r *profileRepository) List(ctx context.Context, appID string, offset, limit int64) ([]*model.Profile, error) {
	collection := r.mongo.Collection(profileCollection)

	opts := options.Find().
		SetSkip(offset).
		SetLimit(limit).
		SetSort(bson.M{"created_at": -1})

	cursor, err := collection.Find(ctx, bson.M{"app_id": appID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var profiles []*model.Profile
	if err := cursor.All(ctx, &profiles); err != nil {
		return nil, err
	}
	return profiles, nil
}

// Count 获取用户档案总数
func (r *profileRepository) Count(ctx context.Context, appID string) (int64, error) {
	collection := r.mongo.Collection(profileCollection)
	return collection.CountDocuments(ctx, bson.M{"app_id": appID})
}

// UpdateCustomData 更新自定义数据
func (r *profileRepository) UpdateCustomData(ctx context.Context, id primitive.ObjectID, data map[string]interface{}) error {
	collection := r.mongo.Collection(profileCollection)
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
