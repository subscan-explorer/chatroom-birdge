package telegram

import (
	"chatroom/model"
	"chatroom/utils"
	"encoding/json"
	"sort"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (a *App) getUserInfo(chatID string, userID ...string) (data []model.User) {
	param := tgbotapi.GetChatMemberConfig{}
	param.ChatID = a.ConvictionChannel(chatID)
	for _, id := range userID {
		param.UserID = a.ConvictionChannel(id)
		if member, err := a.cli.GetChatMember(param); err == nil && member.User != nil {
			d, _ := json.Marshal(member)
			a.log.Printf("userid: %s info: %s", id, string(d))
			data = append(data, model.User{
				ID:          strconv.FormatInt(member.User.ID, 10),
				Name:        member.User.String(),
				DisplayName: member.User.String(),
				BotID:       utils.IfElse(member.User.IsBot, strconv.FormatInt(member.User.ID, 10), ""),
			})
		}
	}
	return
}

func (a *App) ConvictionChannel(channelID string) int64 {
	id, _ := strconv.ParseInt(channelID, 10, 64)
	return id
}

func (a *App) getChannelInfo(channelID ...string) (data []model.ChannelInfo) {
	param := tgbotapi.ChatInfoConfig{}
	for _, id := range channelID {
		param.ChatID = a.ConvictionChannel(id)
		if chat, err := a.cli.GetChat(param); err == nil {
			data = append(data, model.ChannelInfo{
				ID:   strconv.FormatInt(chat.ID, 10),
				Name: chat.Title,
			})
		}
	}
	return
}

func (a *App) Attachment(msg *tgbotapi.Message) []model.Attachment {
	var result []model.Attachment
	fieldDict := make(map[string]struct{})
	sort.Slice(msg.Photo, func(i, j int) bool {
		return msg.Photo[i].Height*msg.Photo[i].Width > msg.Photo[j].Height*msg.Photo[j].Width
	})
	for _, info := range msg.Photo {
		if _, exist := fieldDict[info.FileID]; exist {
			continue
		}
		fieldDict[info.FileID] = struct{}{}
		result = append(result, model.Attachment{
			Name: info.FileUniqueID,
			Type: "Photo",
		})
	}
	if msg.Document != nil {
		result = append(result, model.Attachment{
			Name: msg.Document.FileName,
			Type: utils.IfElse(len(msg.Document.MimeType) == 0, "Doc", msg.Document.MimeType),
		})
	}
	return result
}
