package discord

import (
	"chatroom/model"
	"chatroom/utils"
	"encoding/json"
)

func (a *App) getUserInfo(userID ...string) (data []model.User) {
	for _, id := range userID {
		if user, _ := a.cli.User(id); user != nil {
			data = append(data, model.User{
				ID:          user.ID,
				Name:        user.Username,
				DisplayName: user.Username,
				Avatar:      user.Avatar,
				BotID:       utils.IfElse(user.Bot, user.ID, ""),
			})
		}
	}
	return
}

func (a *App) getChannelInfo(channelID ...string) (data []model.ChannelInfo) {
	for _, id := range channelID {
		if channel, _ := a.cli.Channel(id); channel != nil {
			a.log.Println(channel.GuildID)
			m, _ := a.cli.UserGuildMember(channel.GuildID)
			d, _ := json.Marshal(m)
			a.log.Println("member ", string(d))
			data = append(data, model.ChannelInfo{
				ID:   channel.ID,
				Name: channel.Name,
			})
		}
	}
	return
}

func (a *App) HistoryMessage(channelID string, size int) []model.DiscordMessage {
	var beforeID string
	var messages []model.DiscordMessage
	for size > 0 {
		message, err := app.cli.ChannelMessages(channelID, 100, beforeID, "", "")
		d, _ := json.Marshal(message)
		a.log.Println(string(d))
		if err != nil {
			a.log.Println(err.Error())
			continue
		}
		for _, m := range message {
			v := model.DiscordMessage{
				ID: m.ID,
				Channel: utils.Default(a.GetChannelInfo(m.ChannelID), func(v *model.ChannelInfo) bool {
					return v != nil
				}, model.NewChannelInfo(m.ChannelID)),
			}
			messages = append(messages, v)
		}
		break
	}
	return messages
}
