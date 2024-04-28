package mediautils

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	ffmpeg "github.com/u2takey/ffmpeg-go"
	"github.com/zihaolam/golang-media-upload-server/internal"
	fileutils "github.com/zihaolam/golang-media-upload-server/internal/pkg/file"
	"github.com/zihaolam/golang-media-upload-server/internal/pkg/s3"
)

type Resolution struct {
	Resolution   string
	VideoBitRate string
	AudioBitRate string
	Bandwidth    string
}

type Playlist struct {
	Resolution     Resolution
	OutputFileName string
}

type HLSSegmentOutput struct {
	playlists      []Playlist
	masterPlaylist string
}

var resolutions = []Resolution{
	{
		Resolution:   "320x180",
		VideoBitRate: "500k",
		Bandwidth:    "676800",
		AudioBitRate: "64k",
	},
	{
		Resolution:   "854x480",
		VideoBitRate: "1000k",
		Bandwidth:    "1353600",
		AudioBitRate: "128k",
	},
	{
		Resolution:   "1280x720",
		VideoBitRate: "2500k",
		AudioBitRate: "192k",
		Bandwidth:    "3230400",
	},
}

func generateHLSSegments(resolution Resolution, playlistCh chan Playlist, errCh chan error, wg *sync.WaitGroup, outputDir string, outputPrefix string, tempVideoFileName string) {
	defer wg.Done()
	outputFileName := generateOutputFileName(outputDir, outputPrefix, resolution.Resolution)
	segmentFileName := strings.Replace(outputFileName, ".m3u8", "_m3u8", 1)
	err := ffmpeg.Input(tempVideoFileName).Output(outputFileName, ffmpeg.KwArgs{
		"c:v":                  "h264",
		"b:v":                  resolution.VideoBitRate,
		"c:a":                  "aac",
		"b:a":                  resolution.AudioBitRate,
		"vf":                   "scale=" + resolution.Resolution,
		"f":                    "hls",
		"hls_time":             "10",
		"hls_list_size":        "0",
		"crf":                  "20",
		"hls_segment_filename": segmentFileName + "_%03d.ts",
	}).Run()

	if err != nil {
		errCh <- err
		return
	}

	playlistCh <- Playlist{
		Resolution:     resolution,
		OutputFileName: outputFileName,
	}
}

func generateOutputFileName(outputDir, outputName string, resolution string) string {
	return fmt.Sprintf("%s/%s_%s.m3u8", outputDir, outputName, resolution)
}

func generateMasterPlaylist(playlists *[]Playlist, outputDir string) string {
	masterPlaylist := "#EXTM3U\n"
	for _, playlist := range *playlists {
		playlistFileName := strings.Replace(playlist.OutputFileName, outputDir+"/", "", 1)
		masterPlaylist += fmt.Sprintf("#EXT-X-STREAM-INF:BANDWIDTH=%s,RESOLUTION=%s\n%s\n", playlist.Resolution.Bandwidth, playlist.Resolution.Resolution, playlistFileName)
	}
	return masterPlaylist
}

func generateSegmentsForResolutions(resolutions []Resolution, outputDir, outputPrefix, storedTempFileName string) (*HLSSegmentOutput, error) {
	playlistArr := []Playlist{}

	playlistCh := make(chan Playlist)
	errorCh := make(chan error)

	var wg sync.WaitGroup
	for _, resolution := range resolutions {
		wg.Add(1)
		go generateHLSSegments(resolution, playlistCh, errorCh, &wg, outputDir, outputPrefix, storedTempFileName)
	}

	go func() {
		wg.Wait()
		close(playlistCh)
		close(errorCh)
	}()

	for {
		select {
		case playlist, ok := <-playlistCh:
			if !ok {
				playlistCh = nil
				break
			}
			playlistArr = append(playlistArr, playlist)
		case err, ok := <-errorCh:
			if !ok {
				errorCh = nil
				break
			}
			if err != nil {
				return nil, err
			}
		}
		if errorCh == nil && playlistCh == nil {
			break
		}
	}

	masterPlaylist := generateMasterPlaylist(&playlistArr, outputDir)

	return &HLSSegmentOutput{
		playlists:      playlistArr,
		masterPlaylist: masterPlaylist,
	}, nil
}

func replaceHLSSegmentsBasePath(directory string, basePath string, prefix string) {
	filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(path, ".m3u8") {
			fileBytes, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			content := strings.ReplaceAll(string(fileBytes), basePath, prefix+"/"+basePath)

			err = os.WriteFile(path, []byte(content), info.Mode())

			if err != nil {
				return err
			}
		}

		return nil
	})
}

func getUploadedS3HLSMasterDirectory(masterPlaylistFileName, tmpDir string) string {
	return internal.Env.PublicAssetEndpoint + strings.Replace(masterPlaylistFileName, tmpDir, "", 1)
}

// transcodes video to hls and uploads to s3 bucket
func TranscodeVideoToHLS(videoFilename, tmpDir string) (string, error) {
	fileOutputDirLeaf := uuid.New().String()
	fileOutputDir := filepath.Join(tmpDir, fileOutputDirLeaf)
	fileOutputPrefix := uuid.New().String()

	if err := os.MkdirAll(fileOutputDir, os.ModePerm); err != nil {
		return "", err
	}

	hlsSegments, err := generateSegmentsForResolutions(resolutions, fileOutputDir, fileOutputPrefix, videoFilename)
	if err != nil {
		return "", err
	}

	masterPlaylistFileName := fmt.Sprintf("%s/%s_master.m3u8", fileOutputDir, fileOutputPrefix)

	if err := fileutils.WriteToFile(masterPlaylistFileName, hlsSegments.masterPlaylist); err != nil {
		return "", err
	}

	newDirPrefix := internal.Env.PublicAssetEndpoint + "/" + fileOutputDirLeaf

	if err = UploadTranscodedSegmentsToS3(fileOutputDir, fileOutputPrefix, newDirPrefix); err != nil {
		log.Println(err)
		return "", fiber.ErrInternalServerError
	}

	uploadedDirectory := getUploadedS3HLSMasterDirectory(masterPlaylistFileName, tmpDir)

	return uploadedDirectory, nil
}

func UploadTranscodedSegmentsToS3(directory, directoryPrefix, newDirPrefix string) error {
	s3Client := s3.NewS3Client()
	ctx := context.Context(context.Background())
	replaceHLSSegmentsBasePath(directory, directoryPrefix, newDirPrefix)
	return s3Client.UploadDirectory(ctx, directory)
}

func ExtractAudio(videoFileName string) (*string, error) {
	audioFileName := strings.Replace(videoFileName, ".mp4", ".mp3", 1)
	err := ffmpeg.Input(videoFileName).Output(audioFileName, ffmpeg.KwArgs{"map": "0:a"}).Run()
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return &audioFileName, nil
}
