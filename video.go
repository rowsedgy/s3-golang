package main

import (
	"bytes"
	"encoding/json"
	"fmt"
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

func processVideoForFastStart(filePath string) (string, error) {
	outPath := filePath + ".processing"
	fmt.Println(outPath)
	fmt.Println(filePath)

	cmd := exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", outPath)
	err := cmd.Run()
	if err != nil {
		log.Fatalf("error running command %v", err)
	}

	return outPath, nil
}

// func generatePresignedURL(s3Client *s3.Client, bucket, key string, expireTime time.Duration) (string, error) {
// 	presignClient := s3.NewPresignClient(s3Client)
// 	presignedUrl, err := presignClient.PresignGetObject(context.Background(), &s3.GetObjectInput{
// 		Bucket: &bucket,
// 		Key:    &key,
// 	},
// 		s3.WithPresignExpires(expireTime))

// 	if err != nil {
// 		return "", err
// 	}

// 	return presignedUrl.URL, nil
// }

// func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {
// 	if video.VideoURL == nil {
// 		return video, nil
// 	}
// 	params := strings.Split(*video.VideoURL, ",")
// 	bucket := params[0]
// 	key := params[1]

// 	presignedUrl, err := generatePresignedURL(cfg.s3Client, bucket, key, time.Minute*2)
// 	if err != nil {
// 		return database.Video{}, err
// 	}

// 	video.VideoURL = &presignedUrl

// 	return video, nil
// }
