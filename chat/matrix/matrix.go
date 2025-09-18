package matrix

import (
	"chatroom/conf"
	"chatroom/model"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/crypto/cryptohelper"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
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
	cli *mautrix.Client
	log *log.Logger
}

var app *App

func NewClient(ctx context.Context, conf conf.Matrix) {
	if len(conf.Host) == 0 || len(conf.User) == 0 || len(conf.Password) == 0 {
		return
	}
	app = new(App)
	app.log = log.New(os.Stdout, "Matrix: ", log.Lshortfile|log.Ldate|log.Ltime)
	app.Users = make(map[string]*model.User)
	app.ChannelInfo = make(map[string]*model.ChannelInfo)
	app.SubscriptMessage = make(map[string][]chan model.IChatMessage)
	cli, err := mautrix.NewClient(conf.Host, "", "")
	if err != nil {
		app.log.Panicln(err.Error())
	}
	app.cli = cli

	//var loginResp *mautrix.RespLogin
	//if loginResp, err = cli.Login(ctx, &mautrix.ReqLogin{
	//	Type:             "m.login.password",
	//	Identifier:       mautrix.UserIdentifier{Type: mautrix.IdentifierTypeUser, User: conf.User},
	//	Password:         conf.Password,
	//	StoreCredentials: true,
	//}); err != nil {
	//	app.log.Panicln(err.Error())
	//}
	app.SelfID = conf.Username

	//cli.Store = mautrix.NewMemorySyncStore()
	// init sec
	cryptoHelper, err := cryptohelper.NewCryptoHelper(cli, []byte("meow"), conf.CryptoStorePath)
	if err != nil {
		panic(err)
	}
	cryptoHelper.LoginAs = &mautrix.ReqLogin{
		Type:       mautrix.AuthTypePassword,
		Identifier: mautrix.UserIdentifier{Type: mautrix.IdentifierTypeUser, User: conf.User},
		Password:   conf.Password,
	}
	err = cryptoHelper.Init(context.TODO())
	if err != nil {
		panic(err)
	}
	cli.Crypto = cryptoHelper
	app.SelfID = cli.UserID.String()
	app.init(ctx)
	if err = app.eventLoop(ctx); err != nil {
		app.log.Panicln(err.Error())
	}
}

func (a *App) init(ctx context.Context) {
	channelIds := conf.Conf.GetMatrixChat()
	a.joinRoom(ctx, channelIds...)
	a.updateChannelMember(ctx, channelIds...)
}

func (a *App) eventLoop(ctx context.Context) error {
	channel := conf.Conf.GetMatrixChat()
	var roomID []id.RoomID
	for _, s := range channel {
		roomID = append(roomID, id.RoomID(s))
	}
	syncer := a.cli.Syncer.(*mautrix.DefaultSyncer)
	filter := &mautrix.Filter{
		Room: &mautrix.RoomFilter{
			Rooms: roomID,
			State: &mautrix.FilterPart{
				NotSenders: []id.UserID{id.UserID(a.SelfID)},
				Rooms:      roomID,
			},
			Timeline: &mautrix.FilterPart{
				NotSenders: []id.UserID{id.UserID(a.SelfID)},
				Rooms:      roomID,
			},
		},
	}
	syncer.FilterJSON = filter
	nowTime := time.Now()
	syncer.OnEventType(event.EventMessage, func(ctx context.Context, evt *event.Event) {
		if v := time.UnixMilli(evt.Timestamp).Sub(nowTime); v.Seconds() < -20 {
			a.log.Println("filter message", evt.Sender, evt.RoomID, evt.Type, v.Seconds())
			return
		}
		if evt.Sender.String() == a.SelfID {
			return
		}
		a.handlerMessage(ctx, evt)
	})
	//syncer.OnEventType(event.StateMember, func(ctx context.Context, evt *event.Event) {
	//	// 成员状态
	//	data, _ := json.Marshal(evt)
	//	a.log.Println(string(data))
	//})
	syncer.OnEventType(event.EventReaction, func(ctx context.Context, evt *event.Event) {
		if v := time.UnixMilli(evt.Timestamp).Sub(nowTime); v.Seconds() < -20 {
			a.log.Println("filter message", evt.Sender, evt.RoomID, evt.Type, v.Seconds())
			return
		}
		if evt.Sender.String() == a.SelfID {
			return
		}
		a.handlerMessage(ctx, evt)
	})
	//syncer.OnEventType(event.EventEncrypted, func(ctx context.Context, evt *event.Event) {
	//	if v := time.UnixMilli(evt.Timestamp).Sub(nowTime); v.Seconds() < 10 {
	//		c.log.Println("filter encrypted message", evt.Sender, evt.RoomID, evt.Type, v.Seconds())
	//		return
	//	}
	//	data, _ := json.Marshal(evt)
	//	c.log.Println(string(data))
	//	//if evt.Sender.String() == c.SelfID {
	//	//	return
	//	//}
	//	c.handlerMessage(ctx, evt)
	//})
	go func() {
		if err := a.cli.SyncWithContext(ctx); err != nil {
			a.log.Println(err.Error())
		}
	}()
	return nil
}

func (a *App) ReceiveMessage(msg *model.MatrixMessage) {
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

func (a *App) handlerMessage(_ context.Context, evt *event.Event) {
	msg := new(model.MatrixMessage)
	switch evt.Type {
	case event.EventMessage:
		msg.ID = evt.ID.String()
		msg.Type = model.MessageTypeTextCreate
		msg.Channel = a.getChannelInfo(evt.RoomID.String())
		msg.SendTime = evt.Timestamp
		msg.User = a.getUserInfo(evt.RoomID.String(), evt.Sender.String())
		em := evt.Content.AsMessage()
		if em.RelatesTo != nil {
			if em.RelatesTo.Type == event.RelReplace {
				msg.Type = model.MessageTypeTextUpdate
				msg.ID = em.RelatesTo.GetReplaceID().String()
				if em.NewContent != nil {
					msg.Message = formatMessageBody(em.NewContent.MsgType, em.NewContent.Body)
				} else {
					msg.Message = formatMessageBody(em.MsgType, em.Body) // 回退到原始内容
				}
			} else if em.RelatesTo.InReplyTo != nil {
				msg.Type = model.MessageTypeTextReply
				msg.ParentID = em.RelatesTo.InReplyTo.EventID.String()
				msg.Message = formatMessageBody(em.MsgType, em.Body)
			} else {
				msg.Message = formatMessageBody(em.MsgType, em.Body)
				msg.Type = model.MessageTypeTextCreate
			}
		} else {
			msg.Message = formatMessageBody(em.MsgType, em.Body)
		}
		a.ReceiveMessage(msg)
	case event.EventReaction:
		msg.ID = evt.ID.String()
		msg.Type = model.MessageTypeActionAdd
		msg.Channel = a.getChannelInfo(evt.RoomID.String())
		msg.SendTime = evt.Timestamp
		msg.User = a.getUserInfo(evt.RoomID.String(), evt.Sender.String())
		if em := evt.Content.AsReaction().GetRelatesTo(); em != nil {
			if em.Type == event.RelAnnotation {
				msg.Type = model.MessageTypeActionAdd
				msg.Reaction = em.Key
				msg.ID = em.EventID.String()
			}
		}
		a.ReceiveMessage(msg)
	default:
	}
}

func formatMessageBody(tp event.MessageType, body string) string {
	if tp.IsText() {
		return body
	}
	return fmt.Sprintf("[%s] %s", strings.TrimPrefix(string(tp), "m."), body)
}
func (a *App) getChannelIds() []string {
	a.lock.RLock()
	result := make([]string, 0, len(a.ChannelInfo))
	for cid := range a.ChannelInfo {
		result = append(result, cid)
	}
	a.lock.RUnlock()
	return result
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
