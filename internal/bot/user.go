package bot

import (
	"errors"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID         primitive.ObjectID `bson:"_id"`
	TelegramId int64              `bson:"telegram_id"`
	Data       map[string]string  `bson:"data"`
}

func NewUser(telegramId *int64, fromWhatData string, encryptedData string) (*User, error) {
	if telegramId == nil {
		return nil, errors.New("invalid telegramId")
	}

	if fromWhatData == "" {
		return nil, errors.New("invalid fromWhatData")
	}

	if encryptedData == "" {
		return nil, errors.New("invalid encryptedData")
	}

	return &User{
		ID:         primitive.NewObjectID(),
		TelegramId: *telegramId,
		Data:       map[string]string{fromWhatData: encryptedData},
	}, nil
}

func (u *User) AddData(fromWhatData string, encryptedData string) {
	u.Data[fromWhatData] = encryptedData
}
