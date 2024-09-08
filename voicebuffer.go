package main

import "time"

type VoiceBuffer[T any] struct {
	pcm          []T
	identifier   string
	firstUpdated time.Time
	lastUpdated  time.Time
}

func NewVoiceBuffer[T any](pcm []T, identifier string) VoiceBuffer[T] {
	return VoiceBuffer[T]{pcm, identifier, time.Now(), time.Now()}
}

func (vb *VoiceBuffer[T]) Buffer(otherBuffer *VoiceBuffer[T]) {
	if len(vb.pcm) == 0 {
		vb.firstUpdated = otherBuffer.firstUpdated
	}
	vb.pcm = append(vb.pcm, otherBuffer.pcm...)
	vb.lastUpdated = time.Now()
}

func (vb *VoiceBuffer[T]) Clear() {
	vb.pcm = []T{}
	vb.firstUpdated = time.Time{}
	vb.lastUpdated = time.Time{}
}
