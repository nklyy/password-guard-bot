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
	CheckDuplicateFromWhat(user UserState, chatId int64, from string) bool
	CreateOrUpdateUserData(chatId int64, user UserState) error
}

type service struct {
	botApi     *tgbotapi.BotAPI
	cryptoSvc  crypto.CryptoService
	messageSvc MessageService
	repository Repository
	logger     *zap.SugaredLogger
}

func NewService(botApi *tgbotapi.BotAPI, cryptoSvc crypto.CryptoService, messageSvc MessageService, repository Repository, logger *zap.SugaredLogger) (Service, error) {
	if botApi == nil {
		return nil, errors.New("invalid telegram bot api")
	}
	if cryptoSvc == nil {
		return nil, errors.New("invalid crypto service")
	}
	if messageSvc == nil {
		return nil, errors.New("invalid message service")
	}
	if repository == nil {
		return nil, errors.New("invalid repository")
	}
	if logger == nil {
		return nil, errors.New("invalid logger")
	}

	return &service{botApi: botApi, cryptoSvc: cryptoSvc, messageSvc: messageSvc, repository: repository, logger: logger}, nil
}

func (s *service) CheckDuplicateFromWhat(user UserState, chatId int64, from string) bool {
	dbUser, err := s.repository.GetUser(context.Background(), bson.M{"telegram_id": chatId})
	if err != nil {
		s.messageSvc.SendWrongMessage(chatId)
	}

	if _, ok := dbUser.Data[from]; ok {
		return true
	}

	return false
}

func (s *service) CreateOrUpdateUserData(chatId int64, user UserState) error {
	rawData := fmt.Sprintf("%s:%s", strings.TrimSpace(user.Login), strings.TrimSpace(user.Password))
	encryptedData, err := s.cryptoSvc.Encrypt(s.cryptoSvc.GenerateNormalSizeCode(strings.TrimSpace(user.Pin)), []byte(rawData))
	if err != nil {
		s.messageSvc.SendFailedEncryptMessage(chatId)
	}

	dbUser, err := s.repository.GetUser(context.Background(), bson.M{"telegram_id": chatId})
	if err != nil {
		s.messageSvc.SendWrongMessage(chatId)
	} else if dbUser == nil {
		dbUser, err := NewUser(&chatId, user.From, encryptedData)
		if err != nil {
			s.messageSvc.SendFailedCreateMessage(chatId)
		}

		s.repository.CreatUser(context.Background(), dbUser)
	}

	dbUser.AddData(user.From, encryptedData)
	err = s.repository.UpdateUser(context.Background(), dbUser)
	if err != nil {
		s.messageSvc.SendWrongMessage(chatId)
	}

	return nil
}
