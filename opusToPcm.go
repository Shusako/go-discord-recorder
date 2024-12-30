package main

import (
	"github.com/bwmarrin/discordgo"
	"github.com/rs/zerolog/log"
	"layeh.com/gopus"
)

func opusDecode(opusRecv <-chan *discordgo.Packet) chan VoiceBuffer[int16] {
	decoderMap := make(map[string]*gopus.Decoder)
	pcmChannel := make(chan VoiceBuffer[int16])

	go func() {
		defer close(pcmChannel)

		for p := range opusRecv {
			user, userExists := GetSSRCMap().GetUser(p.SSRC)
			if !userExists {
				log.Warn().Uint32("p.SSRC", p.SSRC).Msg("user not found for ssrc, opus packet is going to be dropped")
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
					log.Err(err).Uint32("p.SSRC", p.SSRC).Msg("failed to create decoder")
					return
				}
				decoderMap[user.ID] = decoder
			}

			pcm, err := decoder.Decode(p.Opus, discordFrameSize, false)
			if err != nil {
				log.Err(err).Msg("failed to decode")
				return
			}

			pcmChannel <- NewVoiceBuffer(pcm, user.ID)
		}

		log.Error().Msg("We are past the opus decode, this shouldn't happen under normal circumatances")
	}()

	return pcmChannel
}
