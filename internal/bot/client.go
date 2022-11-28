package bot

import (
	"errors"
	"fmt"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

type Client interface {
	StartBot(updates tgbotapi.UpdatesChannel)
}

type client struct {
	botSvc     Service
	messageSvc MessageService
	logger     *zap.SugaredLogger
}

func NewClient(botSvc Service, messageSvc MessageService, logger *zap.SugaredLogger) (Client, error) {
	if botSvc == nil {
		return nil, errors.New("invalid bot service")
	}
	if messageSvc == nil {
		return nil, errors.New("invalid message service")
	}
	if logger == nil {
		return nil, errors.New("invalid logger")
	}

	return &client{botSvc: botSvc, messageSvc: messageSvc, logger: logger}, nil
}

func (c *client) StartBot(updates tgbotapi.UpdatesChannel) {
	user_state := make(map[int64]*UserState)

	for update := range updates {
		if update.Message != nil {
			if update.Message == nil {
				c.messageSvc.DeleteMessage(update.Message.Chat.ID, update.Message.MessageID)
				continue
			}

			if !update.Message.IsCommand() {
				if user, ok := user_state[update.Message.Chat.ID]; ok {
					switch user_state[update.Message.Chat.ID].State {
					case "from_question":
						c.messageSvc.DeleteMessage(update.Message.Chat.ID, update.Message.MessageID)
						continue
					case "from":
						c.messageSvc.DeleteMessage(update.Message.Chat.ID, update.Message.MessageID)

						user.UpdateFrom(update.Message.Text)

						ok, err := c.botSvc.CheckDuplicateFromWhatData(*user, update.Message.Chat.ID, update.Message.Text)
						if err != nil {
							c.messageSvc.SendWrongMessage(update.Message.Chat.ID)
							continue
						}

						if ok {
							user.UpdateState("from_question")
							c.messageSvc.SendAlreadyHaveNameWithKeyboard(update.Message.Chat.ID)
							continue
						}

						user.UpdateState("pin-encrypt")

						c.messageSvc.AskPin(update.Message.Chat.ID, true)
						continue
					case "pin-encrypt":
						c.messageSvc.DeleteMessage(update.Message.Chat.ID, update.Message.MessageID)

						user.UpdatePin(update.Message.Text)
						user.UpdateState("login")

						c.messageSvc.AskLogin(update.Message.Chat.ID)
						continue
					case "pin-decrypt":
						c.messageSvc.DeleteMessage(update.Message.Chat.ID, update.Message.MessageID)

						user.UpdatePin(update.Message.Text)

						dec, err := c.botSvc.DecryptData(update.Message.Chat.ID, user.Pin, user.From)
						if err != nil {
							c.messageSvc.SendWrongMessage(update.Message.Chat.ID)
							continue
						}

						msg := c.messageSvc.SendManualMessage(tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("🟠 NOTICE: This message will be delete in 10 seconds. \nYour data: %q", *dec)))

						go func(chatId int64, messageId int) {
							time.Sleep(10 * time.Second)
							c.messageSvc.DeleteMessage(chatId, messageId)
							c.messageSvc.SendManualMessage(tgbotapi.NewMessage(chatId, "Thanks for using.😌"))
						}(update.Message.Chat.ID, msg.MessageID)

						user.Refresh()
						continue
					case "login":
						c.messageSvc.DeleteMessage(update.Message.Chat.ID, update.Message.MessageID)

						user.UpdateLogin(update.Message.Text)
						user.UpdateState("password")

						c.messageSvc.AskPassword(update.Message.Chat.ID)
						continue
					case "password":
						c.messageSvc.DeleteMessage(update.Message.Chat.ID, update.Message.MessageID)

						user.UpdatePassword(update.Message.Text)

						encryptedData, err := c.botSvc.EncryptData(update.Message.Chat.ID, *user)
						if err != nil {
							c.messageSvc.SendWrongMessage(update.Message.Chat.ID)
							continue
						}

						err = c.botSvc.UpdateUserEncryptedData(update.Message.Chat.ID, user.From, *encryptedData)
						if err != nil {
							c.messageSvc.SendWrongMessage(update.Message.Chat.ID)
							continue
						}

						c.messageSvc.SendSuccessMessage(update.Message.Chat.ID)

						user.Refresh()
						continue
					default:
						c.messageSvc.DeleteMessage(update.Message.Chat.ID, update.Message.MessageID)
						continue
					}
				} else {
					c.messageSvc.DeleteMessage(update.Message.Chat.ID, update.Message.MessageID)
					continue
				}
			}

			// Extract the command from the Message.
			switch update.Message.Command() {
			case "start":
				c.messageSvc.SendWelcomeMessage(update.Message.Chat.ID)

				err := c.botSvc.CreateUser(update.Message.Chat.ID)
				if err != nil {
					if mongo.IsDuplicateKeyError(err) {
						break
					}

					c.messageSvc.SendWrongMessage(update.Message.Chat.ID)
				}
			case "enc":
				user_state[update.Message.Chat.ID] = &UserState{
					State: "from",
				}
				c.messageSvc.SendStartEncryptProcess(update.Message.Chat.ID)
			case "dec":
				userDataNameChunks, err := c.botSvc.GetUserDataNamesByChunks(update.Message.Chat.ID)
				if err != nil {
					c.messageSvc.SendWrongMessage(update.Message.Chat.ID)
					break
				}

				if userDataNameChunks == nil {
					c.messageSvc.SendDoNotHaveData(update.Message.Chat.ID)
					break
				}

				user_state[update.Message.Chat.ID] = &UserState{
					State: "decrypt",
				}

				c.messageSvc.AskWhatDecrypt(update.Message.Chat.ID, userDataNameChunks)
			// case "del":
			// 	c.messageSvc.SendMessage(tgbotapi.NewMessage(update.Message.Chat.ID, "I'm ok."))
			default:
				c.messageSvc.SendIncorrectCommand(update.Message.Chat.ID)
			}
		} else if update.CallbackQuery != nil {
			c.messageSvc.DeleteMessage(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID)

			if user, ok := user_state[update.CallbackQuery.Message.Chat.ID]; ok {
				if user.State == "decrypt" {
					user.UpdateFrom(update.CallbackQuery.Data)
					user.UpdateState("pin-decrypt")
					c.messageSvc.AskPin(update.CallbackQuery.Message.Chat.ID, false)
				}

				switch update.CallbackQuery.Data {
				case "yes":
					user.UpdateState("pin-encrypt")

					c.messageSvc.AskPin(update.CallbackQuery.Message.Chat.ID, true)
				case "no":
					user.UpdateState("from")

					c.messageSvc.AskNewNameFromData(update.CallbackQuery.Message.Chat.ID)
				}
			}
		}
	}
}
