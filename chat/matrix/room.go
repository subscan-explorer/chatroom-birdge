package matrix

import (
	"chatroom/model"
	"context"
	"fmt"
	"regexp"
	"strings"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

type Chat struct {
	RoomId string
}

func NewMatrixChat(roomId string, receiveCh chan model.IChatMessage) *Chat {
	app.RegisterChannel(roomId, receiveCh)
	return &Chat{RoomId: roomId}
}

func (c Chat) ChannelID() string {
	return c.RoomId
}

func (c Chat) Source() model.TypeSource {
	return model.MatrixType
}

func (c Chat) mentionParsing(text string) string {
	rgx := regexp.MustCompile(`@([^@\\s]*)\\s`)
	userID := rgx.FindAllStringSubmatch(text, -1)
	var args []string
	for _, id := range userID {
		if len(id) != 2 {
			continue
		}
		user := app.SearchUserName(id[1])
		if user == nil {
			continue
		}
		args = append(args, id[0], "<@"+user.UID()+">")
	}
	return strings.NewReplacer(args...).Replace(text)
}

func (c Chat) SendMessage(msg model.IChatMessage) (string, error) {
	rsp, err := app.cli.SendText(context.Background(), id.RoomID(c.RoomId), c.formatText(msg))
	if err != nil {
		return "", err
	}
	if rsp != nil {
		return rsp.EventID.String(), nil
	}
	return "", nil
}

func (c Chat) SendReplyMessage(parentID string, msg model.IChatMessage) (string, error) {
	var rsp *mautrix.RespSendEvent
	var err error
	if len(parentID) == 0 {
		rsp, err = app.cli.SendText(context.Background(), id.RoomID(c.RoomId), fmt.Sprintf("%s\n[Reply Message, Parent message not found]", c.formatText(msg)))
	} else {
		rsp, err = app.cli.SendMessageEvent(context.Background(), id.RoomID(c.RoomId), event.EventMessage, &event.MessageEventContent{
			MsgType: event.MsgText,
			Body:    c.formatText(msg),
			RelatesTo: &event.RelatesTo{
				Type:    event.RelThread,
				EventID: id.EventID(parentID),
			},
		})
	}
	if err != nil {
		return "", err
	}
	if rsp != nil {
		return rsp.EventID.String(), nil
	}
	return "", nil
}

func (c Chat) UpdateMessage(messageID string, msg model.IChatMessage) error {
	var rsp *mautrix.RespSendEvent
	var err error
	if len(messageID) == 0 {
		rsp, err = app.cli.SendText(context.Background(), id.RoomID(c.RoomId), fmt.Sprintf("%s\n[Edit Message, Original message not found]", c.formatText(msg)))
	} else {
		rsp, err = app.cli.SendMessageEvent(context.Background(), id.RoomID(c.RoomId), event.EventMessage, &event.MessageEventContent{
			MsgType: event.MsgText,
			Body:    "* " + c.formatText(msg),
			NewContent: &event.MessageEventContent{
				MsgType: event.MsgText,
				Body:    c.formatText(msg),
			},
			RelatesTo: &event.RelatesTo{
				Type:    event.RelReplace,
				EventID: id.EventID(messageID),
			},
		})
	}
	if err != nil {
		return err
	}
	if rsp != nil {
		return nil
	}
	return nil
}

func (c Chat) DeleteMessage(messageID string) error {
	_, err := app.cli.RedactEvent(context.Background(), id.RoomID(c.RoomId), id.EventID(messageID), mautrix.ReqRedact{Reason: "Source message to delete"})
	return err
}

func (c Chat) SendReaction(messageID string, emojiID string) error {
	_, err := app.cli.SendReaction(context.Background(), id.RoomID(c.RoomId), id.EventID(messageID), emojiID)
	return err
}

func (c Chat) RemoveReaction(messageID string, emojiID string) error {
	return nil
}

func (c Chat) RemoveReactionAll(messageID string) error {
	return nil
}

func (c Chat) formatText(msg model.IChatMessage) string {
	var text string
	if msg.Source() == c.Source() {
		text = fmt.Sprintf("From: [%s] User: [%s] Send: \n%s", msg.BelongChannel().CName(), msg.BelongUser().UName(), msg.RawText())
	} else {
		text = fmt.Sprintf("From: [%s] User: [%s] Send: \n%s", msg.Source(), msg.BelongUser().UName(), c.mentionParsing(msg.Text()))
	}
	if att := msg.Attachment(); len(att) != 0 {
		text += model.Attachments(att).String()
	}
	return text
}
