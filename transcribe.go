package main

import (
	"fmt"
	"io"

	"github.com/ggerganov/whisper.cpp/bindings/go/pkg/whisper"
)

func transcribeAudio(input <-chan VoiceBuffer[float32]) chan Transcript {
	model, err := whisper.New(modelPath)
	if err != nil {
		fmt.Println("error loading model,", err)
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
				fmt.Println("error creating context,", err)
				continue // or handle some other way?
			}

			if err := context.Process(buffer.pcm, nil, nil); err != nil {
				fmt.Println("error transcribing voicebuffer,", err)
				continue // or handle some other way?
			}

			text := ""
			for {
				segment, err := context.NextSegment()
				if err == io.EOF {
					break
				}
				if err != nil {
					fmt.Printf("error fetching next segment: %v\n", err)
					break
				}

				name := GetUserCache().GetUsernameOrDefault(buffer.identifier, "unknown")

				startTime := buffer.firstUpdated.Sub(Context.globalStartTime).Seconds()
				text += fmt.Sprintf("[%6.2fs->%6.2fs] [%s] %s\n", startTime+segment.Start.Seconds(), startTime+segment.End.Seconds(), name, segment.Text)
			}

			fmt.Println(text)
			// TODO: differentiate between temporary and final transcripts
			output <- Transcript{buffer.identifier, "", text, ""}
		}
	}()

	return output
}
