package internal

import (
	"github.com/joho/godotenv"
)

type envVars struct {
	AwsAccessKey        string
	AwsSecretKey        string
	S3Endpoint          string
	S3Bucket            string
	AwsRegion           string
	PublicAssetEndpoint string
	AppEnv              string
	Port                string
	Host                string
	OpenAIApiKey        string
}

func newEnvVars() *envVars {
	envFile, err := godotenv.Read(".env")
	if err != nil {
		panic(err)
	}

	config := envVars{
		AwsAccessKey:        envFile["AWS_ACCESS_KEY"],
		AwsSecretKey:        envFile["AWS_SECRET_KEY"],
		S3Endpoint:          envFile["S3_ENDPOINT"],
		S3Bucket:            envFile["S3_BUCKET"],
		AwsRegion:           envFile["AWS_REGION"],
		PublicAssetEndpoint: envFile["PUBLIC_ASSET_ENDPOINT"],
		AppEnv:              envFile["ENV"],
		Port:                envFile["PORT"],
		Host:                envFile["HOST"],
		OpenAIApiKey:        envFile["OPENAI_API_KEY"],
	}

	return &config
}

var Env = newEnvVars()
