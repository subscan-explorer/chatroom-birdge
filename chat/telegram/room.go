package telegram

import (
	"chatroom/model"
	"fmt"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Chat struct {
	Channel string
}

func NewTelegramChat(channelID string, receiveCh chan model.IChatMessage) *Chat {
	app.RegisterChannel(channelID, receiveCh)
	return &Chat{Channel: channelID}
}

func (c Chat) ChannelID() string {
	return c.Channel
}

func (c Chat) Source() model.TypeSource {
	return model.TelegramType
}

func (c Chat) SendMessage(msg model.IChatMessage) (string, error) {
	rsp, err := app.cli.Send(tgbotapi.NewMessage(stringToInt(c.Channel), c.formatText(msg)))
	if err != nil {
		return "", err
	}
	return strconv.Itoa(rsp.MessageID), nil
}

func (c Chat) SendReplyMessage(parentID string, msg model.IChatMessage) (string, error) {
	var rsp tgbotapi.Message
	var err error
	if len(parentID) == 0 {
		rsp, err = app.cli.Send(tgbotapi.NewMessage(stringToInt(c.Channel), fmt.Sprintf("%s\n[Reply Messsage, Parent message not found]", c.formatText(msg))))
	} else {
		rsp, err = app.cli.Send(tgbotapi.MessageConfig{
			BaseChat: tgbotapi.BaseChat{
				ChatID:           stringToInt(c.Channel),
				ReplyToMessageID: int(stringToInt(parentID)),
			},
			Text:                  c.formatText(msg),
			DisableWebPagePreview: false,
		})
	}
	if err != nil {
		return "", err
	}
	return strconv.Itoa(rsp.MessageID), nil
}

func (c Chat) UpdateMessage(messageID string, msg model.IChatMessage) error {
	if len(messageID) == 0 {
		_, err := app.cli.Send(tgbotapi.NewMessage(stringToInt(c.Channel), fmt.Sprintf("%s\n[Edit Messsage, Original message not found]", c.formatText(msg))))
		return err
	}
	_, err := app.cli.Send(tgbotapi.NewEditMessageText(stringToInt(c.Channel), int(stringToInt(messageID)), c.formatText(msg)))
	return err
}

func (c Chat) DeleteMessage(messageID string) error {
	_, err := app.cli.Send(tgbotapi.NewDeleteMessage(stringToInt(c.Channel), int(stringToInt(messageID))))
	return err
}

func (c Chat) SendReaction(_ string, _ string) error {
	return nil
}

func (c Chat) RemoveReaction(_ string, _ string) error {
	return nil
}

func (c Chat) RemoveReactionAll(_ string) error {
	//return
	return nil
}

func (c Chat) formatText(msg model.IChatMessage) string {
	text := fmt.Sprintf("From: [%s] User:[%s]Send: \n%s", msg.Source(), msg.BelongUser().UName(), msg.Text())
	if att := msg.Attachment(); len(att) != 0 {
		text += model.Attachments(att).String()
	}
	return text
}
