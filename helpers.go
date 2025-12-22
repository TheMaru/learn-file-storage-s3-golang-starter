package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
)

const PresignedURLDuration = 10 * time.Minute

func getVideoAspectRatio(filePath string) (string, error) {
	execCmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)
	var b bytes.Buffer
	execCmd.Stdout = &b

	err := execCmd.Run()
	if err != nil {
		return "", err
	}

	type ffProbeResponse struct {
		Streams []struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"streams"`
	}

	var resp ffProbeResponse
	if err = json.Unmarshal(b.Bytes(), &resp); err != nil {
		return "", err
	}

	width := resp.Streams[0].Width
	height := resp.Streams[0].Height
	ratio := float64(width) / float64(height)

	ratio16by9 := 16.0 / 9.0
	ratio9by16 := 9.0 / 16.0

	if approxEqual(ratio, ratio16by9, 0.05) {
		return "16:9", nil
	} else if approxEqual(ratio, ratio9by16, 0.05) {
		return "9:16", nil
	}
	return "other", nil
}

func approxEqual(a, b, tol float64) bool {
	if a > b {
		return (a - b) <= tol
	}
	return (b - a) <= tol
}

func processVideoForFastStart(filePath string) (string, error) {
	processingPath := filePath + ".processing"
	execCmd := exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", processingPath)
	err := execCmd.Run()
	if err != nil {
		return "", err
	}

	return processingPath, nil
}

func generatePresignedURL(s3Client *s3.Client, bucket, key string, expireTime time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(s3Client)
	presignedHTTPRequest, err := presignClient.PresignGetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expireTime))
	if err != nil {
		return "", err
	}
	return presignedHTTPRequest.URL, nil
}

func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {
	if video.VideoURL == nil {
		return video, nil
	}
	parts := strings.Split(*video.VideoURL, ",")
	if len(parts) != 2 {
		return database.Video{}, errors.New("Video URL not correctly formatted!")
	}
	bucket := parts[0]
	key := parts[1]

	url, err := generatePresignedURL(cfg.s3Client, bucket, key, 1*time.Minute)
	if err != nil {
		return database.Video{}, err
	}
	video.VideoURL = &url
	return video, nil
}
