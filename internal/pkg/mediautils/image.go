package mediautils

import (
	"fmt"

	"github.com/google/uuid"
	ffmpeg "github.com/u2takey/ffmpeg-go"
)

const IMAGE_FORMAT_WEBP = "webp"
const IMAGE_FORMAT_JPEG = "jpeg"
const IMAGE_FORMAT_PNG = "png"

var imageFileFormats = []string{IMAGE_FORMAT_JPEG, IMAGE_FORMAT_PNG, IMAGE_FORMAT_WEBP}

func CheckValidImageFormat(fileType string) bool {
	for _, format := range imageFileFormats {
		if format == fileType {
			return true
		}
	}
	return false
}

func ConvertImage(filename string, outputType string) (string, error) {
	outputFileName := fmt.Sprintf("%s.%s", uuid.NewString(), outputType)

	if isValidFormat := CheckValidImageFormat(outputType); !isValidFormat {
		return "", fmt.Errorf("invalid image output format")
	}

	err := ffmpeg.Input(filename).Output(outputFileName, ffmpeg.KwArgs{
		"compression_level": "6",
	}).Run()

	if err != nil {
		return "", err
	}

	return outputFileName, nil
}
