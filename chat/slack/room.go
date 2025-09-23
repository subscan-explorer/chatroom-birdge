package slack

import (
	"chatroom/model"
	"fmt"
	"regexp"
	"strings"

	"github.com/slack-go/slack"
)

type Chat struct {
	Channel string
}

func NewSlackChat(channelID string, receiveCh chan model.IChatMessage) *Chat {
	app.RegisterChannel(channelID, receiveCh)
	return &Chat{Channel: channelID}
}

func (c Chat) ChannelID() string {
	return c.Channel
}

func (c Chat) Source() model.TypeSource {
	return model.SlackType
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
	_, ts, _, err := app.cli.SendMessage(c.ChannelID(), slack.MsgOptionText(c.formatText(msg), false))
	return ts, err
}

func (c Chat) SendReplyMessage(parentID string, msg model.IChatMessage) (string, error) {
	if len(parentID) == 0 {
		_, ts, _, err := app.cli.SendMessage(c.Channel, slack.MsgOptionText(fmt.Sprintf("%s\n[Reply Message, Parent message not found]", c.formatText(msg)), false))
		return ts, err
	}
	_, ts, _, err := app.cli.SendMessage(c.Channel, slack.MsgOptionTS(parentID), slack.MsgOptionText(c.formatText(msg), false))
	return ts, err
}

func (c Chat) UpdateMessage(messageID string, msg model.IChatMessage) error {
	if len(messageID) == 0 {
		_, _, _, err := app.cli.SendMessage(c.Channel, slack.MsgOptionText(fmt.Sprintf("%s\n[Edit Message, Original message not found]", c.formatText(msg)), false))
		return err
	}
	_, _, _, err := app.cli.UpdateMessage(c.Channel, messageID, slack.MsgOptionText(c.formatText(msg), false))
	return err
}

func (c Chat) DeleteMessage(messageID string) error {
	_, _, err := app.cli.DeleteMessage(c.Channel, messageID)
	return err
}

func (c Chat) SendReaction(messageID string, emojiID string) error {
	return app.cli.AddReaction(emojiID, slack.ItemRef{Channel: c.Channel, Timestamp: messageID})
}

func (c Chat) RemoveReaction(messageID string, emojiID string) error {
	return app.cli.RemoveReaction(emojiID, slack.ItemRef{Channel: c.Channel, Timestamp: messageID})
}

func (c Chat) RemoveReactionAll(messageID string) error {
	list, err := app.cli.GetReactions(slack.ItemRef{Channel: c.Channel, Timestamp: messageID}, slack.GetReactionsParameters{Full: true})
	if err != nil {
		return err
	}
	for _, reaction := range list {
		err = app.cli.RemoveReaction(reaction.Name, slack.ItemRef{Channel: c.Channel, Timestamp: messageID})
	}
	return err
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
