package main

import (
	"bytes"
	"encoding/json"
	"os/exec"
)

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
