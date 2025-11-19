package main

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	const maxMemory = 1 << 30
	r.Body = http.MaxBytesReader(w, r.Body, maxMemory)

	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "unable to parse video id", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "video not found", err)
		return
	}

	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "user not authorized", err)
		return
	}

	fmt.Println("uploading video", videoID, "by user", userID)

	r.ParseMultipartForm(maxMemory)

	file, header, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not retrieve file/header", err)
		return
	}
	defer file.Close()

	mediaType := header.Header.Get("Content-Type")
	mType, _, err := mime.ParseMediaType(mediaType)
	if err != nil {
		respondWithError(w, http.StatusForbidden, "could not get media type", err)
		return
	}
	if mType != "video/mp4" {
		respondWithError(w, http.StatusForbidden, "wrong file type", nil)
		return
	}

	tempFile, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error creating temp file", err)
		return
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	_, err = io.Copy(tempFile, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error copying contents to temp file", err)
		return
	}

	// process video faststart

	processedVideoPath, err := processVideoForFastStart(tempFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error processing video faststart", err)
		return
	}

	processedFile, err := os.Open(processedVideoPath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error opening processed file", err)
		return
	}

	defer os.Remove(processedFile.Name())
	defer processedFile.Close()

	// read video parameters
	aspectRatio, err := getVideoAspectRatio(processedFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error getting video parameters", err)
		return
	}

	// set pointer to beginning of file to read it again
	// tempFile.Seek(0, io.SeekStart)

	// key := "getAssetPath(mType)"
	key := ""

	switch aspectRatio {
	case "16:9":
		key = "landscape/" + getAssetPath(mType)
	case "9:16":
		key = "portrait/" + getAssetPath(mType)
	default:
		key = "other/" + getAssetPath(mType)
	}

	_, err = cfg.s3Client.PutObject(r.Context(), &s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &key,
		Body:        processedFile,
		ContentType: &mediaType,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error uploading to s3", err)
		return
	}

	newURL := cfg.getS3URL(key)

	err = cfg.db.UpdateVideo(database.Video{
		ID:           videoID,
		CreatedAt:    video.CreatedAt,
		UpdatedAt:    time.Now().UTC(),
		ThumbnailURL: video.ThumbnailURL,
		VideoURL:     &newURL,
		CreateVideoParams: database.CreateVideoParams{
			Title:       video.CreateVideoParams.Title,
			Description: video.CreateVideoParams.Description,
			UserID:      video.CreateVideoParams.UserID,
		},
	},
	)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error updating video", err)
		return
	}

}
