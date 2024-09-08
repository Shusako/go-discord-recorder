package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"layeh.com/gopus"
)

const (
	discordFrameSize    = 960
	discordSampleRate   = 48000
	discordChannelCount = 1
	modelPath           = "resources/ggml-small.en.bin"
)

var (
	Context *AppContext
)

type AppContext struct {
	voiceConnection *discordgo.VoiceConnection
	discordSession  *discordgo.Session
	globalStartTime time.Time
}

func main() {
	zerolog.TimeFieldFormat = time.RFC3339
	zerolog.TimestampFunc = func() time.Time {
		return time.Now().Local()
	}

	log.Info().Msg("This is a test of zero log")

	token := os.Getenv("DISCORD_TOKEN")
	guildId := os.Getenv("GUILD_ID")
	channelId := os.Getenv("CHANNEL_ID")

	Context = &AppContext{}

	var err error

	Context.discordSession, Context.voiceConnection, err = startDiscordBot(token, guildId, channelId)
	if err != nil {
		fmt.Println("failed to start discord bot")
		return
	}
	defer Context.discordSession.Close()
	defer Context.voiceConnection.Close()

	// TODO: 2023/12/24 17:37:46 [DG0] wsapi.go:796:onVoiceServerUpdate() onVoiceServerUpdate voice.open, did not receive voice Session ID in time
	// TODO: disconnect protection
	// 		when there's a disconnect, the opusrecv doesnt close, but it stops receiving packets
	// TODO: time not speaking accounted for in timestamps
	// TODO: global timestamp
	// TODO: Check this out to see if we can optimize ffmpeg usage: https://github.com/ducc/GoMusicBot/blob/master/src/framework/audio.go#L60
	// TODO: fan-out option to save to file as well
	// TODO: buffer all transcripts and send to LLM for summarizing

	// channel to handle voiceConnection.OpusRecv and it potentially breaking
	pcmDataChannel := opusDecode(Context.voiceConnection.OpusRecv)
	bufferedAudioChannel := bufferVoice(pcmDataChannel)
	downsampledAudioChannel := downsampleVoice(bufferedAudioChannel)
	transcriptChannel := transcribeAudio(downsampledAudioChannel)
	manageTranscripts(transcriptChannel)

	Context.globalStartTime = time.Now()

	// wait for sigint or sigterm to exit
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	fmt.Println("done")
}

func opusDecode(opusRecv <-chan *discordgo.Packet) chan VoiceBuffer[int16] {
	decoderMap := make(map[string]*gopus.Decoder)
	pcmData := make(chan VoiceBuffer[int16])

	go func() {
		defer close(pcmData)

		for p := range opusRecv {
			user, userExists := GetSSRCMap().GetUser(p.SSRC)
			if !userExists {
				fmt.Println("user not found for ssrc:", p.SSRC, "opus packet going to be dropped")
				continue
			}

			if user.Bot {
				// TODO: fix
				// this is just a quick thing for now, want to ignore bots to prevent music bots from filling up memory
				continue
			}

			decoder, ok := decoderMap[user.ID]
			if !ok {
				var err error
				decoder, err = gopus.NewDecoder(discordSampleRate, discordChannelCount)
				if err != nil {
					fmt.Println(fmt.Sprintf("failed to create decoder for %d", p.SSRC), err)
					return
				}
				decoderMap[user.ID] = decoder
			}

			pcm, err := decoder.Decode(p.Opus, discordFrameSize, false)
			if err != nil {
				fmt.Println("failed to decode", err)
				return
			}

			pcmData <- NewVoiceBuffer(pcm, user.ID)
		}
	}()

	return pcmData
}

func bufferVoice(input <-chan VoiceBuffer[int16]) chan VoiceBuffer[int16] {
	bufferMap := make(map[string]*VoiceBuffer[int16])
	deadTime := 1 * time.Second
	ticker := time.NewTicker(deadTime)

	output := make(chan VoiceBuffer[int16])

	go func() {
		defer close(output)
		for {
			select {
			case buffer := <-input:
				if _, ok := bufferMap[buffer.identifier]; !ok {
					bufferMap[buffer.identifier] = &buffer
				} else {
					bufferMap[buffer.identifier].Buffer(&buffer)
				}
			case <-ticker.C:
				for _, buffer := range bufferMap {
					if len(buffer.pcm) == 0 {
						continue
					}

					if time.Since(buffer.lastUpdated) > deadTime {
						fmt.Println("flushing buffer for", GetUserCache().GetUsernameOrDefault(buffer.identifier, "unknown"), "(", buffer.identifier, ")")
						output <- *buffer
						buffer.Clear()
					}
				}
			}
		}
	}()

	return output
}
