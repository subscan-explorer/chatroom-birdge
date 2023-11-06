package model

type DiscordMessageEmoji struct {
	ID   string
	Name string
}

type DiscordMessage struct {
	ID         string
	Type       MessageType
	Channel    IChannelInfo
	User       IUserInfo
	Message    string
	RawMessage string
	SendTime   int64
	//Mentions   []string // 软件内@可转换消息
	EmojiData   *DiscordMessageEmoji
	Attachments []Attachment
	ParentID    string
}

func (d *DiscordMessage) MessageID() string {
	return d.ID
}

func (d *DiscordMessage) ParentMessageID() string {
	return d.ParentID
}

func (d *DiscordMessage) MessageType() MessageType {
	return d.Type
}

func (d *DiscordMessage) Source() TypeSource {
	return DiscordType
}

func (d *DiscordMessage) BelongChannel() IChannelInfo {
	return d.Channel
}

func (d *DiscordMessage) Text() string {
	return d.Message
}

func (d *DiscordMessage) RawText() string {
	return d.RawMessage
}

func (d *DiscordMessage) Attachment() []Attachment {
	return d.Attachments
}

func (d *DiscordMessage) Emoji() string {
	return d.EmojiData.Name
}

func (d *DiscordMessage) BelongUser() IUserInfo {
	return d.User
}
