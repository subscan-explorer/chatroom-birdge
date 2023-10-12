package room

import (
	"chatroom/model"
)

type MessageRecord struct {
	ID        string
	ChannelID string
	Source    model.TypeSource
}

type MessageTuple struct {
	Message []MessageRecord
	Type    model.MessageType
	Child   *MessageTuple
	Parent  *MessageTuple
}

func (m *MessageTuple) FindMessageID(source model.TypeSource, channelID string) string {
	cur := m
	if source == model.SlackType { // 找slack 顶级parent
		for cur.Parent != nil {
			cur = cur.Parent
		}
	}
	for _, record := range m.Message {
		if record.Source == source && record.ChannelID == channelID {
			return record.ID
		}
	}
	return ""
}

func (m *MessageTuple) Delete() {
	if m.Parent != nil {
		m.Parent.Child = m.Child
	}
	if m.Child != nil {
		m.Child.Parent = m.Parent
	}
	m.Parent = nil
	m.Child = nil
}

func (m *MessageTuple) AddChild(child *MessageTuple) {
	child.Parent = m
	m.Child = child
}
