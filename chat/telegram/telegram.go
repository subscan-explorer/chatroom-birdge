package telegram

import (
	"chatroom/conf"
	"chatroom/model"
	"context"
	"encoding/json"
	"log"
	"os"
	"strconv"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type App struct {
	cli         *tgbotapi.BotAPI
	Users       map[string]*model.User
	ChannelInfo map[string]*model.ChannelInfo
	lock        sync.RWMutex

	SubscriptMessage map[string][]chan model.IChatMessage
	substrateLock    sync.RWMutex

	log *log.Logger
}

var app *App

func NewClient(_ context.Context, conf conf.Telegram) {
	if len(conf.Token) == 0 {
		return
	}
	app = new(App)
	app.log = log.New(os.Stdout, "Telegram: ", log.Lshortfile|log.Ldate|log.Ltime)
	cli, err := tgbotapi.NewBotAPI(conf.Token)
	if err != nil {
		app.log.Fatalf("Cannot connection the telegram: %v\n", err)
	}
	app.cli = cli
	app.SubscriptMessage = make(map[string][]chan model.IChatMessage)
	app.Users = make(map[string]*model.User)
	app.ChannelInfo = make(map[string]*model.ChannelInfo)
	//app.cli.Debug = true
	app.log.Printf("Authorized on account %s", app.cli.Self.UserName)
	go app.init()
}

func (a *App) init() {
	channelIDs := conf.Conf.GetTelegramChat()
	a.getChannelInfo(channelIDs...)
	//a.getUserInfo()
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30
	updates := a.cli.GetUpdatesChan(u)
	for update := range updates {
		d, _ := json.Marshal(update)
		a.log.Printf("receive message: %s", string(d))
		a.log.Printf("%+#v", update)
		a.handlerMessage(update)
	}
}

func (a *App) handlerMessage(msg tgbotapi.Update) {
	if msg.Message != nil {
		message := new(model.TelegramMessage)
		if chat := msg.FromChat(); chat != nil {
			message.Channel = &model.ChannelInfo{ID: intToString(chat.ID), Name: chat.Title}
		}
		if user := msg.SentFrom(); user != nil {
			message.User = &model.User{ID: intToString(user.ID), Name: user.String(), DisplayName: user.String()}
		}
		message.ID = msg.Message.MessageID
		message.Type = model.MessageTypeTextCreate
		if reply := msg.Message.ReplyToMessage; reply != nil {
			message.Type = model.MessageTypeTextReply
			message.ParentID = msg.Message.ReplyToMessage.MessageID
		}
		message.Message = msg.Message.Text
		message.Attachments = a.Attachment(msg.Message)
		if len(message.Message) == 0 {
			message.Message = msg.Message.Caption
		}
		go a.ReceiveMessage(message)
	}
	if msg.EditedMessage != nil {
		message := new(model.TelegramMessage)
		if chat := msg.FromChat(); chat != nil {
			message.Channel = &model.ChannelInfo{ID: intToString(chat.ID), Name: chat.Title}
		}
		if user := msg.SentFrom(); user != nil {
			message.User = &model.User{ID: intToString(user.ID), Name: user.String(), DisplayName: user.String()}
		}
		message.ID = msg.EditedMessage.MessageID
		message.Type = model.MessageTypeTextUpdate
		message.Message = msg.EditedMessage.Text
		go a.ReceiveMessage(message)
	}
}

func (a *App) ReceiveMessage(msg *model.TelegramMessage) {
	var chs []chan model.IChatMessage
	a.substrateLock.RLock()
	chs = append(chs, a.SubscriptMessage[msg.Channel.CID()]...)
	a.substrateLock.RUnlock()
	for _, ch := range chs {
		ch <- msg
	}
}

func (a *App) RegisterChannel(channelID string, ch chan model.IChatMessage) {
	a.substrateLock.Lock()
	a.SubscriptMessage[channelID] = append(a.SubscriptMessage[channelID], ch)
	a.substrateLock.Unlock()
}

func (a *App) GetChannelInfo(channelID string) *model.ChannelInfo {
	if v := a.GetChannelsInfo(channelID); len(v) != 0 {
		return v[channelID]
	}
	return nil
}

func (a *App) GetChannelsInfo(channelID ...string) map[string]*model.ChannelInfo {
	if len(channelID) == 0 {
		return nil
	}
	a.lock.RLock()
	var unknownChannels []string
	var result = make(map[string]*model.ChannelInfo)
	for _, id := range channelID {
		if v := a.ChannelInfo[id]; v != nil {
			result[id] = v
		} else {
			unknownChannels = append(unknownChannels, id)
		}
	}
	a.lock.RUnlock()
	if len(unknownChannels) == 0 {
		return result
	}
	channels := a.getChannelInfo(unknownChannels...)
	if len(channels) > 0 {
		for i := range channels {
			result[channels[i].ID] = &channels[i]
		}
		if a.lock.TryLock() {
			for i := range channels {
				a.ChannelInfo[channels[i].ID] = &channels[i]
			}
			a.lock.Unlock()
		} else {
			go func() {
				a.lock.Lock()
				for i := range channels {
					a.ChannelInfo[channels[i].ID] = &channels[i]
				}
				a.lock.Unlock()
			}()
		}
	}
	return result
}

func (a *App) GetUserInfo(channelID, userID string) *model.User {
	if v := a.GetUsersInfo(channelID, userID); len(v) != 0 {
		return v[userID]
	}
	return nil
}

func (a *App) GetUsersInfo(channelID string, userIDs ...string) map[string]*model.User {
	if len(userIDs) == 0 {
		return nil
	}
	a.lock.RLock()
	var unknownUsers []string
	var result = make(map[string]*model.User)
	for _, id := range userIDs {
		if v := a.Users[id]; v != nil {
			result[id] = v
		} else {
			unknownUsers = append(unknownUsers, id)
		}
	}
	a.lock.RUnlock()
	if len(unknownUsers) == 0 {
		return result
	}
	users := a.getUserInfo(channelID, unknownUsers...)
	if len(users) > 0 {
		for i := range users {
			result[users[i].ID] = &users[i]
		}
		if a.lock.TryLock() {
			for i := range users {
				a.Users[users[i].ID] = &users[i]
			}
			a.lock.Unlock()
		} else {
			go func() {
				a.lock.Lock()
				for i := range users {
					a.Users[users[i].ID] = &users[i]
				}
				a.lock.Unlock()
			}()
		}
	}
	return result
}

func intToString(id int64) string {
	return strconv.FormatInt(id, 10)
}

func stringToInt(id string) int64 {
	i, _ := strconv.ParseInt(id, 10, 64)
	return i
}
