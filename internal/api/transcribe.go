package api

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	fileutils "github.com/zihaolam/golang-media-upload-server/internal/pkg/file"
	"github.com/zihaolam/golang-media-upload-server/internal/pkg/mediautils"
	"github.com/zihaolam/golang-media-upload-server/internal/pkg/openai"
	"github.com/zihaolam/golang-media-upload-server/internal/pkg/s3"
	"github.com/zihaolam/golang-media-upload-server/internal/pkg/utils"
)

type transcribeApi struct {
	api *api
}

func NewTranscribeApi(a *api) *transcribeApi {
	return &transcribeApi{
		api: a,
	}
}

func (ta *transcribeApi) Setup() {
	transcribeApiGroup := ta.api.GetV1Group().Group("/transcribe")
	transcribeApiGroup.Post("/audio", ta.getTranscribeAudioHandler())
}

func (ta *transcribeApi) getTranscribeAudioHandler() Handler {
	return func(c *fiber.Ctx) error {
		tmpDir, err := os.MkdirTemp("", uuid.NewString())
		if err != nil {
			log.Println(err)
			return fiber.ErrInternalServerError
		}

		defer os.RemoveAll(tmpDir)

		tmpFileName, err := fileutils.SaveFileFromCtxToDir(c, "file", tmpDir)

		if err != nil {
			log.Println(err)
			return fiber.ErrInternalServerError
		}

		var audioFileName = tmpFileName
		if strings.HasSuffix(tmpFileName, ".mp4") {
			_audioFileName, err := mediautils.ExtractAudio(tmpFileName)

			if err != nil {
				return fiber.ErrInternalServerError
			}

			audioFileName = *_audioFileName
		}
		englishVTTFileName, err := openai.TranscribeAudio(audioFileName, tmpDir)

		if err != nil {
			log.Println(err)
			return fiber.ErrInternalServerError
		}

		mandarinVTTFileName, err := openai.TranslateVTT(englishVTTFileName, openai.MandarinTranslationLanguage, tmpDir)

		if err != nil {
			log.Println(err)
			return fiber.ErrInternalServerError
		}

		s3Client := s3.NewS3Client()
		ctx := context.Context(context.Background())

		resps, errs := utils.Parallelize(func(arg openai.SubtitleTrack) (openai.SubtitleTrack, error) {
			key, err := s3Client.UploadObject(ctx, arg.Src, func(path string) string {
				return filepath.Base(path)
			})
			return openai.SubtitleTrack{
				Src:      s3.GetAbsolutePath(*key),
				Language: arg.Language,
			}, err
		}, openai.SubtitleTrack{
			Src:      englishVTTFileName,
			Language: openai.EnglishTranslationLanguage,
		}, openai.SubtitleTrack{
			Src:      mandarinVTTFileName,
			Language: openai.MandarinTranslationLanguage,
		})

		if len(errs) > 0 {
			log.Println(errs)
			return fiber.ErrInternalServerError
		}

		return c.JSON(fiber.Map{
			"tracks": resps,
		})
	}
}
