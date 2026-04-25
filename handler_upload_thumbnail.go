package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"

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

	mediaType := header.Header.Get("Content-Type")

	imageData, err := io.ReadAll(file)
	if err != nil {
		respondWithError(w, 500, "Couldn't read file", err)
		return
	}

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

	// Make a new thumbnail struct with data
	//newThumbnail := thumbnail{
	//	data:      imageData,
	//	mediaType: mediaType,
	//}

	// Convert the image data to a string
	imageString := base64.StdEncoding.EncodeToString(imageData)

	// Create dataUrl for image
	dataUrl := fmt.Sprintf("data:%s;base64,%s", mediaType, imageString)

	// Make thumbnail url
	//thumbnailUrl := fmt.Sprintf("http://localhost:8091/api/thumbnails/{%v}", videoID)

	// Update thumbnail URL
	dbVideo.ThumbnailURL = &dataUrl

	// Update video in database
	err = cfg.db.UpdateVideo(dbVideo)
	if err != nil {
		respondWithError(w, 500, "Couldn't update video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, dbVideo)
}
