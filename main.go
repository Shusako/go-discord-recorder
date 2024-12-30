package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	discordFrameSize    = 960
	discordSampleRate   = 48000
	discordChannelCount = 1
)

var (
	globalStartTime time.Time
	modelPath       = os.Getenv("MODEL_PATH")
)

func main() {
	zerolog.TimeFieldFormat = time.RFC3339
	zerolog.TimestampFunc = func() time.Time {
		return time.Now().Local()
	}

	log.Info().Msg("App startup")

	// TODO: 2024/09/08: Considering ripping whipser out and use this instead:
	//		 https://github.com/mutablelogic/go-whisper
	// TODO: 2023/12/24 17:37:46 [DG0] wsapi.go:796:onVoiceServerUpdate() onVoiceServerUpdate voice.open, did not receive voice Session ID in time
	// TODO: disconnect protection
	// 		when there's a disconnect, the opusrecv doesnt close, but it stops receiving packets
	// TODO: time not speaking accounted for in timestamps
	// TODO: global timestamp
	// TODO: Check this out to see if we can optimize ffmpeg usage: https://github.com/ducc/GoMusicBot/blob/master/src/framework/audio.go#L60
	// TODO: fan-out option to save to file as well
	// TODO: buffer all transcripts and send to LLM for summarizing

	// channel to handle voiceConnection.OpusRecv and it potentially breaking
	opusChannel := discordToOpus()
	pcmDataChannel := opusDecode(opusChannel)
	bufferedAudioChannel := bufferVoice(pcmDataChannel)
	downsampledAudioChannel := downsampleVoice(bufferedAudioChannel)
	transcriptChannel := transcribeAudio(downsampledAudioChannel)
	manageTranscripts(transcriptChannel)

	globalStartTime = time.Now()

	// wait for sigint or sigterm to exit
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	log.Trace().Msg("done")
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
						log.Trace().
							Str("Username", GetUserCache().GetUsernameOrDefault(buffer.identifier, "unknown")).
							Str("buffer.identifier", buffer.identifier).
							Msg("flushing buffer")
						output <- *buffer
						buffer.Clear()
					}
				}
			}
		}
	}()

	return output
}
