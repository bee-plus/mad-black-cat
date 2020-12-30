package main

import (
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"gopkg.in/yaml.v2"
)

var (
	// Token is the Discord API token.
	Token string
	// Commands is a map of commands and their outputs.
	Commands map[string]string
	// ApprovedOnly defines if only approved users may use bot commands.
	ApprovedOnly bool
	// IDs is a slice of user IDs approved to use bot commands.
	IDs []string
	// ConfigLoaded defines if the config has been loaded.
	ConfigLoaded bool
)

// Config defines the YAML config data structure.
type Config struct {
	Commands     map[string]string `yaml:"commands"`
	ApprovedOnly bool              `yaml:"approved_only"`
	IDs          []string          `yaml:"ids"`
}

func loadConfig() {
	var config Config

	file, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		if !ConfigLoaded {
			log.Fatal(err)
		} else {
			log.Println(err)
			return
		}
	}

	err = yaml.UnmarshalStrict(file, &config)
	if err != nil {
		if !ConfigLoaded {
			log.Fatal(err)
		} else {
			log.Println(err)
			return
		}
	}

	Commands = config.Commands
	IDs = config.IDs
	ApprovedOnly = config.ApprovedOnly

	ConfigLoaded = true
	log.Println("config loaded successfully")
}

func init() {
	Token = os.Getenv("TOKEN")
	loadConfig()
}

func main() {
	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		log.Println("error creating discord session", err)
		return
	}

	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(messageCreate)

	// In this example, we only care about receiving message events.
	dg.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuildMessages)

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		log.Println("error opening connection", err)
		return
	}

	// Wait here until CTRL-C or other term signal is received.
	log.Println("running; press ctrl-c to exit")
	rc := make(chan os.Signal, 1)
	// Reload config on SIGHUP.
	signal.Notify(rc, syscall.SIGHUP)
	go func() {
		for range rc {
			loadConfig()
		}
	}()
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Cleanly close down the Discord session.
	log.Println("exiting...")
	err = dg.Close()
	if err != nil {
		log.Fatal(err)
	}
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	var approved bool

	// Ignore all messages created by the bot itself.
	if m.Author.ID == s.State.User.ID {
		return
	}

	if ApprovedOnly {
		for _, id := range IDs {
			if id == m.Author.ID {
				approved = true
				break
			}
		}
	} else {
		approved = true
	}

	val, isCmd := Commands[m.Content]

	if isCmd && approved {
		_, err := s.ChannelMessageSend(m.ChannelID, val)
		if err != nil {
			log.Println(err)
		}
	}
}
