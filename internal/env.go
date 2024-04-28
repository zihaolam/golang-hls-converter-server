package internal

import (
	"log"

	"github.com/joho/godotenv"
	"github.com/zihaolam/golang-media-upload-server/internal/pkg/utils"
)

type envVars struct {
	AwsAccessKey           string `validate:"required,min=1"`
	AwsSecretKey           string `validate:"required,min=1"`
	S3Endpoint             string `validate:"required,min=1"`
	S3Bucket               string `validate:"required,min=1"`
	AwsRegion              string `validate:"required,min=1"`
	PublicAssetEndpoint    string `validate:"required,min=1"`
	AppEnv                 string `validate:"required,min=1"`
	Port                   string `validate:"required,min=1"`
	Host                   string `validate:"required,min=1"`
	OpenAIApiKey           string `validate:"required,min=1"`
	VideoPlatformApiKey    string `validate:"required,min=1"`
	VideoPlatformServerUrl string `validate:"required,min=1"`
	SecretKey              string `validate:"required,min=1"`
}

func newEnvVars() *envVars {
	envFile, err := godotenv.Read(".env")
	if err != nil {
		panic(err)
	}

	config := envVars{
		AwsAccessKey:           envFile["AWS_ACCESS_KEY"],
		AwsSecretKey:           envFile["AWS_SECRET_KEY"],
		S3Endpoint:             envFile["S3_ENDPOINT"],
		S3Bucket:               envFile["S3_BUCKET"],
		AwsRegion:              envFile["AWS_REGION"],
		PublicAssetEndpoint:    envFile["PUBLIC_ASSET_ENDPOINT"],
		AppEnv:                 envFile["ENV"],
		Port:                   envFile["PORT"],
		Host:                   envFile["HOST"],
		OpenAIApiKey:           envFile["OPENAI_API_KEY"],
		VideoPlatformApiKey:    envFile["VIDEO_PLATFORM_API_KEY"],
		VideoPlatformServerUrl: envFile["VIDEO_PLATFORM_SERVER_URL"],
		SecretKey:              envFile["SECRET_KEY"],
	}

	if err := utils.Validate(config); err != nil {
		log.Fatal(err)
	}

	return &config
}

var Env = newEnvVars()
