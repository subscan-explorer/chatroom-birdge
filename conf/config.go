package conf

import (
	"context"
	"flag"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

var Conf Config

type Config struct {
	Room     []Room   `yaml:"room"`
	Slack    Slack    `yaml:"slack"`
	Discord  Discord  `yaml:"discord"`
	Matrix   Matrix   `yaml:"matrix"`
	Telegram Telegram `yaml:"telegram"`
	Emoji    []string `yaml:"emoji"`

	slackChat    []string `yaml:"-"`
	discordChat  []string `yaml:"-"`
	telegramChat []string `yaml:"-"`
}

type Room struct {
	Name string     `yaml:"name"`
	Chat []RoomChat `yaml:"chat"`
}

type RoomChat struct {
	Type   string   `yaml:"type"`
	ChatID []string `yaml:"chatID"`
}

type Matrix struct {
	Name     string `yaml:"name"`
	Host     string `yaml:"host"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

type Discord struct {
	Token string `yaml:"token"`
}

type Telegram struct {
	Token string `yaml:"token"`
}

type Slack struct {
	Token         string `yaml:"token"`
	AppLevelToken string `yaml:"appLevelToken"`
}

var path *string

func init() {
	path = flag.String("conf", "conf/config.yml", "configuration file path")
}

func InitConf(_ context.Context) {
	if !flag.Parsed() {
		flag.Parse()
	}
	data, err := os.ReadFile(*path)
	if err != nil {
		log.Fatalln("failed to open configuration file. ", err.Error())
	}
	if err = yaml.Unmarshal(data, &Conf); err != nil {
		log.Fatalf("failed to parse configuration file. err: %s\n", err.Error())
	}
	// check channel
	discordConf, slackConf, telegramConf := false, false, false
	for _, c := range Conf.Room {
		for _, chat := range c.Chat {
			switch chat.Type {
			case "slack":
				slackConf = true
				Conf.slackChat = append(Conf.slackChat, chat.ChatID...)
			case "discord":
				discordConf = true
				Conf.discordChat = append(Conf.discordChat, chat.ChatID...)
			case "telegram":
				telegramConf = true
				Conf.telegramChat = append(Conf.telegramChat, chat.ChatID...)
			}
		}
	}
	if slackConf {
		if len(Conf.Slack.Token) == 0 || len(Conf.Slack.AppLevelToken) == 0 {
			log.Fatalln("needs to configure slack token")
		}
	}
	if discordConf {
		if len(Conf.Discord.Token) == 0 {
			log.Fatalln("needs to configure discord token")
		}
	}
	if telegramConf {
		if len(Conf.Telegram.Token) == 0 {
			log.Fatalln("needs to configure telegram token")
		}
	}
	log.Printf("config: %+v\n", Conf)
}

func (c Config) GetSlackChat() []string {
	return c.slackChat
}
func (c Config) GetDiscordChat() []string {
	return c.discordChat
}
func (c Config) GetTelegramChat() []string {
	return c.telegramChat
}
