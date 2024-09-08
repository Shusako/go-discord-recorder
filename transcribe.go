package main

import (
	"fmt"
	"io"

	"github.com/ggerganov/whisper.cpp/bindings/go/pkg/whisper"
	"github.com/rs/zerolog/log"
)

func transcribeAudio(input <-chan VoiceBuffer[float32]) chan Transcript {
	model, err := whisper.New(modelPath)
	if err != nil {
		log.Err(err).Msg("error loading model")
		return nil
	}

	output := make(chan Transcript)

	go func() {
		defer model.Close()
		defer close(output)
		for {
			buffer := <-input

			context, err := model.NewContext()
			if err != nil {
				log.Err(err).Msg("error creating context")
				continue // or handle some other way?
			}

			if err := context.Process(buffer.pcm, nil, nil); err != nil {
				log.Err(err).Msg("error transcribing voicebuffer")
				continue // or handle some other way?
			}

			text := ""
			for {
				segment, err := context.NextSegment()
				if err == io.EOF {
					break
				}
				if err != nil {
					log.Err(err).Msg("error fetching next segment")
					break
				}

				name := GetUserCache().GetUsernameOrDefault(buffer.identifier, "unknown")

				startTime := buffer.firstUpdated.Sub(Context.globalStartTime).Seconds()
				text += fmt.Sprintf("[%6.2fs->%6.2fs] [%s] %s\n", startTime+segment.Start.Seconds(), startTime+segment.End.Seconds(), name, segment.Text)
			}

			log.Info().Msg(text)
			// TODO: differentiate between temporary and final transcripts
			output <- Transcript{buffer.identifier, "", text, ""}
		}
	}()

	return output
}
