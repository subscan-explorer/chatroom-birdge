package model

type MatrixMessage struct {
	ID          string
	Type        MessageType
	Channel     IChannelInfo
	Message     string
	RawMessage  string
	User        IUserInfo
	SendTime    int64
	Reaction    string
	Attachments []Attachment
	ParentID    string
}

func (s *MatrixMessage) MessageID() string {
	return s.ID
}

func (s *MatrixMessage) ParentMessageID() string {
	return s.ParentID
}

func (s *MatrixMessage) MessageType() MessageType {
	return s.Type
}

func (s *MatrixMessage) Source() TypeSource {
	return MatrixType
}

func (s *MatrixMessage) BelongChannel() IChannelInfo {
	return s.Channel
}

func (s *MatrixMessage) Text() string {
	return s.Message
}

func (s *MatrixMessage) RawText() string {
	return s.Message
}

func (s *MatrixMessage) Attachment() []Attachment {
	return s.Attachments
}

func (s *MatrixMessage) Emoji() string {
	return s.Reaction
}

func (s *MatrixMessage) BelongUser() IUserInfo {
	return s.User
}
