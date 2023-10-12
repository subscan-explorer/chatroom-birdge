package model

import "strconv"

//type SlackReaction struct {
//	Data string
//}

type TelegramMessage struct {
	ID          int
	Type        MessageType
	Channel     IChannelInfo
	Message     string
	RawMessage  string
	User        IUserInfo
	SendTime    int64
	Attachments []Attachment
	//Reaction *SlackReaction
	ParentID int
}

func (t *TelegramMessage) MessageID() string {
	return strconv.Itoa(t.ID)
}

func (t *TelegramMessage) ParentMessageID() string {
	return strconv.Itoa(t.ParentID)
}

func (t *TelegramMessage) MessageType() MessageType {
	return t.Type
}

func (t *TelegramMessage) Source() TypeSource {
	return TelegramType
}

func (t *TelegramMessage) BelongChannel() IChannelInfo {
	return t.Channel
}

func (t *TelegramMessage) Text() string {
	return t.Message
}

func (t *TelegramMessage) RawText() string {
	return t.RawMessage
}

func (t *TelegramMessage) Attachment() []Attachment {
	return t.Attachments
}

func (t *TelegramMessage) Emoji() string {
	//return t.Reaction.Data
	return ""
}

func (t *TelegramMessage) BelongUser() IUserInfo {
	return t.User
}
