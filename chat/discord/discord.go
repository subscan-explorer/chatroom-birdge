package discord

import (
	"chatroom/conf"
	"chatroom/model"
	"chatroom/utils"
	"context"
	"encoding/json"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"
)

type App struct {
	cli         *discordgo.Session
	Users       map[string]*model.User
	ChannelInfo map[string]*model.ChannelInfo
	lock        sync.RWMutex

	SubscriptMessage map[string][]chan model.IChatMessage
	substrateLock    sync.RWMutex

	log *log.Logger
}

var app *App

func NewClient(_ context.Context, conf conf.Discord) {
	app = new(App)
	app.log = log.New(os.Stdout, "Discord: ", log.Lshortfile|log.Ldate|log.Ltime)
	app.cli, _ = discordgo.New("Bot " + conf.Token)
	app.SubscriptMessage = make(map[string][]chan model.IChatMessage)
	app.Users = make(map[string]*model.User)
	app.ChannelInfo = make(map[string]*model.ChannelInfo)
	//app.cli.Identify.Intents = 395137247296
	if err := app.cli.Open(); err != nil {
		app.log.Fatalf("Cannot open the session: %v\n", err)
	}
	app.init()
}

func (a *App) init() {
	channelIds := conf.Conf.GetDiscordChat()
	var userIds []string
	for _, info := range a.GetChannelsInfo(channelIds...) {
		a.log.Printf("sync discord channel: %+v\n", *info)
		userIds = append(userIds, info.Members...)
	}
	userIds = utils.Unique(userIds)
	a.log.Printf("sync discord users: %v\n", userIds)
	a.GetUsersInfo(userIds...)
	a.handler()
}

func (a *App) handler() {
	a.cli.AddHandler(func(s *discordgo.Session, p *discordgo.Ready) {
		a.log.Println("Discord Bot is up!")
	})
	a.cli.AddHandler(func(s *discordgo.Session, c *discordgo.Disconnect) {
		a.log.Println("Discord Disconnection")
	})
	a.cli.AddHandler(func(s *discordgo.Session, r *discordgo.Resumed) {
		a.log.Println("Discord connection Resumed")
	})
	a.cli.AddHandler(func(s *discordgo.Session, c *discordgo.InteractionCreate) {
		a.log.Println("InteractionCreate")
	})
	a.handlerMessageEvent()
	a.handlerMessageReaction()
	a.handlerThread()
}

func (a *App) handlerMessageReaction() {
	a.cli.AddHandler(func(s *discordgo.Session, msg *discordgo.MessageReactionAdd) {
		if msg.UserID == s.State.User.ID {
			return
		}
		d, _ := json.Marshal(msg)
		a.log.Println("reaction add", string(d))
		dm := &model.DiscordMessage{
			ID:   msg.MessageID,
			Type: model.MessageTypeActionAdd,
			Channel: utils.Default(a.GetChannelInfo(msg.ChannelID), func(v *model.ChannelInfo) bool {
				return v != nil
			}, model.NewChannelInfo(msg.ChannelID)),
			EmojiData: &model.DiscordMessageEmoji{ID: msg.Emoji.ID, Name: msg.Emoji.Name},
		}

		if msg.Emoji.User != nil {
			dm.User = utils.Default(a.GetUserInfo(msg.Emoji.User.ID), func(v *model.User) bool {
				return v != nil
			}, &model.User{ID: msg.Emoji.User.ID, Name: msg.Emoji.User.Username, DisplayName: msg.Emoji.User.Username})
		} else {
			dm.User = utils.Default(a.GetUserInfo(msg.UserID), func(v *model.User) bool {
				return v != nil
			}, model.NewUserInfo(msg.UserID))
		}
		go a.ReceiveMessage(dm)
	})
	a.cli.AddHandler(func(s *discordgo.Session, msg *discordgo.MessageReactionRemoveAll) {
		if msg.UserID == s.State.User.ID {
			return
		}
		d, _ := json.Marshal(msg)
		a.log.Println("reaction remove all", string(d))
		dm := &model.DiscordMessage{
			ID:   msg.MessageID,
			Type: model.MessageTypeActionRemoveALL,
			Channel: utils.Default(a.GetChannelInfo(msg.ChannelID), func(v *model.ChannelInfo) bool {
				return v != nil
			}, model.NewChannelInfo(msg.ChannelID)),
		}
		go a.ReceiveMessage(dm)
	})
	a.cli.AddHandler(func(s *discordgo.Session, msg *discordgo.MessageReactionRemove) {
		if msg.UserID == s.State.User.ID {
			return
		}
		d, _ := json.Marshal(msg)
		a.log.Println("reaction remove", string(d))
		dm := &model.DiscordMessage{
			ID:   msg.MessageID,
			Type: model.MessageTypeActionRemove,
			Channel: utils.Default(a.GetChannelInfo(msg.ChannelID), func(v *model.ChannelInfo) bool {
				return v != nil
			}, model.NewChannelInfo(msg.ChannelID)),
			EmojiData: &model.DiscordMessageEmoji{ID: msg.Emoji.ID, Name: msg.Emoji.Name},
		}
		go a.ReceiveMessage(dm)
	})
}

func (a *App) handlerMessageEvent() {
	a.cli.AddHandler(func(s *discordgo.Session, msg *discordgo.MessageDelete) {
		// 过滤自己
		//if msg.Author.ID == s.State.User.ID {
		//	return
		//}
		dm := model.DiscordMessage{
			ID:   msg.Message.ID,
			Type: model.MessageTypeTextDelete,
			Channel: utils.Default(a.GetChannelInfo(msg.ChannelID), func(v *model.ChannelInfo) bool {
				return v != nil
			}, model.NewChannelInfo(msg.ChannelID)),
		}
		go a.ReceiveMessage(&dm)
	})
	a.cli.AddHandler(func(s *discordgo.Session, msg *discordgo.MessageUpdate) {
		// 过滤自己
		if msg.Author == nil || msg.Author.ID == s.State.User.ID {
			return
		}
		d, _ := json.Marshal(msg)
		a.log.Println("message update", string(d))
		userInfo := model.User{ID: msg.Author.ID, Name: msg.Author.Username, DisplayName: msg.Author.Username}
		a.SetUserInfo(userInfo)
		dm := model.DiscordMessage{
			ID:   msg.Message.ID,
			Type: model.MessageTypeTextUpdate,
			Channel: utils.Default(a.GetChannelInfo(msg.ChannelID), func(v *model.ChannelInfo) bool {
				return v != nil
			}, model.NewChannelInfo(msg.ChannelID)),
			User:     &userInfo,
			SendTime: msg.Timestamp.UnixNano(),
		}
		dm.RawMessage = msg.Content
		if text, err := msg.ContentWithMoreMentionsReplaced(a.cli); err != nil {
			dm.Message = msg.ContentWithMentionsReplaced()
		} else {
			dm.Message = text
		}
		for _, attachment := range msg.Attachments {
			dm.Attachments = append(dm.Attachments, model.Attachment{
				Type: attachment.ContentType,
				URL:  attachment.URL,
			})
		}
		go a.ReceiveMessage(&dm)
	})
	a.cli.AddHandler(func(s *discordgo.Session, msg *discordgo.MessageCreate) {
		// 过滤自己
		if msg.Author == nil || msg.Author.ID == s.State.User.ID {
			return
		}
		d, _ := json.Marshal(msg)
		a.log.Println("receive message,", string(d))
		userInfo := model.User{ID: msg.Author.ID, Name: msg.Author.Username, DisplayName: msg.Author.Username}
		a.SetUserInfo(userInfo)
		dm := &model.DiscordMessage{
			ID:   msg.ID,
			Type: model.MessageTypeTextCreate,
			Channel: utils.Default(a.GetChannelInfo(msg.ChannelID), func(v *model.ChannelInfo) bool {
				return v != nil
			}, model.NewChannelInfo(msg.ChannelID)),
			User:     &userInfo,
			SendTime: msg.Timestamp.UnixNano(),
		}
		dm.RawMessage = msg.Content
		if text, err := msg.ContentWithMoreMentionsReplaced(a.cli); err != nil {
			dm.Message = msg.ContentWithMentionsReplaced()
		} else {
			dm.Message = text
		}
		for _, attachment := range msg.Attachments {
			dm.Attachments = append(dm.Attachments, model.Attachment{
				Type: attachment.ContentType,
				URL:  attachment.URL,
			})
		}
		if msg.MessageReference != nil && msg.Type == discordgo.MessageTypeReply {
			// 回复消息
			dm.Type = model.MessageTypeTextReply
			dm.ParentID = msg.MessageReference.MessageID
		}
		go a.ReceiveMessage(dm)
	})
}

func (a *App) handlerThread() {
	a.cli.AddHandler(func(s *discordgo.Session, c *discordgo.ThreadCreate) {})
	a.cli.AddHandler(func(s *discordgo.Session, c *discordgo.ThreadDelete) {})
}

func (a *App) ReceiveMessage(msg *model.DiscordMessage) {
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

func (a *App) SetUserInfo(user model.User) {
	a.lock.Lock()
	a.Users[user.ID] = &user
	a.lock.Unlock()
}

func (a *App) SearchUserName(name string) *model.User {
	var result *model.User
	a.lock.RLock()
	for _, user := range a.Users {
		if strings.EqualFold(user.UName(), name) {
			result = user
		}
	}
	a.lock.RUnlock()
	return result
}

func (a *App) GetUserInfo(userID string) *model.User {
	if v := a.GetUsersInfo(userID); len(v) != 0 {
		return v[userID]
	}
	return nil
}

func (a *App) GetUsersInfo(userIds ...string) map[string]*model.User {
	if len(userIds) == 0 {
		return nil
	}
	a.lock.RLock()
	var unknownUsers []string
	var result = make(map[string]*model.User)
	for _, id := range userIds {
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
	users := a.getUserInfo(unknownUsers...)
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
