package bot

import (
	"errors"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID         primitive.ObjectID `bson:"_id"`
	TelegramId int64              `bson:"telegram_id"`
	Data       *map[string]string `bson:"data"`
}

func NewUser(telegramId *int64) (*User, error) {
	if telegramId == nil {
		return nil, errors.New("invalid telegramId")
	}

	return &User{
		ID:         primitive.NewObjectID(),
		TelegramId: *telegramId,
		Data:       nil,
	}, nil
}

func (u *User) AddData(fromWhatData string, encryptedData string) {
	if u.Data == nil {
		u.Data = &map[string]string{fromWhatData: encryptedData}
	} else {
		(*u.Data)[fromWhatData] = encryptedData
	}
}
