package model

import (
	"fmt"
	"strings"
)

type TypeSource int
type MessageType int

const (
	SlackType TypeSource = iota
	DiscordType
	TelegramType
	MatrixType
)

const (
	MessageTypeTextCreate MessageType = iota
	MessageTypeTextUpdate
	MessageTypeTextDelete
	MessageTypeTextReply
	MessageTypeActionAdd
	MessageTypeActionRemove
	MessageTypeActionRemoveALL
)

func (t TypeSource) String() string {
	switch t {
	case SlackType:
		return "Slack"
	case DiscordType:
		return "Discord"
	case TelegramType:
		return "Telegram"
	case MatrixType:
		return "Matrix"
	}
	return "Unknown"
}

type User struct {
	ID          string
	Name        string
	DisplayName string
	Avatar      string
	BotID       string
}

func NewUserInfo(id string) *User {
	return &User{
		ID:          id,
		Name:        "Unknown",
		DisplayName: "Unknown",
	}
}

type ChannelInfo struct {
	ID      string
	Name    string
	Members []string
}

func NewChannelInfo(id string) *ChannelInfo {
	return &ChannelInfo{
		ID:   id,
		Name: "Unknown",
	}
}

func (u *User) UID() string {
	return u.ID
}

func (u *User) UName() string {
	return u.DisplayName
}

func (c ChannelInfo) CID() string {
	return c.ID
}

func (c ChannelInfo) CName() string {
	return c.Name
}

type ChatMessage struct {
	Type      TypeSource
	ChannelID string
	Text      string
}

type IChatMessage interface {
	MessageID() string
	MessageType() MessageType
	Source() TypeSource
	BelongUser() IUserInfo
	BelongChannel() IChannelInfo
	Text() string
	RawText() string
	Attachment() []Attachment
	ParentMessageID() string
	Emoji() string
}

type IUserInfo interface {
	UID() string
	UName() string
}

type IChannelInfo interface {
	CID() string
	CName() string
}

type Attachments []Attachment
type Attachment struct {
	Name string
	Type string
	URL  string
}

func (a Attachments) String() string {
	str := strings.Builder{}
	str.WriteString("\nAttachment:")
	for _, att := range a {
		str.WriteByte('\n')
		if len(att.Name) != 0 {
			str.WriteString(fmt.Sprintf("Name: [%s] ", att.Name))
		}
		if len(att.URL) != 0 {
			str.WriteString(att.URL)
			str.WriteString(" ")
		}
		if len(att.Type) != 0 {
			str.WriteString(fmt.Sprintf("Type: [%s]", att.Type))
		}

	}
	return str.String()
}
