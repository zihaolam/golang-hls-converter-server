package api

import (
	"log"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	ffprobe "github.com/vansante/go-ffprobe"
	fileutils "github.com/zihaolam/golang-media-upload-server/internal/pkg/file"
	"github.com/zihaolam/golang-media-upload-server/internal/pkg/mediautils"
)

type transcodeApi struct {
	api *api
}

func NewTranscodeApi(a *api) *transcodeApi {
	return &transcodeApi{
		api: a,
	}
}

func (ta *transcodeApi) RegisterRoutes() {
	transcodeApiGroup := ta.api.GetV1Group().Group("/transcode")
	transcodeApiGroup.Post("/video", ta.getTranscodeVideoHandler())
	transcodeApiGroup.Post("/image", ta.getTranscodeImageHandler())
}

func (ta *transcodeApi) getTranscodeVideoHandler() Handler {
	return func(c *fiber.Ctx) error {
		form, err := c.MultipartForm()

		if err != nil {
			return fiber.ErrBadRequest
		}

		file := form.File["file"][0]

		if !strings.HasSuffix(file.Filename, ".mp4") {
			return fiber.ErrBadRequest
		}

		tmpDir, err := os.MkdirTemp("", uuid.NewString())

		defer os.RemoveAll(tmpDir)

		if err != nil {
			log.Println(err)
			return fiber.ErrInternalServerError
		}

		tmpVideoFilename, err := fileutils.SaveFileFromCtxToDir(c, file, tmpDir)

		if err != nil {
			log.Println(err)
			return fiber.ErrInternalServerError
		}

		data, err := ffprobe.GetProbeData(tmpVideoFilename, 120000*time.Millisecond)

		if err != nil {
			log.Println(err)
			return fiber.ErrInternalServerError
		}

		transcodedVideoDirectory, err := mediautils.TranscodeVideoToHLS(tmpVideoFilename, tmpDir)
		if err != nil {
			log.Println(err)
			return fiber.ErrInternalServerError
		}

		return c.JSON(fiber.Map{
			"dir":           transcodedVideoDirectory,
			"videoDuration": data.Format.DurationSeconds,
		})
	}
}

func (ta *transcodeApi) getTranscodeImageHandler() Handler {
	return func(c *fiber.Ctx) error {
		form, err := c.MultipartForm()

		if err != nil {
			return fiber.ErrBadRequest
		}

		file := form.File["file"][0]

		tmpDir, err := os.MkdirTemp("", uuid.NewString())

		defer os.RemoveAll(tmpDir)

		if err != nil {
			return fiber.ErrInternalServerError
		}

		tmpImageFilename, err := fileutils.SaveFileFromCtxToDir(c, file, tmpDir)

		if err != nil {
			return fiber.ErrInternalServerError
		}

		transcodedImageFilename, err := mediautils.ConvertImage(tmpImageFilename, mediautils.IMAGE_FORMAT_WEBP)

		if err != nil {
			return fiber.ErrInternalServerError
		}

		return c.SendFile(transcodedImageFilename)
	}
}
