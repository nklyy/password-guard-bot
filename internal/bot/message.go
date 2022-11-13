package bot

import (
	"errors"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
)

type MessageService interface {
	SendMessage(message tgbotapi.MessageConfig)
	DeleteMessage(chatId int64, messageId int)

	SendWrongMessage(chatId int64)
	SendSuccessMessage(chatId int64)
	SendFailedEncryptMessage(chatId int64)
	SendFailedCreateMessage(chatId int64)
	SendAlreadyHaveNameWithKeyboard(chatId int64)

	AskPin(chatId int64)
	AskLogin(chatId int64)
	AskPassword(chatId int64)
	AskNewNameFromData(chatId int64)
}

type messageService struct {
	botApi *tgbotapi.BotAPI
	logger *zap.SugaredLogger
}

func NewMessageService(botApi *tgbotapi.BotAPI, logger *zap.SugaredLogger) (MessageService, error) {
	if botApi == nil {
		return nil, errors.New("invalid telegram bot api")
	}
	if logger == nil {
		return nil, errors.New("invalid logger")
	}

	return &messageService{botApi: botApi, logger: logger}, nil
}

var keyboard = tgbotapi.NewInlineKeyboardMarkup(
	tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("Yes", "yes"),
		tgbotapi.NewInlineKeyboardButtonData("No", "no"),
	),
)

func (s *messageService) SendMessage(message tgbotapi.MessageConfig) {
	if _, err := s.botApi.Send(message); err != nil {
		s.logger.Panic(err)
	}
}

func (s *messageService) DeleteMessage(chatId int64, messageId int) {
	if _, err := s.botApi.Request(tgbotapi.NewDeleteMessage(chatId, messageId)); err != nil {
		s.logger.Panic(err)
	}
}

func (s *messageService) SendWrongMessage(chatId int64) {
	if _, err := s.botApi.Send(tgbotapi.NewMessage(chatId, "Something wrong. Please try later.")); err != nil {
		s.logger.Panic(err)
	}
}

func (s *messageService) SendSuccessMessage(chatId int64) {
	if _, err := s.botApi.Send(tgbotapi.NewMessage(chatId, "Success. Your password has been encrypted and added.")); err != nil {
		s.logger.Panic(err)
	}
}

func (s *messageService) SendFailedEncryptMessage(chatId int64) {
	if _, err := s.botApi.Send(tgbotapi.NewMessage(chatId, "Failed encrypt data. Please try later.")); err != nil {
		s.logger.Panic(err)
	}
}

func (s *messageService) SendFailedCreateMessage(chatId int64) {
	if _, err := s.botApi.Send(tgbotapi.NewMessage(chatId, "Failed create data. Please try later.")); err != nil {
		s.logger.Panic(err)
	}
}

func (s *messageService) SendAlreadyHaveNameWithKeyboard(chatId int64) {
	msg := tgbotapi.NewMessage(chatId, "You already have this name. Do you want replace?")

	msg.ReplyMarkup = keyboard
	if _, err := s.botApi.Send(msg); err != nil {
		s.logger.Panic(err)
	}
}

func (s *messageService) AskPin(chatId int64) {
	if _, err := s.botApi.Send(tgbotapi.NewMessage(chatId, "Goog. Now enter pin code. You can use one pin code for all passwords or one pin code for one password.")); err != nil {
		s.logger.Panic(err)
	}
}

func (s *messageService) AskLogin(chatId int64) {
	if _, err := s.botApi.Send(tgbotapi.NewMessage(chatId, "Goog. Now enter login.")); err != nil {
		s.logger.Panic(err)
	}
}

func (s *messageService) AskPassword(chatId int64) {
	if _, err := s.botApi.Send(tgbotapi.NewMessage(chatId, "Goog. Now enter password.")); err != nil {
		s.logger.Panic(err)
	}
}

func (s *messageService) AskNewNameFromData(chatId int64) {
	if _, err := s.botApi.Send(tgbotapi.NewMessage(chatId, "Please enter new name.")); err != nil {
		s.logger.Panic(err)
	}
}
