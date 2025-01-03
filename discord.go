package main

import (
	"os"

	"github.com/bwmarrin/discordgo"
	"github.com/rs/zerolog/log"
)

var (
	voiceConnection *discordgo.VoiceConnection
	discordSession  *discordgo.Session
)

func discordToOpus() chan *discordgo.Packet {
	opusChannel := make(chan *discordgo.Packet)

	token := os.Getenv("DISCORD_TOKEN")
	guildId := os.Getenv("GUILD_ID")
	channelId := os.Getenv("CHANNEL_ID")

	var err error

	go func() {
		discordSession, voiceConnection, err = startDiscordBot(token, guildId, channelId)

		if err != nil {
			log.Err(err).Msg("failed to start discord bot")
			return
		}

		defer discordSession.Close()
		defer voiceConnection.Close()

		log.Info().Msg("Processing discord packets")

		for packet := range voiceConnection.OpusRecv {
			opusChannel <- packet
		}

		log.Error().Msg("We are past the voiceConnection.opusrecv, this shouldn't happen under normal circumstances")
	}()

	return opusChannel
}

func startDiscordBot(token, guildId, channelId string) (*discordgo.Session, *discordgo.VoiceConnection, error) {
	// Create a new Discord session using the provided bot token.
	discordSession, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Err(err).Msg("error creating Discord session")
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
		log.Err(err).Msg("error opening connection")
		discordSession.Close()
		return nil, nil, err
	}

	// Join the provided voice channel
	voiceConnection, err := discordSession.ChannelVoiceJoin(guildId, channelId, true, false)
	if err != nil {
		log.Err(err).Msg("error joining voice channel")
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
	log.Trace().Msg("Bot is ready!")
}

func voiceStatusUpdate(s *discordgo.Session, event *discordgo.VoiceStateUpdate) {
	log.Trace().
		Str("Username", event.Member.User.Username).
		Bool("Channel empty", event.ChannelID == "").
		Str("ChannelID", event.ChannelID)
}

func voiceDisconnected(s *discordgo.Session, event *discordgo.Disconnect) {
	log.Trace().Msg("voice disconnected")
}

func voiceConnected(s *discordgo.Session, event *discordgo.Connect) {
	log.Trace().Msg("voice connected")
}
