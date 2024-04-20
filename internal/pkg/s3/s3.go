package s3

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/zihaolam/golang-media-upload-server/internal"
)

func newConfig() *aws.Config {
	if internal.Env.AppEnv == "dev" {
		return &aws.Config{
			Credentials: credentials.NewStaticCredentials(internal.Env.AwsAccessKey, internal.Env.AwsSecretKey, ""),
			Region:      aws.String(internal.Env.AwsRegion),
			Endpoint:    aws.String(internal.Env.S3Endpoint),
		}
	}
	return &aws.Config{
		Credentials: credentials.NewStaticCredentials(internal.Env.AwsAccessKey, internal.Env.AwsSecretKey, ""),
		Region:      aws.String(internal.Env.AwsRegion),
		Endpoint:    aws.String(internal.Env.S3Endpoint),
	}
}

type S3Client struct {
	awsCfg *aws.Config
	bucket string
}

func NewS3Client() *S3Client {
	return &S3Client{
		awsCfg: newConfig(),
		bucket: internal.Env.S3Bucket,
	}
}

func (sc *S3Client) UploadDirectory(directory string) error {
	sess, err := session.NewSession(sc.awsCfg)

	if err != nil {
		return err
	}

	svc := s3.New(sess)
	var wg sync.WaitGroup
	errCh := make(chan error)
	leafDirPath := filepath.Base(directory)
	filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() { // Upload files only, not directories
			wg.Add(1)
			go sc.UploadObject(&wg, errCh, svc, path, func(_ string) string {
				return filepath.Join(leafDirPath, info.Name())
			})
		}
		return nil
	})

	go func() {
		wg.Wait()
		close(errCh)
	}()

	for err := range errCh {
		return err
	}

	return nil
}

func getFileType(path string) string {
	if filepath.Ext(path) == ".m3u8" {
		return "application/x-mpegURL"
	}
	if filepath.Ext(path) == ".ts" {
		return "video/mp2t"
	}

	return ""
}

func (sc *S3Client) UploadObject(wg *sync.WaitGroup, errCh chan error, svc *s3.S3, path string, transformPathName func(string) string) {
	defer wg.Done()
	file, err := os.Open(path)
	if err != nil {
		errCh <- err
		return
	}
	defer file.Close()

	objectKey := transformPathName(path)

	contentType := getFileType(objectKey)

	opts := s3.PutObjectInput{
		Bucket: aws.String(sc.bucket),
		Key:    aws.String(objectKey),
		Body:   file,
	}

	if contentType != "" {
		opts.ContentType = aws.String(contentType)
	}

	_, err = svc.PutObject(&opts)

	if err != nil {
		errCh <- err
		return
	} else {
		fmt.Printf("Uploaded file: %s\n", objectKey)
	}
}
