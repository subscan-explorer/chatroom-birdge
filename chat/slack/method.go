package slack

import (
	"chatroom/model"

	"github.com/slack-go/slack"
)

func (c *App) GetAllChannels() ([]slack.Channel, error) {
	param := new(slack.GetConversationsParameters)
	param.ExcludeArchived = true
	param.Limit = 100
	param.TeamID = c.TeamID
	param.Types = []string{"public_channel", "private_channel"}
	var result []slack.Channel
	for {
		channels, cursor, err := c.cli.GetConversations(param)
		if err != nil {
			return result, err
		}
		result = append(result, channels...)
		if len(cursor) == 0 {
			break
		}
		param.Cursor = cursor
	}
	return result, nil
}

func (c *App) getChannelInfo(channelID string) (*model.ChannelInfo, error) {
	param := new(slack.GetConversationInfoInput)
	param.ChannelID = channelID
	param.IncludeNumMembers = true
	channel, err := c.cli.GetConversationInfo(param)
	if err != nil {
		return nil, err
	}
	info := new(model.ChannelInfo)
	info.Name = channel.Name
	info.ID = channel.ID
	if channel.NumMembers > 0 {
		if info.Members, err = c.GetUsersByChannel(channel.ID); err != nil {
			return nil, err
		}
	}
	return info, nil
}

func (c *App) GetUsersByChannel(channelID string) ([]string, error) {
	param := new(slack.GetUsersInConversationParameters)
	param.ChannelID = channelID
	param.Limit = 100
	var users []string
	for {
		data, cursor, err := c.cli.GetUsersInConversation(param)
		if err != nil {
			return users, err
		}
		users = append(users, data...)
		if len(cursor) == 0 {
			break
		}
		param.Cursor = cursor
	}
	return users, nil
}

func (c *App) getUserInfo(userID ...string) (data []model.User, err error) {
	var users *[]slack.User
	if users, err = c.cli.GetUsersInfo(userID...); err != nil {
		return
	}
	for _, user := range *users {
		data = append(data, model.User{
			ID:          user.ID,
			Name:        user.RealName,
			DisplayName: user.Profile.DisplayName,
			Avatar:      user.Profile.Image512,
			BotID:       user.Profile.BotID,
		})
	}
	return
}
