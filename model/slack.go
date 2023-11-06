package model

type SlackReaction struct {
	Data string
}

type SlackReplyMessage struct {
	ParentMessageID string
}

type SlackMessage struct {
	ID          string
	Type        MessageType
	Channel     IChannelInfo
	Message     string
	RawMessage  string
	User        IUserInfo
	SendTime    int64
	Reaction    *SlackReaction
	Attachments []Attachment
	ParentID    string
}

func (s *SlackMessage) MessageID() string {
	return s.ID
}

func (s *SlackMessage) ParentMessageID() string {
	return s.ParentID
}

func (s *SlackMessage) MessageType() MessageType {
	return s.Type
}

func (s *SlackMessage) Source() TypeSource {
	return SlackType
}

func (s *SlackMessage) BelongChannel() IChannelInfo {
	return s.Channel
}

func (s *SlackMessage) Text() string {
	return s.Message
}

func (s *SlackMessage) RawText() string {
	return s.RawMessage
}

func (s *SlackMessage) Attachment() []Attachment {
	return s.Attachments
}

func (s *SlackMessage) Emoji() string {
	return s.Reaction.Data
}

func (s *SlackMessage) BelongUser() IUserInfo {
	return s.User
}
