package bot

import (
	"context"
	"errors"
	"fmt"
	"log"
	"password-manager-bot/pkg"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
)

type Client interface {
	StartBot()
}

type client struct {
	botApi     *tgbotapi.BotAPI
	repository Repository
	logger     *zap.SugaredLogger
}

func NewClient(botApi *tgbotapi.BotAPI, repository Repository, logger *zap.SugaredLogger) (Client, error) {
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

func (s *client) StartBot() {
	s.botApi.Debug = true

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := s.botApi.GetUpdatesChan(u)

	user_state := make(map[int64]*UserState)

	var numericKeyboard = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Yes", "yes"),
			tgbotapi.NewInlineKeyboardButtonData("No", "no"),
		),
	)

	for update := range updates {
		if update.Message != nil { // If we got a message

			if update.Message == nil { // ignore any non-Message updates
				if _, err := s.botApi.Request(tgbotapi.NewDeleteMessage(update.Message.Chat.ID, update.Message.MessageID)); err != nil {
					s.logger.Panic(err)
				}
				continue
			}

			if !update.Message.IsCommand() { // ignore any non-command Messages
				if user, ok := user_state[update.Message.Chat.ID]; ok {
					switch user_state[update.Message.Chat.ID].State {
					case "from_question":
						if _, err := s.botApi.Request(tgbotapi.NewDeleteMessage(update.Message.Chat.ID, update.Message.MessageID)); err != nil {
							s.logger.Panic(err)
						}
						continue
					case "from":
						if _, err := s.botApi.Request(tgbotapi.NewDeleteMessage(update.Message.Chat.ID, update.Message.MessageID)); err != nil {
							s.logger.Panic(err)
						}
						user.UpdateFrom(update.Message.Text)

						dbUser, err := s.repository.GetUser(context.Background(), bson.M{"telegram_id": update.Message.Chat.ID})
						if err != nil {
							if _, err := s.botApi.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Something wrong. Please try later.")); err != nil {
								s.logger.Panic(err)
							}
						}

						if _, ok := dbUser.Data[update.Message.Text]; ok {
							user.UpdateState("from_question")
							msg := tgbotapi.NewMessage(update.Message.Chat.ID, "You already have this name. Do you want replace?")
							msg.ReplyMarkup = numericKeyboard
							if _, err := s.botApi.Send(msg); err != nil {
								s.logger.Panic(err)
							}
							continue
						}

						user.UpdateState("pin")

						if _, err := s.botApi.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Goog. Now enter pin code. You can use one pin code for all passwords or one pin code for one password.")); err != nil {
							s.logger.Panic(err)
						}
						continue
					case "pin":
						user.UpdatePin(update.Message.Text)
						user.UpdateState("login")

						if _, err := s.botApi.Request(tgbotapi.NewDeleteMessage(update.Message.Chat.ID, update.Message.MessageID)); err != nil {
							s.logger.Panic(err)
						}

						if _, err := s.botApi.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Goog. Now enter login.")); err != nil {
							s.logger.Panic(err)
						}
						continue
					case "login":
						user.UpdateLogin(update.Message.Text)
						user.UpdateState("password")

						if _, err := s.botApi.Request(tgbotapi.NewDeleteMessage(update.Message.Chat.ID, update.Message.MessageID)); err != nil {
							s.logger.Panic(err)
						}

						if _, err := s.botApi.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Goog. Now enter password.")); err != nil {
							s.logger.Panic(err)
						}
						continue
					case "password":
						user.UpdatePassword(update.Message.Text)

						if _, err := s.botApi.Request(tgbotapi.NewDeleteMessage(update.Message.Chat.ID, update.Message.MessageID)); err != nil {
							s.logger.Panic(err)
						}

						rawData := fmt.Sprintf("%s:%s", strings.TrimSpace(user.Login), strings.TrimSpace(user.Password))
						encryptedData, err := pkg.Encrypt(pkg.GenerateNormalSizeCode(strings.TrimSpace(user.Pin)), []byte(rawData))
						if err != nil {
							if _, err := s.botApi.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Failed encrypt data. Please try later.")); err != nil {
								s.logger.Panic(err)
							}
						}

						dbUser, err := s.repository.GetUser(context.Background(), bson.M{"telegram_id": update.Message.Chat.ID})
						if err != nil {
							if _, err := s.botApi.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Something wrong. Please try later.")); err != nil {
								s.logger.Panic(err)
							}
						} else if dbUser == nil {
							dbUser, err := NewUser(&update.Message.Chat.ID, user.From, encryptedData)
							if err != nil {
								if _, err := s.botApi.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Failed create data. Please try later.")); err != nil {
									s.logger.Panic(err)
								}
							}

							s.repository.CreatUser(context.Background(), dbUser)
						}

						dbUser.AddData(user.From, encryptedData)
						err = s.repository.UpdateUser(context.Background(), dbUser)
						if err != nil {
							if _, err := s.botApi.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Something wrong. Please try later.")); err != nil {
								s.logger.Panic(err)
							}
						}

						if _, err := s.botApi.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Success. Your password has been encrypted and added.")); err != nil {
							s.logger.Panic(err)
						}

						user.Refresh()
						continue
					default:
						if _, err := s.botApi.Request(tgbotapi.NewDeleteMessage(update.Message.Chat.ID, update.Message.MessageID)); err != nil {
							s.logger.Panic(err)
						}
						// go func(chatId int64, messageId int) {
						// 	time.Sleep(10 * time.Second)
						// 	bot.Request(tgbotapi.NewDeleteMessage(chatId, messageId))
						// }(update.Message.Chat.ID, update.Message.MessageID)

						continue
					}
				} else {
					if _, err := s.botApi.Request(tgbotapi.NewDeleteMessage(update.Message.Chat.ID, update.Message.MessageID)); err != nil {
						s.logger.Panic(err)
					}
					continue
				}
			}

			// Create a new MessageConfig. We don't have text yet,
			// so we leave it empty.
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")

			// Extract the command from the Message.
			switch update.Message.Command() {
			// case "start":
			// 	user_state[update.Message.From.ID] = User{}
			case "add":
				msg.Text = "Ok. Let's start. First step enter from what password."
				user_state[update.Message.Chat.ID] = &UserState{
					State: "from",
				}
			case "delete":
				msg.Text = "Hi :)"
			case "show":
				msg.Text = "I'm ok."
			default:
				msg.Text = "I don't know that command"
			}

			if _, err := s.botApi.Send(msg); err != nil {
				log.Panic(err)
			}
		} else if update.CallbackQuery != nil {
			if _, err := s.botApi.Request(tgbotapi.NewDeleteMessage(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID)); err != nil {
				s.logger.Panic(err)
			}

			if user, ok := user_state[update.CallbackQuery.Message.Chat.ID]; ok {
				switch update.CallbackQuery.Data {
				case "yes":
					user.UpdateState("pin")

					if _, err := s.botApi.Send(tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "Goog. Now enter pin code. You can use one pin code for all passwords or one pin code for one password.")); err != nil {
						s.logger.Panic(err)
					}
				case "no":
					user.UpdateState("from")
					msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "Please enter new name.")
					if _, err := s.botApi.Send(msg); err != nil {
						s.logger.Panic(err)
					}
				}
			}
		}
	}
}
