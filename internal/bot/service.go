package bot

import (
	"errors"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

type Service interface {
}

type service struct {
	botApi     *tgbotapi.BotAPI
	repository Repository
	logger     *zap.SugaredLogger
}

func NewService(botApi *tgbotapi.BotAPI, repository Repository, logger *zap.SugaredLogger) (Client, error) {
	if botApi == nil {
		return nil, errors.New("invalid telegram bot api")
	}
	if repository == nil {
		return nil, errors.New("invalid repository")
	}
	if logger == nil {
		return nil, errors.New("invalid logger")
	}

	return &client{botApi: botApi, repository: repository, logger: logger}, nil
}
