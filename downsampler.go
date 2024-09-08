package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
)

// preprocessAudio handles converting from discords
func preprocessAudio(pcmData []int16) ([]float32, error) {
	// Convert []int16 to []byte for FFmpeg input
	var pcmBytes bytes.Buffer
	for _, sample := range pcmData {
		if err := binary.Write(&pcmBytes, binary.LittleEndian, sample); err != nil {
			return nil, err
		}
	}

	// Set up the FFmpeg command
	cmd := exec.Command("ffmpeg", "-f", "s16le", "-ar", "48000", "-ac", "1", "-i", "pipe:0", "-ar", "16000", "-f", "f32le", "pipe:1")
	// cmd := exec.Command("ffmpeg", "-f", "s16le", "-ar", "48000", "-ac", "1", "-i", "pipe:0", "-ar", "16000", "-ac", "1", "-y", "output.flac")

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdin = bytes.NewReader(pcmBytes.Bytes()) // PCM data is passed to FFmpeg via stdin
	cmd.Stdout = &out                             // FFmpeg's output will be captured here
	cmd.Stderr = &stderr                          // Any error messages will be captured here

	// Run the FFmpeg command
	if err := cmd.Run(); err != nil {
		os.Stderr.WriteString(stderr.String())
		return nil, err
	}

	// Convert the output bytes back to []float32
	float32PCM := make([]float32, out.Len()/4) // 4 bytes per float32
	if err := binary.Read(bytes.NewReader(out.Bytes()), binary.LittleEndian, &float32PCM); err != nil {
		return nil, err
	}

	return float32PCM, nil
}

func downsampleVoice(input <-chan VoiceBuffer[int16]) chan VoiceBuffer[float32] {
	output := make(chan VoiceBuffer[float32])

	go func() {
		defer close(output)

		for buffer := range input {
			// 48kHz (discord) -> 16kHz (whisper)
			floatData, err := preprocessAudio(buffer.pcm)
			if err != nil {
				fmt.Println("error preprocessing audio,", err)
				return
			}

			fmt.Printf("Sending length of data to the transcriber: %d \n", len(floatData))
			output <- NewVoiceBuffer[float32](floatData, buffer.identifier)
		}
	}()

	return output
}
