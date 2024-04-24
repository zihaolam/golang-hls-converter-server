package s3

import (
	"log"
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
	s3     *s3.S3
}

func NewS3Client() *S3Client {
	awsCfg := newConfig()
	sess, err := session.NewSession(awsCfg)
	if err != nil {
		log.Fatal(err)
	}
	s3 := s3.New(sess)
	return &S3Client{
		s3:     s3,
		awsCfg: newConfig(),
		bucket: internal.Env.S3Bucket,
	}
}

func (sc *S3Client) UploadDirectory(directory string) error {
	var wg sync.WaitGroup
	errCh := make(chan error)
	leafDirPath := filepath.Base(directory)
	filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() { // Upload files only, not directories
			wg.Add(1)
			go func(wg *sync.WaitGroup) {
				defer wg.Done()
				_, err := sc.UploadObject(path, func(_ string) string {
					return filepath.Join(leafDirPath, info.Name())
				})
				if err != nil {
					errCh <- err
				}
			}(&wg)
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

func (sc *S3Client) UploadObject(path string, transformPathName func(string) string) (*string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	objectKey := path

	if transformPathName != nil {
		objectKey = transformPathName(path)
	}

	contentType := getFileType(objectKey)

	opts := s3.PutObjectInput{
		Bucket: aws.String(sc.bucket),
		Key:    aws.String(objectKey),
		Body:   file,
	}

	if contentType != "" {
		opts.ContentType = aws.String(contentType)
	}

	_, err = sc.s3.PutObject(&opts)

	if err != nil {
		return nil, err
	}

	log.Printf("Uploaded file: %s\n", objectKey)

	return &objectKey, nil
}
