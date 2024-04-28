package fileutils

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func SaveFileFromCtxToDir(c *fiber.Ctx, formFileKey, dir string) (string, error) {
	form, err := c.MultipartForm()
	if err != nil {
		return "", err
	}

	file := form.File[formFileKey][0]

	randomFileName := fmt.Sprintf("%s%s", uuid.New().String(), filepath.Ext(file.Filename))

	storedTempFileName := fmt.Sprintf("%s/%s", dir, randomFileName)

	if err := c.SaveFile(file, storedTempFileName); err != nil {
		return "", err
	}

	return storedTempFileName, nil
}

func WriteToFile(filename string, data string) error {
	f, err := os.Create(filename)

	if err != nil {
		return err
	}

	_, err = f.WriteString(data)
	if err != nil {
		return err
	}

	return f.Sync()
}
