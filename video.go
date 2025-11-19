package main

import (
	"bytes"
	"encoding/json"
	"log"
	"math"
	"os/exec"
)

func getVideoAspectRatio(filePath string) (string, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)

	var b bytes.Buffer
	cmd.Stdout = &b

	err := cmd.Run()
	if err != nil {
		log.Fatalf("error executing command %v", err)
	}

	type videoProperties struct {
		Streams []struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"streams"`
	}

	var props videoProperties
	err = json.Unmarshal(b.Bytes(), &props)
	if err != nil {
		return "", err
	}

	width := props.Streams[0].Width
	height := props.Streams[0].Height

	ratio := float64(width) / float64(height)
	epsilon := 0.01

	switch {
	case math.Abs(ratio-16.0/9.0) < epsilon:
		return "16:9", nil
	case math.Abs(ratio-9.0/16.0) < epsilon:
		return "9:16", nil
	default:
		return "other", nil
	}

}
