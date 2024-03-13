package main

import (
	"github.com/joho/godotenv"
)

type EnvVars struct {
	AwsAccessKey        string
	AwsSecretKey        string
	S3Endpoint          string
	S3Bucket            string
	AwsRegion           string
	PublicAssetEndpoint string
	Env                 string
}

func GetConfig() *EnvVars {
	envFile, err := godotenv.Read(".env")
	if err != nil {
		panic(err)
	}

	config := EnvVars{
		AwsAccessKey:        envFile["AWS_ACCESS_KEY"],
		AwsSecretKey:        envFile["AWS_SECRET_KEY"],
		S3Endpoint:          envFile["S3_ENDPOINT"],
		S3Bucket:            envFile["S3_BUCKET"],
		AwsRegion:           envFile["AWS_REGION"],
		PublicAssetEndpoint: envFile["PUBLIC_ASSET_ENDPOINT"],
		Env:                 envFile["ENV"],
	}

	return &config
}

var Config = GetConfig()
