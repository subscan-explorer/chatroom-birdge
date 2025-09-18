package matrix

import (
	"chatroom/model"
	"context"
	"strings"

	"maunium.net/go/mautrix/id"
)

func (a *App) joinRoom(ctx context.Context, roomId ...string) {
	for _, rid := range roomId {
		if v, err := a.cli.JoinRoomByID(ctx, id.RoomID(rid)); err != nil {
			a.log.Printf("join room: %s error: %s\n", rid, err.Error())
		} else {
			a.log.Printf("room info: %s\n", *v)
		}
	}
}

func (a *App) getChannelInfo(roomID string) (result *model.ChannelInfo) {
	defer func() {
		if result == nil {
			result = &model.ChannelInfo{ID: roomID, Name: formatUserDisplay(roomID)}
		}
	}()
	if len(roomID) == 0 {
		return
	}
	a.lock.RLock()
	result = a.ChannelInfo[roomID]
	a.lock.RUnlock()
	if result != nil {
		return
	}
	a.updateChannelMember(context.Background(), roomID)
	a.lock.RLock()
	result = a.ChannelInfo[roomID]
	a.lock.RUnlock()
	return
}

func (a *App) getUserInfo(roomID, userID string) (result *model.User) {
	defer func() {
		if result == nil {
			result = &model.User{ID: userID, Name: formatUserDisplay(userID), DisplayName: formatUserDisplay(userID)}
		}
	}()
	if len(userID) == 0 {
		return
	}
	a.lock.RLock()
	result = a.Users[userID]
	a.lock.RUnlock()
	if result != nil {
		return
	}
	a.updateChannelMember(context.Background(), roomID)
	a.lock.RLock()
	result = a.Users[userID]
	a.lock.RUnlock()
	return
}

func (a *App) updateChannelMember(ctx context.Context, roomId ...string) {
	channelInfo := make(map[string]*model.ChannelInfo)
	userInfo := make(map[string]*model.User)
	for _, rid := range roomId {
		summary, _ := a.cli.GetRoomSummary(ctx, rid)
		info := model.ChannelInfo{
			ID:   rid,
			Name: summary.Name,
		}
		member, _ := a.cli.StateStore.GetAllMembers(ctx, id.RoomID(rid))
		info.Members = make([]string, 0, len(member))

		for uid, u := range member {
			info.Members = append(info.Members, uid.String())
			userInfo[uid.String()] = &model.User{ID: uid.String(), Name: formatUserDisplay(uid.String()), DisplayName: u.Displayname, Avatar: string(u.AvatarURL)}
		}
		channelInfo[rid] = &info
	}
	a.lock.Lock()
	for cid, info := range channelInfo {
		a.ChannelInfo[cid] = info
	}
	for uid, user := range userInfo {
		a.Users[uid] = user
	}
	a.lock.Unlock()
}

func formatUserDisplay(userId string) string {
	userId = strings.TrimPrefix(userId, "@")
	return strings.TrimSuffix(userId, ":matrix.org")
}
