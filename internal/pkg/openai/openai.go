package openai

import (
	"context"
	"os"

	"github.com/google/uuid"
	openai "github.com/sashabaranov/go-openai"
	"github.com/zihaolam/golang-media-upload-server/internal"
)

func TranscribeAudio(mp3FileName, outputDir string) (string, error) {
	client := openai.NewClient(internal.Env.OpenAIApiKey)

	ctx := context.Background()
	resp, err := client.CreateTranscription(ctx, openai.AudioRequest{
		Model:    openai.Whisper1,
		FilePath: mp3FileName,
		Format:   openai.AudioResponseFormatVTT,
	})
	if err != nil {
		return "", err
	}

	f, err := os.Create(uuid.NewString() + ".vtt")
	if err != nil {
		return "", err
	}
	defer f.Close()

	if _, err := f.WriteString(resp.Text); err != nil {
		return "", err
	}

	return f.Name(), nil
}
