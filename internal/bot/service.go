package bot

import (
	"context"
	"errors"
	"fmt"
	"password-guard-bot/pkg/crypto"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

type Service interface {
	CheckDuplicateFromWhatData(user UserState, chatId int64, from string) (bool, error)
	CheckExistUser(chatId int64) (bool, error)

	GetUserData(chatId int64) (map[string]string, error)
	GetUserDataNamesByChunks(chatId int64, page int) ([][]tgbotapi.InlineKeyboardButton, error)

	CreateUser(chatId int64) error

	UpdateUser(chatId int64, user User) error
	UpdateUserEncryptedData(chatId int64, fromWhat, encryptedData string) error

	DeleteData(chatId int64, what string) error

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

func (s *service) CheckExistUser(chatId int64) (bool, error) {
	_, err := s.repository.GetUser(context.Background(), bson.M{"telegram_id": chatId})
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return false, nil
		}

		return false, err
	}

	return true, nil
}

func (s *service) GetUserData(chatId int64) (map[string]string, error) {
	user, err := s.repository.GetUser(context.Background(), bson.M{"telegram_id": chatId})
	if err != nil {
		return nil, err
	}

	return (*user.Data), nil
}

func (s *service) GetUserDataNamesByChunks(chatId int64, page int) ([][]tgbotapi.InlineKeyboardButton, error) {
	user, dataSize, err := s.repository.GetUserWithSliceAndDataSize(context.Background(), bson.M{"telegram_id": chatId}, page)
	if err != nil {
		return nil, err
	}

	if user.Data == nil || len(*user.Data) == 0 {
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

	var buttons []tgbotapi.InlineKeyboardButton
	if page > 1 {
		buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData("< Prev", "prev"))
	}
	if len(*user.Data) == 9 && len(*user.Data) != *dataSize {
		buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData("Next >", "next"))
	}

	chunks = append(chunks, chunk)

	if len(buttons) > 0 {
		chunks = append(chunks, buttons)
	}

	return chunks, nil
}

func (s *service) CreateUser(chatId int64) error {
	user, err := NewUser(&chatId)
	if err != nil {
		s.logger.Errorf("failed to create user instance: %s", err)
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

	user.AddData(fromWhat, encryptedData)

	err = s.repository.UpdateUser(context.Background(), user)
	if err != nil {
		return err
	}

	return nil
}

func (s *service) DeleteData(chatId int64, what string) error {
	user, err := s.repository.GetUser(context.Background(), bson.M{"telegram_id": chatId})
	if err != nil {
		return err
	}

	user.DeleteData(what)

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
		s.logger.Errorf("failed to encrypt data: %s", err)
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
		s.logger.Errorf("failed to decrypt data: %s", err)
		return nil, err
	}

	replacedDec := strings.Replace(decrypted, "\n", " ", -1)

	return &replacedDec, nil
}
