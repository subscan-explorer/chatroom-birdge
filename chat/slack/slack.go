package slack

import (
	"chatroom/conf"
	"chatroom/emoji"
	"chatroom/model"
	"chatroom/utils"
	"context"
	"encoding/json"
	"log"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

type baseInfo struct {
	TeamID      string
	SelfID      string
	BotID       string
	Users       map[string]*model.User
	ChannelInfo map[string]*model.ChannelInfo

	SubscriptMessage map[string][]chan model.IChatMessage
	substrateLock    sync.RWMutex
	lock             sync.RWMutex
}

type App struct {
	baseInfo
	cli *socketmode.Client
	log *log.Logger
}

var app *App

func NewClient(ctx context.Context, conf conf.Slack) {
	app = new(App)
	app.log = log.New(os.Stdout, "Slack: ", log.Lshortfile|log.Ldate|log.Ltime)
	app.Users = make(map[string]*model.User)
	app.ChannelInfo = make(map[string]*model.ChannelInfo)
	app.SubscriptMessage = make(map[string][]chan model.IChatMessage)
	app.cli = socketmode.New(slack.New(conf.Token,
		slack.OptionDebug(false),
		slack.OptionAppLevelToken(conf.AppLevelToken),
		slack.OptionLog(app.log)),
		socketmode.OptionDebug(false),
		socketmode.OptionLog(app.log))

	rsp, err := app.cli.AuthTestContext(ctx)
	if err != nil {
		app.log.Panicln(err.Error())
	}
	app.log.Printf("connection info: %+v", rsp)
	app.TeamID = rsp.TeamID
	app.SelfID = rsp.UserID
	app.BotID = rsp.BotID
	app.init()
	if err = app.eventLoop(ctx); err != nil {
		app.log.Panicln(err.Error())
	}
}

func (c *App) init() {
	channelIds := conf.Conf.GetSlackChat()
	var userIds []string
	for _, info := range c.GetChannelsInfo(channelIds...) {
		c.log.Printf("sync channel: %+v\n", *info)
		userIds = append(userIds, info.Members...)
	}
	userIds = utils.Unique(userIds)
	c.log.Printf("sync users: %v\n", userIds)
	c.GetUsersInfo(userIds...)
}

func (c *App) eventLoop(ctx context.Context) error {
	handler := socketmode.NewSocketmodeHandler(c.cli)
	handler.Handle(socketmode.EventTypeConnecting, func(event *socketmode.Event, client *socketmode.Client) {
		c.log.Println("connecting")
	})
	handler.Handle(socketmode.EventTypeConnectionError, func(event *socketmode.Event, client *socketmode.Client) {
		c.log.Println("Connection failed. Retrying later...")
	})
	handler.Handle(socketmode.EventTypeHello, func(event *socketmode.Event, client *socketmode.Client) {
		//for _, id := range c.getChannelIds() {
		//	if channel, _, err := client.PostMessage(id, slack.MsgOptionText("sync message online", true)); err != nil {
		//		log.Printf("failed to send message. channel [%s], err: %s", channel, err.Error())
		//	}
		//}
		c.log.Println("success receive message.")
	})
	handler.Handle(socketmode.EventTypeConnected, func(event *socketmode.Event, client *socketmode.Client) {
		c.log.Println("Connected")
	})
	handler.Handle(socketmode.EventTypeEventsAPI, func(event *socketmode.Event, client *socketmode.Client) {
		apiEvent, ok := event.Data.(slackevents.EventsAPIEvent)
		if !ok {
			return
		}
		c.cli.Ack(*event.Request)
		c.handlerMessage(apiEvent)
	})
	handler.Handle(socketmode.EventTypeSlashCommand, func(event *socketmode.Event, client *socketmode.Client) {
		cmd, ok := event.Data.(slack.SlashCommand)
		if !ok {
			c.log.Printf("Ignored %+v\n", event)
			return
		}
		client.Debugf("Slash command received: %+v", cmd)
	})
	//handler.HandleSlashCommand("/users", func(event *socketmode.Event, client *socketmode.Client) {
	//	cmd, ok := event.Data.(slack.SlashCommand)
	//	if !ok {
	//		log.Printf("Ignored %+v\n", event)
	//		return
	//	}
	//	client.Debugf("Slash command received: %+v", cmd)
	//})
	go func() {
		if err := handler.RunEventLoopContext(ctx); err != nil {
			c.log.Println(err.Error())
		}
	}()
	return nil
}

func (c *App) ReceiveMessage(msg *model.SlackMessage) {
	var chs []chan model.IChatMessage
	c.substrateLock.RLock()
	chs = append(chs, c.SubscriptMessage[msg.Channel.CID()]...)
	c.substrateLock.RUnlock()
	for _, ch := range chs {
		ch <- msg
	}
}

func (c *App) RegisterChannel(channelID string, ch chan model.IChatMessage) {
	c.substrateLock.Lock()
	c.SubscriptMessage[channelID] = append(c.SubscriptMessage[channelID], ch)
	c.substrateLock.Unlock()
}

func (c *App) handlerMessage(event slackevents.EventsAPIEvent) {
	switch event.Type {
	case slackevents.CallbackEvent:
		innerEvent := event.InnerEvent
		switch ev := innerEvent.Data.(type) {
		case *slackevents.AppMentionEvent:
			_, _, err := c.cli.PostMessage(ev.Channel, slack.MsgOptionText("Yes, hello.", false))
			if err != nil {
				c.log.Printf("failed posting message: %v", err)
			}
		case *slackevents.MemberJoinedChannelEvent:
			c.log.Printf("user %q joined to channel %q", ev.User, ev.Channel)
		case *slackevents.ChannelLeftEvent:
			c.log.Printf(" %q left to channel %q", ev.EventTimestamp, ev.Channel)
		case *slackevents.MessageEvent:
			switch ev.ChannelType {
			case "channel", "group": // 公共频道 group: 私人频道
				msg := new(model.SlackMessage)
				msg.Channel = utils.Default(c.GetChannelInfo(ev.Channel), func(v *model.ChannelInfo) bool { return v != nil }, model.NewChannelInfo(ev.Channel))
				switch ev.SubType {
				case "message_changed":
					msg.Type = model.MessageTypeTextUpdate
					if ev.Message == nil {
						c.log.Println("receive update message not found message")
						return
					}
					if ev.Message.User == c.SelfID {
						return // 跳过服务自身消息
					}
					msg.ID = ev.Message.TimeStamp
					msg.User = utils.Default(c.GetUserInfo(ev.Message.User), func(v *model.User) bool { return v != nil }, model.NewUserInfo(ev.User))
					msg.Message = c.ContentWithEmojiReplaced(c.ContentWithMentionsReplaced(ev.Message.Text))
					msg.RawMessage = ev.Message.Text
					for _, file := range ev.Message.Files {
						msg.Attachments = append(msg.Attachments, model.Attachment{
							Name: file.Name,
							Type: file.Mimetype,
							URL:  file.URLPrivate,
						})
					}
				case "message_deleted":
					msg.Type = model.MessageTypeTextDelete
					if ev.PreviousMessage == nil {
						c.log.Println("receive delete message not found previous message")
						return
					}
					if ev.PreviousMessage.User == c.SelfID {
						return // 跳过服务自身消息
					}
					msg.ID = ev.PreviousMessage.TimeStamp
				default:
					if ev.User == c.SelfID {
						return // 跳过服务自身发送的消息
					}
					msg.ID = ev.TimeStamp
					msg.Type = model.MessageTypeTextCreate
					msg.User = utils.Default(c.GetUserInfo(ev.User), func(v *model.User) bool { return v != nil }, model.NewUserInfo(ev.User))
					msg.Message = c.ContentWithEmojiReplaced(c.ContentWithMentionsReplaced(ev.Text))
					msg.RawMessage = ev.Text
					msg.SendTime = utils.ParseSlackTimestamp(ev.TimeStamp)
					if len(ev.ThreadTimeStamp) != 0 {
						msg.Type = model.MessageTypeTextReply
						msg.ParentID = ev.ThreadTimeStamp
					}
					for _, file := range ev.Files {
						msg.Attachments = append(msg.Attachments, model.Attachment{
							Name: file.Name,
							Type: file.Mimetype,
							URL:  file.URLPrivate,
						})
					}
				}
				go c.ReceiveMessage(msg)
			default: // im | mim
				return
			}
			d, _ := json.Marshal(innerEvent)
			c.cli.Debugf("receive %s of to channel %s[%s] message: %s, raw: %s\n", ev.User, ev.Channel, ev.ChannelType, ev.Text, string(d))
			c.log.Printf("receive %s of to channel %s[%s] message: %s, raw: %s\n", ev.User, ev.Channel, ev.ChannelType, ev.Text, string(d))

		case *slackevents.ReactionAddedEvent:
			if ev.User == c.SelfID {
				return // 跳过服务自身消息
			}
			msg := new(model.SlackMessage)
			msg.ID = ev.Item.Timestamp
			msg.Type = model.MessageTypeActionAdd
			msg.Channel = utils.Default(c.GetChannelInfo(ev.Item.Channel), func(v *model.ChannelInfo) bool { return v != nil }, model.NewChannelInfo(ev.Item.Channel))
			msg.User = utils.Default(c.GetUserInfo(ev.User), func(v *model.User) bool { return v != nil }, model.NewUserInfo(ev.User))
			msg.Reaction = &model.SlackReaction{
				Data: ev.Reaction,
			}
			go c.ReceiveMessage(msg)
			c.log.Printf("receive reaction add: %+#v\n", ev)
		case *slackevents.ReactionRemovedEvent:
			if ev.User == c.SelfID {
				return // 跳过服务自身消息
			}
			msg := new(model.SlackMessage)
			msg.ID = ev.Item.Timestamp
			msg.Type = model.MessageTypeActionRemove
			msg.Channel = utils.Default(c.GetChannelInfo(ev.Item.Channel), func(v *model.ChannelInfo) bool { return v != nil }, model.NewChannelInfo(ev.Item.Channel))
			msg.User = utils.Default(c.GetUserInfo(ev.User), func(v *model.User) bool { return v != nil }, model.NewUserInfo(ev.User))
			msg.Reaction = &model.SlackReaction{
				Data: ev.Reaction,
			}
			go c.ReceiveMessage(msg)
			c.log.Printf("receive reaction remove: %+#v\n", ev)
		default:
			c.cli.Debugf("unsupported Callback Events API %s received", innerEvent.Type)
		}
	case slackevents.AppRateLimited:
		c.cli.Debugf("rate limit")
	default:
		c.cli.Debugf("unsupported Events API %s received", event.Type)
	}
}

func (c *App) getChannelIds() []string {
	c.lock.RLock()
	result := make([]string, 0, len(c.ChannelInfo))
	for id := range c.ChannelInfo {
		result = append(result, id)
	}
	c.lock.RUnlock()
	return result
}

func (c *App) GetChannelInfo(channelID string) *model.ChannelInfo {
	if v := c.GetChannelsInfo(channelID); len(v) != 0 {
		return v[channelID]
	}
	return nil
}

func (c *App) GetChannelsInfo(channelIds ...string) map[string]*model.ChannelInfo {
	if len(channelIds) == 0 {
		return nil
	}
	c.lock.RLock()
	var unknownChannels []string
	var result = make(map[string]*model.ChannelInfo, len(channelIds))
	for _, id := range channelIds {
		if v := c.ChannelInfo[id]; v != nil {
			result[id] = v
		} else {
			unknownChannels = append(unknownChannels, id)
		}
	}
	c.lock.RUnlock()
	if len(unknownChannels) == 0 {
		return result
	}
	var channels []model.ChannelInfo
	for _, id := range unknownChannels {
		info, err := c.getChannelInfo(id)
		if err != nil {
			c.log.Printf("failed to get channel info by %s\n", id)
			continue
		}
		channels = append(channels, *info)
	}
	if len(channels) == 0 {
		return result
	}
	for i := range channels {
		result[channels[i].ID] = &channels[i]
	}
	if c.lock.TryLock() {
		for i := range channels {
			c.ChannelInfo[channels[i].ID] = &channels[i]
		}
		c.lock.Unlock()
	} else {
		go func() {
			c.lock.Lock()
			for i := range channels {
				c.ChannelInfo[channels[i].ID] = &channels[i]
			}
			c.lock.Unlock()
		}()
	}
	return result
}

func (c *App) SearchUserName(name string) *model.User {
	var result *model.User
	c.lock.RLock()
	for _, user := range c.Users {
		if strings.EqualFold(user.UName(), name) {
			result = user
		}
	}
	c.lock.RUnlock()
	return result
}

func (c *App) GetUserInfo(userID string) *model.User {
	if v := c.GetUsersInfo(userID); len(v) != 0 {
		return v[userID]
	}
	return nil
}

func (c *App) GetUsersInfo(userIds ...string) map[string]*model.User {
	if len(userIds) == 0 {
		return nil
	}
	c.lock.RLock()
	var unknownUsers []string
	var result = make(map[string]*model.User)
	for _, id := range userIds {
		if v := c.Users[id]; v != nil {
			result[id] = v
		} else {
			unknownUsers = append(unknownUsers, id)
		}
	}
	c.lock.RUnlock()
	if len(unknownUsers) == 0 {
		return result
	}
	users, err := c.getUserInfo(unknownUsers...)
	if err == nil && len(users) > 0 {
		for i := range users {
			result[users[i].ID] = &users[i]
		}
		if c.lock.TryLock() {
			for i := range users {
				c.Users[users[i].ID] = &users[i]
			}
			c.lock.Unlock()
		} else {
			go func() {
				c.lock.Lock()
				for i := range users {
					c.Users[users[i].ID] = &users[i]
				}
				c.lock.Unlock()
			}()
		}
	}
	return result
}

func (c *App) ContentWithMentionsReplaced(text string) string {
	rgx := regexp.MustCompile(`<@([^@\\s]*)\\s>`)
	userID := rgx.FindAllStringSubmatch(text, -1)
	var args []string
	for _, id := range userID {
		if len(id) != 2 {
			continue
		}
		user := c.GetUserInfo(id[1])
		if user == nil {
			user = model.NewUserInfo(id[1])
		}
		args = append(args, id[0], "@"+user.UName())
	}
	return strings.NewReplacer(args...).Replace(text)
}

func (c *App) ContentWithEmojiReplaced(text string) string {
	rgx := regexp.MustCompile(":([^:a-z0-9_+-]*):")
	e := rgx.FindAllStringSubmatch(text, -1)
	var args []string
	for _, id := range e {
		if len(id) != 2 {
			continue
		}
		if ej := emoji.SlackConvertEmoji(id[1]); len(ej) != 0 {
			args = append(args, id[0], ej)
		}
	}
	return strings.NewReplacer(args...).Replace(text)
}
