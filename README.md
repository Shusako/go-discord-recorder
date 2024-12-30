# Discord Transcriber

Affectionately known as Parsley. Written in golang.
Parsley joins a given channel and transcribes what is spoken by each individual person.
These transcriptions are outputted to the 'transcripts' folder.

Example:

```
[  3.26s->  5.18s] [shusako] I don't know about that one, Chief.
[  7.19s-> 10.33s] [sablemyst] I have my little seconds timer clock open.
[ 20.19s-> 24.51s] [lostsky000] I'm just waiting for the day when I can tell Parsley to do my roles for me.
```

# Installation

- You'll have to setup a discord bot through https://discord.com/developers/applications
  - It needs Voice Connect access
  - Copy the client secret and set it up in the GO_DISCORD_RECORDER_TOKEN environment variable
- Setup your guild ID / channel ID in the GO_DISCORD_RECORDER_GUILD and GO_DISCORD_RECORDER_CHANNEL environment variables
- Download a whisper model that is compatible with whisper.cpp from https://huggingface.co/ggerganov/whisper.cpp/tree/main - place it in the resources/ folder and post the relative path into the GO_DISCORD_RECORDER_MODEL_PATH environment variable (e.g. resources/ggml-small.en.bin)
- Clone repo
- docker compose up --build
