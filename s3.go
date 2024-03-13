package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

func NewAWSConfig() *aws.Config {
	if Config.Env == "dev" {
		return &aws.Config{
			Credentials: credentials.NewStaticCredentials(Config.AwsAccessKey, Config.AwsSecretKey, ""),
			Region:      aws.String(Config.AwsRegion),
			Endpoint:    aws.String(Config.S3Endpoint),
		}
	}
	return &aws.Config{
		Credentials: credentials.NewStaticCredentials(Config.AwsAccessKey, Config.AwsSecretKey, ""),
		Region:      aws.String(Config.AwsRegion),
		Endpoint:    aws.String(Config.S3Endpoint),
	}
}

var awsConfig = NewAWSConfig()

func UploadDirToS3(directory string) error {
	sess, err := session.NewSession(awsConfig)

	if err != nil {
		return err
	}

	svc := s3.New(sess)
	var wg sync.WaitGroup
	errCh := make(chan error)

	filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() { // Upload files only, not directories
			wg.Add(1)
			go func(wg *sync.WaitGroup) {
				defer wg.Done()
				file, err := os.Open(path)
				if err != nil {
					errCh <- err
					return
				}
				defer file.Close()

				objectKey := strings.Replace(filepath.ToSlash(path), "tmp/", "", 1)

				_, err = svc.PutObject(&s3.PutObjectInput{
					Bucket: aws.String(Config.S3Bucket),
					Key:    aws.String(objectKey),
					Body:   file,
				})
				if err != nil {
					errCh <- err
					return
				} else {
					fmt.Printf("Uploaded file: %s\n", objectKey)
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
