package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	ffmpeg "github.com/u2takey/ffmpeg-go"
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

type playlistChan struct {
	playlistCh chan Playlist
	errorCh    chan error
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

func generateSegments(resolution Resolution, ch *playlistChan, wg *sync.WaitGroup, outputDir string, outputPrefix string, tempVideoFileName string) {
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
		"hls_segment_filename": segmentFileName + "_%03d.ts",
	}).Run()

	if err != nil {
		ch.errorCh <- err
		return
	}

	ch.playlistCh <- Playlist{
		Resolution:     resolution,
		OutputFileName: outputFileName,
	}
}

func generateOutputFileName(outputDir, outputName string, resolution string) string {
	return fmt.Sprintf("%s/%s_%s.m3u8", outputDir, outputName, resolution)
}

func replaceBasePath(directory string, basePath string, prefix string) {
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

func transcodeFileHandler(c *fiber.Ctx) error {
	form, err := c.MultipartForm()
	if err != nil {
		return err
	}

	file := form.File["file"][0]

	randomFileName := fmt.Sprintf("%s%s", uuid.New().String(), filepath.Ext(file.Filename))

	storedTempFileName := fmt.Sprintf("./tmp/%s", randomFileName)

	if err := c.SaveFile(file, storedTempFileName); err != nil {
		return err
	}

	defer func() {
		os.Remove(storedTempFileName)
	}()

	fileOutputDir := "./tmp/" + uuid.New().String()
	fileOutputPrefix := uuid.New().String()

	if err := os.MkdirAll(fileOutputDir, os.ModePerm); err != nil {
		return err
	}

	playlistArr := []Playlist{}

	ch := playlistChan{
		playlistCh: make(chan Playlist),
		errorCh:    make(chan error),
	}

	var wg sync.WaitGroup
	for _, resolution := range resolutions {
		wg.Add(1)
		go generateSegments(resolution, &ch, &wg, fileOutputDir, fileOutputPrefix, storedTempFileName)
	}

	go func() {
		wg.Wait()
		close(ch.playlistCh)
		close(ch.errorCh)
	}()

	for {
		select {
		case playlist, ok := <-ch.playlistCh:
			if !ok {
				ch.playlistCh = nil
				break
			}
			playlistArr = append(playlistArr, playlist)
		case err, ok := <-ch.errorCh:
			if !ok {
				ch.errorCh = nil
				break
			}
			if err != nil {
				return err
			}
		}
		if ch.errorCh == nil && ch.playlistCh == nil {
			break
		}
	}

	masterPlaylist := "#EXTM3U\n"

	for _, playlist := range playlistArr {
		playlistFileName := strings.Replace(playlist.OutputFileName, fileOutputDir+"/", "", 1)
		masterPlaylist += fmt.Sprintf("#EXT-X-STREAM-INF:BANDWIDTH=%s,RESOLUTION=%s\n%s\n", playlist.Resolution.Bandwidth, playlist.Resolution.Resolution, playlistFileName)
	}

	masterPlaylistFileName := fmt.Sprintf("%s/%s_master.m3u8", fileOutputDir, fileOutputPrefix)
	fmt.Println("masterplaylist name", masterPlaylistFileName)
	f, err := os.Create(masterPlaylistFileName)
	if err != nil {
		return err
	}

	_, err = f.WriteString(masterPlaylist)
	if err != nil {
		return err
	}

	f.Sync()

	replaceBasePath(fileOutputDir, fileOutputPrefix, Config.PublicAssetEndpoint)
	err = UploadDirToS3(fileOutputDir)
	if err != nil {
		return err
	}

	return c.SendString("Transcoding done")
}
