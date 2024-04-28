package job

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/zihaolam/golang-media-upload-server/internal"
	"github.com/zihaolam/golang-media-upload-server/internal/pkg/openai"
)

const StatusPending = "pending"
const StatusProcessing = "processing"
const StatusDone = "done"
const StatusFailed = "failed"

type JobService struct {
	http *http.Client
}

func NewJobService() *JobService {
	return &JobService{
		http: http.DefaultClient,
	}
}

func newVideoPlatformApiRequest(method, path string, body []byte) (*http.Request, error) {
	req, err := http.NewRequest(method, getVideoPlatformServerUrl(path), bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Add("authorization", internal.Env.VideoPlatformApiKey)
	return req, nil
}

func getVideoPlatformServerUrl(additionalPath string) string {
	return fmt.Sprintf("%s/api/%s", internal.Env.VideoPlatformServerUrl, additionalPath)
}

type Job struct {
	Id       string `json:"id"`
	Status   string `json:"status"`
	VideoUrl string `json:"videoUrl"`
}

type JobCompletionRequest struct {
	Id             string                 `json:"id"`
	Status         string                 `json:"status"`
	VideoUrl       string                 `json:"videoUrl"`
	SubtitleTracks []openai.SubtitleTrack `json:"subtitleTracks"`
	VideoDuration  float64                `json:"videoDuration"`
}

func (j *JobService) GetJob(jobId string) (*Job, error) {
	req, err := newVideoPlatformApiRequest("GET", "job/"+jobId, nil)
	if err != nil {
		return nil, err
	}

	resp, err := j.http.Do(req)

	if err != nil {
		return nil, err
	}

	if !(resp.StatusCode >= 200 && resp.StatusCode <= 299) {
		return nil, fmt.Errorf("failed to get job: %s", resp.Status)
	}

	job := Job{}

	defer resp.Body.Close()

	if err = json.NewDecoder(resp.Body).Decode(&job); err != nil {
		return nil, err
	}

	return &job, nil
}

func (j *JobService) SendJobCompletionWebhook(job *JobCompletionRequest) error {
	jsonData, err := json.Marshal(job)
	if err != nil {
		return err
	}
	req, err := newVideoPlatformApiRequest("POST", "job", jsonData)

	if err != nil {
		return err
	}

	resp, err := j.http.Do(req)

	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send job completion webhook: %s", resp.Status)
	}

	return nil
}

func (j *JobService) SendJobProcessingStartedWebhook(jobId string) error {
	req, err := newVideoPlatformApiRequest("POST", "job/"+jobId+"/start", nil)

	if err != nil {
		return err
	}

	resp, err := j.http.Do(req)

	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to start job processing: %s", resp.Status)
	}

	return nil
}

func (j *JobService) SendJobProcessingFailedWebhook(jobId string, err error) error {
	jsonData, err := json.Marshal(map[string]string{"error": err.Error()})

	if err != nil {
		return err
	}
	req, err := newVideoPlatformApiRequest("POST", "job/"+jobId+"/fail", jsonData)

	if err != nil {
		return err
	}

	resp, err := j.http.Do(req)

	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send job processing failed webhook: %s", resp.Status)
	}

	return nil
}
