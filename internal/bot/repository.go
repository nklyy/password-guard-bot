package bot

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

//go:generate mockgen -source=repository.go -destination=mocks/repository_mock.go
type Repository interface {
	GetUser(ctx context.Context, filter bson.M) (*User, error)
	CreatUser(ctx context.Context, user *User) error
	CreateUniqueIndexes(ctx context.Context) error
	UpdateUser(ctx context.Context, user *User) error
	DeleteData(ctx context.Context, filter bson.M) error
}

type repository struct {
	db     *mongo.Client
	dbName string
	logger *zap.SugaredLogger
}

func NewRepository(db *mongo.Client, dbName string, logger *zap.SugaredLogger) (Repository, error) {
	if db == nil {
		return nil, errors.New("invalid database client")
	}
	if dbName == "" {
		return nil, errors.New("invalid database name")
	}
	if logger == nil {
		return nil, errors.New("invalid logger")
	}

	return &repository{db: db, dbName: dbName, logger: logger}, nil
}

func (r *repository) GetUser(ctx context.Context, filter bson.M) (*User, error) {
	var user User

	if err := r.db.Database(r.dbName).Collection("data").FindOne(ctx, filter).Decode(&user); err != nil {
		if err == mongo.ErrNoDocuments {
			r.logger.Errorf("unable to find user by name %v", err)
			return nil, err
		}

		r.logger.Errorf("unable to find user due to internal error: %v", err)
		return nil, err
	}

	return &user, nil
}

func (r *repository) CreatUser(ctx context.Context, user *User) error {
	_, err := r.db.Database(r.dbName).Collection("data").InsertOne(ctx, user)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			r.logger.Errorf("failed to insert user data to db due to duplicate error: %v", err)
			return err
		}

		r.logger.Errorf("failed to insert user data to db: %v", err)
		return err
	}

	return nil
}

func (r *repository) CreateUniqueIndexes(ctx context.Context) error {
	mod := mongo.IndexModel{
		Keys:    bson.M{"telegram_id": 1},
		Options: options.Index().SetUnique(true),
	}

	_, err := r.db.Database(r.dbName).Collection("data").Indexes().CreateOne(ctx, mod)
	if err != nil {
		return err
	}

	return nil
}

func (r *repository) UpdateUser(ctx context.Context, user *User) error {
	_, err := r.db.Database(r.dbName).Collection("data").UpdateOne(ctx, bson.M{"telegram_id": user.TelegramId},
		bson.D{primitive.E{Key: "$set", Value: user}})

	if err != nil {
		r.logger.Errorf("failed to update user %v", err)
		return err
	}

	return nil
}

func (r *repository) DeleteData(ctx context.Context, filter bson.M) error {
	_, err := r.db.Database(r.dbName).Collection("data").DeleteOne(ctx, filter)
	if err != nil {
		r.logger.Errorf("failed to delete data %v", err)
		return err
	}

	return nil
}
