package main

import (
	"fmt"
	"os"

	"github.com/bwmarrin/discordgo"
)

func startDiscordBot(token, guildId, channelId string) (*discordgo.Session, *discordgo.VoiceConnection, error) {
	// Create a new Discord session using the provided bot token.
	discordSession, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return nil, nil, err
	}

	discordSession.StateEnabled = true
	discordSession.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuildVoiceStates)

	// Register handlers
	discordSession.AddHandler(ready)
	discordSession.AddHandler(voiceStatusUpdate)

	// Open a websocket connection to Discord and begin listening.
	err = discordSession.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		discordSession.Close()
		return nil, nil, err
	}

	// Join the provided voice channel
	voiceConnection, err := discordSession.ChannelVoiceJoin(guildId, channelId, true, false)
	if err != nil {
		fmt.Println("error joining voice channel,", err)
		discordSession.Close()
		voiceConnection.Close()
		return nil, nil, err
	}

	registerCacheHandlers(discordSession, voiceConnection)

	// discordSession.AddHandler(voiceDisconnected)
	// discordSession.AddHandler(voiceConnected)

	discordSession.UpdateListeningStatus(os.Getenv("LISTENING_TO"))

	return discordSession, voiceConnection, nil
}

func ready(s *discordgo.Session, event *discordgo.Ready) {
	fmt.Println("Bot is ready!")
}

func voiceStatusUpdate(s *discordgo.Session, event *discordgo.VoiceStateUpdate) {
	fmt.Println(event.Member.User.Username, event.ChannelID == "", event.ChannelID)
}

func voiceDisconnected(s *discordgo.Session, event *discordgo.Disconnect) {
	fmt.Println("voice disconnected")
}

func voiceConnected(s *discordgo.Session, event *discordgo.Connect) {
	fmt.Println("voice connected")
}
