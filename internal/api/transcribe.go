package api

import (
	"log"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	fileutils "github.com/zihaolam/golang-media-upload-server/internal/pkg/file"
	"github.com/zihaolam/golang-media-upload-server/internal/pkg/mediautils"
	"github.com/zihaolam/golang-media-upload-server/internal/pkg/openai"
	"github.com/zihaolam/golang-media-upload-server/internal/pkg/s3"
)

type transcribeApi struct {
	api *api
}

func NewTranscribeApi(a *api) *transcribeApi {
	return &transcribeApi{
		api: a,
	}
}

func (ta *transcribeApi) RegisterRoutes() {
	transcribeApiGroup := ta.api.GetV1Group().Group("/transcribe")
	transcribeApiGroup.Post("/audio", ta.getTranscribeAudioHandler())
}

func (ta *transcribeApi) getTranscribeAudioHandler() Handler {
	return func(c *fiber.Ctx) error {
		form, err := c.MultipartForm()
		if err != nil {
			log.Println(err)
			return fiber.ErrBadRequest
		}

		file := form.File["file"][0]
		tmpDir, err := os.MkdirTemp("", uuid.NewString())
		if err != nil {
			log.Println(err)
			return fiber.ErrInternalServerError
		}

		defer os.RemoveAll(tmpDir)

		tmpFileName, err := fileutils.SaveFileFromCtxToDir(c, file, tmpDir)

		if err != nil {
			return fiber.ErrInternalServerError
		}

		var audioFileName = tmpFileName
		if strings.HasSuffix(file.Filename, ".mp4") {
			_audioFileName, err := mediautils.ExtractAudio(tmpFileName)

			if err != nil {
				return fiber.ErrInternalServerError
			}

			audioFileName = *_audioFileName
		}

		transcriptFileName, err := openai.TranscribeAudio(audioFileName, tmpDir)

		if err != nil {
			return fiber.ErrInternalServerError
		}

		s3Client := s3.NewS3Client()
		key, err := s3Client.UploadObject(transcriptFileName, nil)

		if err != nil {
			log.Println(err)
			return fiber.ErrInternalServerError
		}

		assetPath := mediautils.GetAbsolutePath(*key)

		return c.JSON(fiber.Map{
			"transcript": assetPath,
		})
	}
}
