package room

import (
	"chatroom/chat/discord"
	"chatroom/chat/matrix"
	"chatroom/chat/slack"
	"chatroom/chat/telegram"
	"chatroom/conf"
	"chatroom/emoji"
	"chatroom/model"
	"chatroom/utils"
	"chatroom/utils/queue"
	"context"
	"fmt"
	"log"
	"os"
	"sync"
)

type IChat interface {
	ChannelID() string
	Source() model.TypeSource
	SendMessage(model.IChatMessage) (string, error)
	SendReplyMessage(string, model.IChatMessage) (string, error)
	UpdateMessage(messageID string, message model.IChatMessage) error
	DeleteMessage(string) error
	SendReaction(messageID string, emoji string) error
	RemoveReaction(messageID string, emoji string) error
	RemoveReactionAll(messageID string) error
}

type ChatRoom struct {
	Name        string
	Room        []IChat
	Receive     chan model.IChatMessage
	MessageList *queue.List[MessageTuple]
	log         *log.Logger
}

func NewMainRoom(ctx context.Context) {
	var rooms []*ChatRoom
	for _, room := range conf.Conf.Room {
		rooms = append(rooms, NewChatRoom(ctx, room))
	}
	wg := new(sync.WaitGroup)
	for i := range rooms {
		wg.Add(1)
		go func(r *ChatRoom) {
			r.Loop(ctx)
			wg.Done()
		}(rooms[i])
	}
	log.Println("chatroom bridge running...")
	wg.Wait()
}

func NewChatRoom(_ context.Context, chat conf.Room) *ChatRoom {
	room := new(ChatRoom)
	room.Name = chat.Name
	room.MessageList = queue.NewMessageList[MessageTuple](500)
	room.log = log.New(os.Stdout, fmt.Sprintf("Room: [%s]: ", room.Name), log.Lshortfile|log.Ldate|log.Ltime)
	room.Receive = make(chan model.IChatMessage, 100*len(chat.Chat))
	for _, roomChat := range chat.Chat {
		for _, id := range roomChat.ChatID {
			switch roomChat.Type {
			case "slack":
				room.Room = append(room.Room, slack.NewSlackChat(id, room.Receive))
			case "discord":
				room.Room = append(room.Room, discord.NewDiscordChat(id, room.Receive))
			case "telegram":
				room.Room = append(room.Room, telegram.NewTelegramChat(id, room.Receive))
			case "matrix":
				room.Room = append(room.Room, matrix.NewMatrixChat(id, room.Receive))
			}
		}
	}
	return room
}

func (c *ChatRoom) Loop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-c.Receive:
			c.Dispatch(msg)
		}
	}
}

func (c *ChatRoom) Dispatch(msg model.IChatMessage) {
	// 过滤消息的来源 channel
	room := utils.FilterSlice(c.Room, func(chat IChat) bool {
		return chat.Source() == msg.Source() && chat.ChannelID() == msg.BelongChannel().CID()
	})
	defer func() {
		c.log.Printf("message queue: %s", c.MessageList.String())
	}()
	switch msg.MessageType() {
	case model.MessageTypeTextCreate:
		var tuple = MessageTuple{Type: msg.MessageType(), Message: []MessageRecord{{ID: msg.MessageID(), ChannelID: msg.BelongChannel().CID(), Source: msg.Source()}}}
		for _, chat := range room {
			c.log.Printf("dispatch message to [%s], from: %s %s %s", chat.ChannelID(), msg.BelongChannel().CName(), msg.BelongUser().UName(), msg.Text())
			id, err := chat.SendMessage(msg)
			if err != nil {
				c.log.Printf("send message to [%s] failed, from: %s %s %s", chat.ChannelID(), msg.BelongChannel().CName(), msg.BelongUser().UName(), msg.Text())
			}
			tuple.Message = append(tuple.Message, MessageRecord{ID: id, ChannelID: chat.ChannelID(), Source: chat.Source()})
		}
		c.MessageList.Push(tuple)
		// 回执
	case model.MessageTypeTextUpdate:
		origin := c.SearchMessage(msg.Source(), msg.BelongChannel().CID(), msg.MessageID())
		if origin == nil {
			origin = new(MessageTuple)
			c.log.Printf("update message failed,not found [%s] channel [%s] messageID: [%s]\n", msg.Source(), msg.BelongChannel().CID(), msg.MessageID())
			//return
		}
		for _, chat := range room {
			messageID := origin.FindMessageID(chat.Source(), chat.ChannelID())
			err := chat.UpdateMessage(messageID, msg)
			if err != nil {
				c.log.Printf("failed to update [%s] message, id: [%s], %v", chat.ChannelID(), messageID, err)
			}
		}
	case model.MessageTypeTextDelete:
		origin := c.SearchMessageDelete(msg.Source(), msg.BelongChannel().CID(), msg.MessageID())
		if origin == nil {
			c.log.Printf("delete message failed,not found [%s] channel [%s] messageID: [%s]\n", msg.Source(), msg.BelongChannel().CID(), msg.MessageID())
			return
		}
		for _, chat := range room {
			messageID := origin.FindMessageID(chat.Source(), chat.ChannelID())
			if len(messageID) == 0 {
				c.log.Printf("delete [%s] message failed,not found channel [%s] messageID\n", chat.Source(), chat.ChannelID())
				continue
			}
			err := chat.DeleteMessage(messageID)
			if err != nil {
				c.log.Printf("failed to delete [%s] message, id: [%s], %v", chat.ChannelID(), messageID, err)
			}
		}
		origin.Delete()
	case model.MessageTypeTextReply:
		origin := c.SearchMessage(msg.Source(), msg.BelongChannel().CID(), msg.ParentMessageID())
		if origin == nil {
			origin = new(MessageTuple)
			c.log.Printf("reply message failed,not found [%s] channel [%s] messageID: [%s]\n", msg.Source(), msg.BelongChannel().CID(), msg.MessageID())
			//return
		}
		tuple := MessageTuple{Type: msg.MessageType(), Message: []MessageRecord{{ID: msg.MessageID(), ChannelID: msg.BelongChannel().CID(), Source: msg.Source()}}}
		for _, chat := range room {
			messageID := origin.FindMessageID(chat.Source(), chat.ChannelID())
			if len(messageID) == 0 {
				c.log.Printf("reply [%s] message failed,not found channel [%s] messageID\n", chat.Source(), chat.ChannelID())
				//continue
			}
			c.log.Printf("dispatch message to [%s], from: %s %s %s", chat.ChannelID(), msg.BelongChannel().CName(), msg.BelongUser().UName(), msg.Text())
			id, _ := chat.SendReplyMessage(messageID, msg)
			tuple.Message = append(tuple.Message, MessageRecord{ID: id, ChannelID: chat.ChannelID(), Source: chat.Source()})
		}
		if len(origin.Message) != 0 {
			origin.AddChild(&tuple)
		}
		c.MessageList.Push(tuple)
	case model.MessageTypeActionAdd:
		origin := c.SearchMessage(msg.Source(), msg.BelongChannel().CID(), msg.MessageID())
		if origin == nil {
			c.log.Printf("add reaction failed,not found [%s] channel [%s] messageID: [%s]\n", msg.Source(), msg.BelongChannel().CID(), msg.MessageID())
			return
		}
		for _, chat := range room {
			messageID := origin.FindMessageID(chat.Source(), chat.ChannelID())
			if len(messageID) == 0 {
				c.log.Printf("add reaction [%s] failed,not found channel [%s] messageID\n", chat.Source(), chat.ChannelID())
				continue
			}
			emojiID := emoji.Convert(msg.Source(), chat.Source(), msg.Emoji())
			if len(emojiID) == 0 {
				c.log.Printf("add reaction [%s] failed,not found channel [%s] messageID\n", chat.Source(), chat.ChannelID())
				continue
			}
			if err := chat.SendReaction(messageID, emojiID); err != nil {
				c.log.Printf("failed to send [%s] id:[%s] reaction, %v", chat.ChannelID(), messageID, err)
			}
		}
	case model.MessageTypeActionRemove:
		origin := c.SearchMessage(msg.Source(), msg.BelongChannel().CID(), msg.MessageID())
		if origin == nil {
			c.log.Printf("remove reaction failed,not found [%s] channel [%s] messageID: [%s]\n", msg.Source(), msg.BelongChannel().CID(), msg.MessageID())
			return
		}
		for _, chat := range room {
			messageID := origin.FindMessageID(chat.Source(), chat.ChannelID())
			if len(messageID) == 0 {
				c.log.Printf("remove reaction [%s] failed,not found channel [%s] messageID\n", chat.Source(), chat.ChannelID())
				continue
			}
			emojiID := emoji.Convert(msg.Source(), chat.Source(), msg.Emoji())
			if len(emojiID) == 0 {
				c.log.Printf("remove reaction [%s] failed,not found channel [%s] messageID\n", chat.Source(), chat.ChannelID())
				continue
			}
			if err := chat.RemoveReaction(messageID, emojiID); err != nil {
				c.log.Printf("failed to remove [%s] id:[%s] reaction, %v", chat.ChannelID(), messageID, err)
			}
		}
	case model.MessageTypeActionRemoveALL:
		origin := c.SearchMessage(msg.Source(), msg.BelongChannel().CID(), msg.MessageID())
		if origin == nil {
			c.log.Printf("remove all reaction failed,not found [%s] channel [%s] messageID: [%s]\n", msg.Source(), msg.BelongChannel().CID(), msg.MessageID())
			return
		}
		for _, chat := range room {
			messageID := origin.FindMessageID(chat.Source(), chat.ChannelID())
			if len(messageID) == 0 {
				c.log.Printf("remove all reaction [%s] failed,not found channel [%s] messageID\n", chat.Source(), chat.ChannelID())
				continue
			}
			if err := chat.RemoveReactionAll(messageID); err != nil {
				c.log.Printf("failed to remove [%s] id:[%s] all reaction, %v", chat.ChannelID(), messageID, err)
			}
		}
	}
}

func (c *ChatRoom) SearchMessage(source model.TypeSource, channelID, messageID string) *MessageTuple {
	//return c.MessageQueue.Search(func(v MessageTuple) bool {
	//	for _, record := range v.Message {
	//		if record.Source == source && record.ChannelID == channelID && record.ID == MessageID {
	//			return true
	//		}
	//	}
	//	return false
	//})
	return c.MessageList.SearchFunc(func(v MessageTuple) bool {
		for _, record := range v.Message {
			if record.Source == source && record.ChannelID == channelID && record.ID == messageID {
				return true
			}
		}
		return false
	})
}

func (c *ChatRoom) SearchMessageDelete(source model.TypeSource, channelID, messageID string) *MessageTuple {
	return c.MessageList.DeleteFunc(func(v MessageTuple) bool {
		for _, record := range v.Message {
			if record.Source == source && record.ChannelID == channelID && record.ID == messageID {
				return true
			}
		}
		return false
	})
}
