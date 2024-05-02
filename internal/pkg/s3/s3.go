package s3

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
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

func (sc *S3Client) UploadDirectory(ctx aws.Context, directory string) error {
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
				_, err := sc.UploadObject(ctx, path, func(_ string) string {
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

func (sc *S3Client) newSession() (*session.Session, error) {
	return session.NewSession(sc.awsCfg)
}

func (sc *S3Client) UploadObject(ctx aws.Context, path string, transformPathName func(string) string) (*string, error) {
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

	_, err = sc.s3.PutObjectWithContext(ctx, &opts)

	if err != nil {
		return nil, err
	}

	log.Printf("Uploaded file: %s\n", objectKey)

	return &objectKey, nil
}

func GetAbsolutePath(path string) string {
	return fmt.Sprintf("%s/%s", internal.Env.PublicAssetEndpoint, path)
}

// Download object to specific file and return the file, remember to cleanup file after use
func (sc *S3Client) GetObject(ctx context.Context, key string) (*os.File, error) {
	strippedKey := strings.Replace(key, internal.Env.PublicAssetEndpoint, "", 1)
	sess, err := sc.newSession()
	if err != nil {
		return nil, err
	}

	file, err := os.CreateTemp("", fmt.Sprintf("*_%s", filepath.Base(key)))
	log.Println(file.Name())
	if err != nil {
		return nil, err
	}

	downloader := s3manager.NewDownloader(sess)
	if _, err = downloader.DownloadWithContext(ctx, file, &s3.GetObjectInput{
		Bucket: aws.String(sc.bucket),
		Key:    aws.String(strippedKey),
	}); err != nil {
		return nil, err
	}

	return file, nil
}

func (sc *S3Client) DeleteObject(ctx context.Context, key string) error {
	_, err := sc.s3.DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(sc.bucket),
		Key:    aws.String(key),
	})
	return err
}

func (sc *S3Client) DeleteDirectory(ctx context.Context, dir string) error {
	objects, err := sc.s3.ListObjectsV2WithContext(ctx, &s3.ListObjectsV2Input{
		Prefix: aws.String(dir),
		Bucket: aws.String(sc.bucket),
	})

	if err != nil {
		return err
	}

	if len(objects.Contents) == 0 {
		return nil
	}

	objectsToDelete := make([]*s3.ObjectIdentifier, 0, len(objects.Contents))

	for _, object := range objects.Contents {
		objectsToDelete = append(objectsToDelete, &s3.ObjectIdentifier{
			Key: object.Key,
		})
	}

	_, err = sc.s3.DeleteObjectsWithContext(ctx, &s3.DeleteObjectsInput{
		Bucket: aws.String(sc.bucket),
		Delete: &s3.Delete{
			Objects: objectsToDelete,
		},
	})

	return err
}
