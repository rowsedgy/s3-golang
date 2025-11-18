package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	const maxMemory = 10 << 20

	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
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

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	r.ParseMultipartForm(maxMemory)

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not retrieve file/header", err)
		return
	}
	defer file.Close()

	mediaType := header.Header.Get("Content-Type")

	mtype, _, err := mime.ParseMediaType(mediaType)
	if err != nil {
		respondWithError(w, http.StatusForbidden, "could not get media type", err)
		return
	}
	if mtype != "image/jpeg" && mtype != "image/png" {
		respondWithError(w, http.StatusForbidden, "invalid media type", err)
		return
	}

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "video not found", err)
		return
	}

	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "you're not the owner of that video", err)
		return
	}

	// create file
	key := make([]byte, 32)
	rand.Read(key)
	encoded := base64.RawURLEncoding.EncodeToString(key)
	assetPath := getAssetPath(encoded, mediaType)
	filePath := cfg.getAssetDiskPath(assetPath)
	fileURL := cfg.getAssetURL(assetPath)

	f, err := os.Create(filePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not create file", err)
		return
	}
	defer f.Close()

	_, err = io.Copy(f, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "could not copy content to file", err)
		return
	}

	// turl := fmt.Sprintf("http://localhost:%s/api/thumbnails/%s", cfg.port, videoID)

	turl := fileURL

	video.ThumbnailURL = &turl

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "error updating video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
