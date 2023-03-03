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
	GetUserWithSliceAndDataSize(ctx context.Context, filter bson.M, page int) (*User, *int, error)
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
			r.logger.Errorf("failed to find user by name: %s", err)
			return nil, err
		}

		r.logger.Errorf("failed to find user due to internal error: s", err)
		return nil, err
	}

	return &user, nil
}

func (r *repository) GetUserWithSliceAndDataSize(ctx context.Context, filter bson.M, page int) (*User, *int, error) {
	var dbUser struct {
		ID         primitive.ObjectID  `bson:"_id"`
		TelegramId int64               `bson:"telegram_id"`
		Data       []map[string]string `bson:"data"`
		DataSize   int                 `bson:"data_size"`
	}

	limit := 9
	offset := (page - 1) * limit

	projection := bson.M{
		"data": bson.M{
			"$slice": bson.A{bson.D{{Key: "$objectToArray", Value: "$data"}}, offset, limit},
		},
		"data_size": bson.M{
			"$size": bson.A{bson.D{{Key: "$objectToArray", Value: "$data"}}},
		},
	}

	options := options.FindOne().SetProjection(projection)

	if err := r.db.Database(r.dbName).Collection("data").FindOne(ctx, filter, options).Decode(&dbUser); err != nil {
		if err == mongo.ErrNoDocuments {
			r.logger.Errorf("failed to find user by name: %s", err)
			return nil, nil, err
		}

		r.logger.Errorf("failed to find user due to internal error: s", err)
		return nil, nil, err
	}

	var user User
	user.ID = dbUser.ID
	user.TelegramId = dbUser.TelegramId

	data := make(map[string]string)
	for _, item := range dbUser.Data {
		data[item["k"]] = item["v"]
	}

	user.Data = &data

	return &user, &dbUser.DataSize, nil
}

func (r *repository) CreatUser(ctx context.Context, user *User) error {
	_, err := r.db.Database(r.dbName).Collection("data").InsertOne(ctx, user)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			r.logger.Errorf("failed to insert user data to db due to duplicate error: %s", err)
			return err
		}

		r.logger.Errorf("failed to insert user data to db: %s", err)
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
		r.logger.Errorf("failed to update user %s", err)
		return err
	}

	return nil
}

func (r *repository) DeleteData(ctx context.Context, filter bson.M) error {
	_, err := r.db.Database(r.dbName).Collection("data").DeleteOne(ctx, filter)
	if err != nil {
		r.logger.Errorf("failed to delete data %s", err)
		return err
	}

	return nil
}
