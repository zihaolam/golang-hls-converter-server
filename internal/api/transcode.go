package api

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	ffprobe "github.com/vansante/go-ffprobe"
	fileutils "github.com/zihaolam/golang-media-upload-server/internal/pkg/file"
	"github.com/zihaolam/golang-media-upload-server/internal/pkg/job"
	"github.com/zihaolam/golang-media-upload-server/internal/pkg/mediautils"
	"github.com/zihaolam/golang-media-upload-server/internal/pkg/openai"
	"github.com/zihaolam/golang-media-upload-server/internal/pkg/s3"
	"github.com/zihaolam/golang-media-upload-server/internal/pkg/utils"
)

type transcodeApi struct {
	api   *api
	jobCh chan job.Job
}

func NewTranscodeApi(a *api) *transcodeApi {
	return &transcodeApi{
		api:   a,
		jobCh: make(chan job.Job),
	}
}

const JOB_ID_PARAM = "jobId"

func (ta *transcodeApi) Setup() {
	transcodeApiGroup := ta.api.NewRouteGroup("/transcode")
	transcodeApiGroup.Post("/video", ta.handleVideoTranscode())
	transcodeApiGroup.Post(fmt.Sprintf("/job/:%s", JOB_ID_PARAM), ta.handleVideoTranscodeJob())
	transcodeApiGroup.Post("/image", ta.handleImageTranscode())
	ta.handleJobs()
}

func (ta *transcodeApi) handleVideoTranscode() Handler {
	return func(c *fiber.Ctx) error {
		tmpDir, err := os.MkdirTemp("", uuid.NewString())

		defer os.RemoveAll(tmpDir)

		if err != nil {
			log.Println(err)
			return fiber.ErrInternalServerError
		}

		tmpVideoFilename, err := fileutils.SaveFileFromCtxToDir(c, "file", tmpDir)

		if !strings.HasSuffix(tmpVideoFilename, ".mp4") {
			return fiber.ErrBadRequest
		}

		if err != nil {
			log.Println(err)
			return fiber.ErrInternalServerError
		}

		data, err := ffprobe.GetProbeData(tmpVideoFilename, 120000*time.Millisecond)

		if err != nil {
			log.Println(err)
			return fiber.ErrInternalServerError
		}

		transcodedVideoMasterPlaylist, err := mediautils.TranscodeVideoToHLS(tmpVideoFilename, tmpDir)
		if err != nil {
			log.Println(err)
			return fiber.ErrInternalServerError
		}

		return c.JSON(fiber.Map{
			"dir":           transcodedVideoMasterPlaylist,
			"videoDuration": data.Format.DurationSeconds,
		})
	}
}

func (ta *transcodeApi) handleImageTranscode() Handler {
	return func(c *fiber.Ctx) error {
		tmpDir, err := os.MkdirTemp("", uuid.NewString())

		defer os.RemoveAll(tmpDir)

		if err != nil {
			return fiber.ErrInternalServerError
		}

		tmpImageFilename, err := fileutils.SaveFileFromCtxToDir(c, "file", tmpDir)

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

func (ta *transcodeApi) handleVideoTranscodeJob() Handler {
	jobService := job.NewJobService()
	return func(c *fiber.Ctx) error {
		jobId := c.Params(JOB_ID_PARAM)
		if jobId == "" {
			return fiber.ErrBadRequest
		}

		j, err := jobService.GetJob(jobId)

		if err != nil {
			log.Println(err)
			return fiber.ErrNotFound
		}

		ta.jobCh <- *j

		return nil
	}
}

func (ta *transcodeApi) handleJobs() {
	js := job.NewJobService()

	go func() {
		transcodeJobLock := new(sync.Mutex)
		for j := range ta.jobCh {
			go func(j job.Job, jLock *sync.Mutex) {
				jLock.Lock()
				defer jLock.Unlock()
				if err := ta.runTranscodeJob(js, j); err != nil {
					log.Println(err)
					js.SendJobProcessingFailedWebhook(j.Id, err)
				}
			}(j, transcodeJobLock)
		}
	}()
}

func (ta *transcodeApi) runTranscodeJob(js *job.JobService, j job.Job) error {
	go js.SendJobProcessingStartedWebhook(j.Id)

	ctx := context.Context(context.Background())
	s3Client := s3.NewS3Client()

	file, err := s3Client.GetObject(ctx, j.VideoUrl)
	if err != nil {
		go js.SendJobProcessingFailedWebhook(j.Id, err)
		log.Println(err)
		return err
	}

	if j.Status != "pending" {
		err = fmt.Errorf("job status has started")
		log.Println(err)
		return err
	}

	defer file.Close()
	defer os.Remove(file.Name())
	defer s3Client.DeleteObject(ctx, j.VideoUrl)

	data, err := ffprobe.GetProbeData(file.Name(), 120000*time.Millisecond)

	if err != nil {
		go js.SendJobProcessingFailedWebhook(j.Id, err)
		log.Println(err)
		return err
	}

	tmpDir, err := os.MkdirTemp("", uuid.NewString())

	if err != nil {
		go js.SendJobProcessingFailedWebhook(j.Id, err)
		log.Println(err)
		return err
	}

	var wg sync.WaitGroup
	wg.Add(2)
	errCh := make(chan error)
	transcodedVideoMasterPlaylist := ""

	subtitleTracks := []openai.SubtitleTrack{}
	go func(wg *sync.WaitGroup, videoDirectory *string) {
		defer wg.Done()
		transcodedVideoMasterPlaylist, err := mediautils.TranscodeVideoToHLS(file.Name(), tmpDir)
		if err != nil {
			errCh <- err
		}
		*videoDirectory = transcodedVideoMasterPlaylist
	}(&wg, &transcodedVideoMasterPlaylist)

	go func(wg *sync.WaitGroup, subtitleTracks *[]openai.SubtitleTrack) {
		defer wg.Done()

		audioFileName, err := mediautils.ExtractAudio(file.Name())

		if err != nil {
			log.Println(err)
			errCh <- err
			return
		}

		defer os.Remove(*audioFileName)

		englishVTTFileName, err := openai.TranscribeAudio(*audioFileName, tmpDir)
		if err != nil {
			errCh <- err
		}

		mandarinVTTFileName, err := openai.TranslateVTT(englishVTTFileName, openai.MandarinTranslationLanguage, tmpDir)
		if err != nil {
			errCh <- err
		}

		*subtitleTracks = append(*subtitleTracks, openai.SubtitleTrack{
			Src:      englishVTTFileName,
			Language: openai.EnglishTranslationLanguage,
		}, openai.SubtitleTrack{
			Src:      mandarinVTTFileName,
			Language: openai.MandarinTranslationLanguage,
		})
	}(&wg, &subtitleTracks)

	go func(wg *sync.WaitGroup) {
		wg.Wait()
		close(errCh)
	}(&wg)

	for err := range errCh {
		go js.SendJobProcessingFailedWebhook(j.Id, err)
		log.Println(err)
		return err
	}

	resps, errs := utils.Parallelize(func(arg openai.SubtitleTrack) (openai.SubtitleTrack, error) {
		key, err := s3Client.UploadObject(ctx, arg.Src, func(path string) string {
			return filepath.Base(path)
		})
		return openai.SubtitleTrack{
			Src:      s3.GetAbsolutePath(*key),
			Language: arg.Language,
		}, err
	}, subtitleTracks...)

	if len(errs) > 0 {
		go js.SendJobProcessingFailedWebhook(j.Id, errs[0])
		log.Println(errs[0])
		return err
	}

	if err := js.SendJobCompletionWebhook(&job.JobCompletionRequest{
		Id:             j.Id,
		Status:         job.StatusDone,
		VideoUrl:       transcodedVideoMasterPlaylist,
		SubtitleTracks: resps,
		VideoDuration:  data.Format.DurationSeconds,
	}); err != nil {
		log.Println(err)
		return err
	}

	return nil
}
