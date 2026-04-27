package main

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
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

	const maxMemory = 10 << 20

	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, 500, "Could parse data", err)
		return
	}

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, 500, "Couldn't find thumbnail", err)
		return
	}
	defer file.Close()

	mediaType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, 500, "Couldn't parse media type", err)
		return
	}

	// Make sure it is a jpeg or png
	if mediaType != "image/jpeg" && mediaType != "image/png" {
		respondWithError(w, 500, "Invalid file type. Must be a jpeg or png.", err)
		return
	}

	// Make sure the file is an image

	dbVideo, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, 500, "Couldn't find video", err)
		return
	}

	// Make sure the authenticated user owns this video
	if userID != dbVideo.UserID {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized", err)
		return
	}

	// Strip prefix
	fileExtension, ok := strings.CutPrefix(mediaType, "image/")
	if !ok {
		respondWithError(w, 500, "File doesn't have 'image/' prefix", nil)
		return
	}

	// Create file path
	filePath := filepath.Join(cfg.assetsRoot, fmt.Sprintf("%s.%s", videoIDString, fileExtension))

	// Create new file
	newFile, err := os.Create(filePath)
	if err != nil {
		respondWithError(w, 500, fmt.Sprintf("Couldn't create new file: %v", filePath), err)
		return
	}

	// Copy the multipart.File to the new file
	io.Copy(newFile, file)

	// Create dataUrl for image
	ThumbnailURL := fmt.Sprintf("http://localhost:8091/assets/%s.%s", videoIDString, fileExtension)

	// Update thumbnail URL
	dbVideo.ThumbnailURL = &ThumbnailURL

	// Update video in database
	err = cfg.db.UpdateVideo(dbVideo)
	if err != nil {
		respondWithError(w, 500, "Couldn't update video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, dbVideo)
}
