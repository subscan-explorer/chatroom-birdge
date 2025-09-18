package main

import (
	"chatroom/chat/discord"
	"chatroom/chat/matrix"
	"chatroom/chat/slack"
	"chatroom/chat/telegram"
	"chatroom/conf"
	"chatroom/emoji"
	"chatroom/room"
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	// init config
	conf.InitConf(ctx)
	emoji.InitEmojiConvert()

	slack.NewClient(ctx, conf.Conf.Slack)
	discord.NewClient(ctx, conf.Conf.Discord)
	telegram.NewClient(ctx, conf.Conf.Telegram)
	matrix.NewClient(ctx, conf.Conf.Matrix)
	//handler := server.Route()
	go listenExit(cancel)
	room.NewMainRoom(ctx)
}

func listenExit(cancel context.CancelFunc) {
	sign := make(chan os.Signal, 1)
	signal.Notify(sign, os.Kill, os.Interrupt, syscall.SIGTERM)
	s := <-sign
	log.Printf("receive signal %s, exit...\n", s.String())
	cancel()
}
