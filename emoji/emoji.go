package emoji

import (
	"chatroom/conf"
	"chatroom/model"
	"log"
	"strings"
)

const EmojiType = 2

var emojis [][EmojiType]string

func InitEmojiConvert() {
	for _, e := range conf.Conf.Emoji {
		emo := strings.Split(e, ",")
		if len(emo) != EmojiType {
			continue
		}
		emojis = append(emojis, [EmojiType]string{emo[0], emo[1]})
	}
	log.Printf("init emoji: %+v\n", emojis)
}

func SlackConvertEmoji(emoji string) string {
	for _, e := range emojis {
		if e[0] == emoji {
			return e[1]
		}
	}
	return ""
}

func Convert(source, target model.TypeSource, emoji string) string {
	if source == target || (source > model.SlackType && target > model.SlackType) {
		return emoji
	}

	sourceIdx, targetIdx := 0, 0
	if source == model.TelegramType || source == model.DiscordType || source == model.MatrixType {
		sourceIdx = 1
	}
	if target == model.TelegramType || target == model.DiscordType || target == model.MatrixType {
		targetIdx = 1
	}
	for _, e := range emojis {
		if e[sourceIdx] == emoji {
			return e[targetIdx]
		}
	}
	return emoji
}
