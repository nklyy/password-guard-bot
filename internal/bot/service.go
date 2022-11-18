package bot

import (
	"context"
	"errors"
	"fmt"
	"password-manager-bot/pkg/crypto"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
)

type Service interface {
	CheckDuplicateFromWhatData(user UserState, chatId int64, from string) (bool, error)

	GetUserData(chatId int64) (map[string]string, error)
	GetUserDataNamesByChunks(chatId int64) ([][]tgbotapi.InlineKeyboardButton, error)

	CreateUser(chatId int64) error

	UpdateUser(chatId int64, user User) error
	UpdateUserEncryptedData(chatId int64, fromWhat, encryptedData string) error

	EncryptData(chatId int64, userState UserState) (*string, error)
	DecryptData(chatId int64, pin, fromWhat string) (*string, error)
}

type service struct {
	botApi     *tgbotapi.BotAPI
	cryptoSvc  crypto.CryptoService
	repository Repository
	logger     *zap.SugaredLogger
}

func NewService(botApi *tgbotapi.BotAPI, cryptoSvc crypto.CryptoService, repository Repository, logger *zap.SugaredLogger) (Service, error) {
	if botApi == nil {
		return nil, errors.New("invalid telegram bot api")
	}
	if cryptoSvc == nil {
		return nil, errors.New("invalid crypto service")
	}
	if repository == nil {
		return nil, errors.New("invalid repository")
	}
	if logger == nil {
		return nil, errors.New("invalid logger")
	}

	return &service{botApi: botApi, cryptoSvc: cryptoSvc, repository: repository, logger: logger}, nil
}

func (s *service) CheckDuplicateFromWhatData(user UserState, chatId int64, from string) (bool, error) {
	dbUser, err := s.repository.GetUser(context.Background(), bson.M{"telegram_id": chatId})
	if err != nil {
		return false, err
	}

	if dbUser.Data == nil {
		return false, nil
	}

	if _, ok := (*dbUser.Data)[from]; ok {
		return true, nil
	}

	return false, nil
}

func (s *service) GetUserData(chatId int64) (map[string]string, error) {
	user, err := s.repository.GetUser(context.Background(), bson.M{"telegram_id": chatId})
	if err != nil {
		return nil, err
	}

	if user == nil {
		return nil, nil
	}

	return (*user.Data), nil
}

func (s *service) GetUserDataNamesByChunks(chatId int64) ([][]tgbotapi.InlineKeyboardButton, error) {
	user, err := s.repository.GetUser(context.Background(), bson.M{"telegram_id": chatId})
	if err != nil {
		return nil, err
	}

	if user.Data == nil {
		return nil, nil
	}

	var chunks [][]tgbotapi.InlineKeyboardButton
	var chunk []tgbotapi.InlineKeyboardButton

	for k := range *user.Data {
		if len(chunk) == 3 {
			chunks = append(chunks, chunk)
			chunk = nil
		}
		chunk = append(chunk, tgbotapi.NewInlineKeyboardButtonData(k, k))
	}

	chunks = append(chunks, chunk)

	return chunks, nil
}

func (s *service) CreateUser(chatId int64) error {
	user, err := NewUser(&chatId)
	if err != nil {
		return err
	}

	err = s.repository.CreatUser(context.Background(), user)
	if err != nil {
		return err
	}

	return nil
}

func (s *service) UpdateUser(chatId int64, user User) error {
	err := s.repository.UpdateUser(context.Background(), &user)
	if err != nil {
		return err
	}

	return nil
}

func (s *service) UpdateUserEncryptedData(chatId int64, fromWhat, encryptedData string) error {
	user, err := s.repository.GetUser(context.Background(), bson.M{"telegram_id": chatId})
	if err != nil {
		return err
	}

	if user == nil {
		return errors.New("User not found")
	}

	user.AddData(fromWhat, encryptedData)

	err = s.repository.UpdateUser(context.Background(), user)
	if err != nil {
		return err
	}

	return nil
}

func (s *service) EncryptData(chatId int64, userState UserState) (*string, error) {
	rawData := fmt.Sprintf("%s:%s", strings.TrimSpace(userState.Login), strings.TrimSpace(userState.Password))
	encryptedData, err := s.cryptoSvc.Encrypt(s.cryptoSvc.GenerateNormalSizeCode(strings.TrimSpace(userState.Pin)), []byte(rawData))
	if err != nil {
		return nil, err
	}

	return &encryptedData, nil

}

func (s *service) DecryptData(chatId int64, pin, fromWhat string) (*string, error) {
	user, err := s.repository.GetUser(context.Background(), bson.M{"telegram_id": chatId})
	if err != nil {
		return nil, err
	}

	data := (*user.Data)[fromWhat]

	normalizePin := s.cryptoSvc.GenerateNormalSizeCode(strings.TrimSpace(pin))
	decrypted, err := s.cryptoSvc.Decrypt([]byte(normalizePin), data)
	if err != nil {
		return nil, err
	}

	return &decrypted, nil
}
