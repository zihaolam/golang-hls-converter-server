package openai

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	openai "github.com/sashabaranov/go-openai"
	"github.com/zihaolam/golang-media-upload-server/internal"
)

const MandarinTranslationLanguage = "cn"
const EnglishTranslationLanguage = "en"

type SubtitleTrack struct {
	Src      string `json:"src"`
	Language string `json:"language"`
}

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

	f, err := os.Create(filepath.Join(outputDir, uuid.NewString()+".vtt"))
	if err != nil {
		return "", err
	}
	defer f.Close()

	if _, err := f.WriteString(resp.Text); err != nil {
		return "", err
	}

	return f.Name(), nil
}

func TranslateVTT(vttFileName, language, outputDir string) (string, error) {
	if language != MandarinTranslationLanguage && language != EnglishTranslationLanguage {
		return "", fmt.Errorf("invalid language")
	}
	f, err := os.ReadFile(vttFileName)
	if err != nil {
		return "", err
	}

	client := openai.NewClient(internal.Env.OpenAIApiKey)
	ctx := context.Background()

	resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: openai.GPT4Turbo,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    "system",
				Content: "Pretend you are an expert language translator for subtitles.",
			},
			{
				Role:    "user",
				Content: fmt.Sprintf(" Translate the following VTT to %s. Do not include any explanations, only provide a RFC8216 compliant VTT file without deviation.\n\n%s", language, string(f)),
			},
		},
		N: 1,
	})

	if err != nil {
		return "", err
	}

	translatedVtt := resp.Choices[0].Message.Content

	translationFile, err := os.Create(fmt.Sprintf("%s/%s_%s.vtt", outputDir, uuid.NewString(), language))
	if err != nil {
		return "", err
	}
	defer translationFile.Close()
	translationFile.WriteString(translatedVtt)

	return translationFile.Name(), nil
}
