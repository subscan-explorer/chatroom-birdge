package discord

import (
	"chatroom/model"
	"fmt"
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type Chat struct {
	Channel string
}

func NewDiscordChat(channelID string, receiveCh chan model.IChatMessage) *Chat {
	app.RegisterChannel(channelID, receiveCh)
	c := new(Chat)
	c.Channel = channelID
	c.init()
	return c
}

func (c *Chat) init() {
}

func (c *Chat) ChannelID() string {
	return c.Channel
}

func (c *Chat) Source() model.TypeSource {
	return model.DiscordType
}

func (c *Chat) mentionParsing(text string) string {
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
		args = append(args, id[0], "<@"+user.UID()+"> ")
	}
	return strings.NewReplacer(args...).Replace(text)
}

func (c *Chat) SendMessage(msg model.IChatMessage) (string, error) {
	rsp, err := app.cli.ChannelMessageSend(c.Channel, c.formatText(msg))
	if err != nil {
		return "", err
	}
	return rsp.ID, nil
}

func (c *Chat) SendReplyMessage(parentID string, msg model.IChatMessage) (string, error) {
	var rsp *discordgo.Message
	var err error
	if len(parentID) == 0 {
		rsp, err = app.cli.ChannelMessageSend(c.Channel, fmt.Sprintf("%s\n[Reply Messsage, Parent message not found]", c.formatText(msg)))
	} else {
		rsp, err = app.cli.ChannelMessageSendReply(c.Channel, c.formatText(msg), &discordgo.MessageReference{MessageID: parentID, ChannelID: c.Channel})
	}
	if err != nil {
		return "", err
	}
	return rsp.ID, err
}

func (c *Chat) UpdateMessage(messageID string, msg model.IChatMessage) error {
	if len(messageID) == 0 {
		_, err := app.cli.ChannelMessageSend(c.Channel, fmt.Sprintf("%s\n[Edit Messsage, Original message not found]", c.formatText(msg)))
		return err
	}
	_, err := app.cli.ChannelMessageEdit(c.Channel, messageID, c.formatText(msg))
	return err
}

func (c *Chat) DeleteMessage(messageID string) error {
	return app.cli.ChannelMessageDelete(c.Channel, messageID)
}

func (c *Chat) SendReaction(messageID string, emojiID string) error {
	return app.cli.MessageReactionAdd(c.Channel, messageID, emojiID)
}

func (c *Chat) RemoveReaction(messageID string, emojiID string) error {
	return app.cli.MessageReactionsRemoveEmoji(c.Channel, messageID, emojiID)
}

func (c *Chat) RemoveReactionAll(messageID string) error {
	return app.cli.MessageReactionsRemoveAll(c.Channel, messageID)
}

func (c *Chat) formatText(msg model.IChatMessage) string {
	var text string
	if msg.Source() == c.Source() {
		text = fmt.Sprintf("From: [%s] User: [%s]Send: \n%s", msg.BelongChannel().CName(), msg.BelongUser().UName(), msg.RawText())
	} else {
		text = fmt.Sprintf("From: [%s] User: [%s]Send: \n%s", msg.Source(), msg.BelongUser().UName(), c.mentionParsing(msg.Text()))
	}
	if att := msg.Attachment(); len(att) != 0 {
		text += model.Attachments(att).String()
	}
	return text
}
