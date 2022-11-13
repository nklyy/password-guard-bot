package bot

import (
	"errors"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
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
		if update.Message != nil { // If we got a message

			if update.Message == nil { // ignore any non-Message updates
				c.messageSvc.DeleteMessage(update.Message.Chat.ID, update.Message.MessageID)
				continue
			}

			if !update.Message.IsCommand() { // ignore any non-command Messages
				if user, ok := user_state[update.Message.Chat.ID]; ok {
					switch user_state[update.Message.Chat.ID].State {
					case "from_question":
						c.messageSvc.DeleteMessage(update.Message.Chat.ID, update.Message.MessageID)
						continue
					case "from":
						c.messageSvc.DeleteMessage(update.Message.Chat.ID, update.Message.MessageID)

						user.UpdateFrom(update.Message.Text)

						if ok := c.botSvc.CheckDuplicateFromWhat(*user, update.Message.Chat.ID, update.Message.Text); ok {
							user.UpdateState("from_question")
							c.messageSvc.SendAlreadyHaveNameWithKeyboard(update.Message.Chat.ID)
							continue
						}

						user.UpdateState("pin")

						c.messageSvc.AskPin(update.Message.Chat.ID)
						continue
					case "pin":
						c.messageSvc.DeleteMessage(update.Message.Chat.ID, update.Message.MessageID)

						user.UpdatePin(update.Message.Text)
						user.UpdateState("login")

						c.messageSvc.AskLogin(update.Message.Chat.ID)
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

						c.botSvc.CreateOrUpdateUserData(update.Message.Chat.ID, *user)

						c.messageSvc.SendSuccessMessage(update.Message.Chat.ID)

						user.Refresh()
						continue
					default:
						c.messageSvc.DeleteMessage(update.Message.Chat.ID, update.Message.MessageID)
						// go func(chatId int64, messageId int) {
						// 	time.Sleep(10 * time.Second)
						// 	bot.Request(tgbotapi.NewDeleteMessage(chatId, messageId))
						// }(update.Message.Chat.ID, update.Message.MessageID)

						continue
					}
				} else {
					c.messageSvc.DeleteMessage(update.Message.Chat.ID, update.Message.MessageID)
					continue
				}
			}

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
			// case "delete":
			// 	msg.Text = "Hi :)"
			// case "show":
			// 	msg.Text = "I'm ok."
			default:
				msg.Text = "Incorrect command!"
			}

			c.messageSvc.SendMessage(msg)
		} else if update.CallbackQuery != nil {
			c.messageSvc.DeleteMessage(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID)

			if user, ok := user_state[update.CallbackQuery.Message.Chat.ID]; ok {
				switch update.CallbackQuery.Data {
				case "yes":
					user.UpdateState("pin")

					c.messageSvc.AskPin(update.CallbackQuery.Message.Chat.ID)
				case "no":
					user.UpdateState("from")

					c.messageSvc.AskNewNameFromData(update.CallbackQuery.Message.Chat.ID)
				}
			}
		}
	}
}
