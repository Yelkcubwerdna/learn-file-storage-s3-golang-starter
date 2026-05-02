package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	// Set  upload limit
	r.Body = http.MaxBytesReader(w, r.Body, 1<<30)

	// Extract videoID
	videoIDString := r.PathValue("videoID")

	// Parse into uuid
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, 500, "Couldn't parse uuid", err)
		return
	}

	// Get user's bearer token
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find bearer token", err)
		return
	}

	// Authenticate user and get their ID
	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate user", err)
		return
	}

	// Get the video's metadata
	dbVideo, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, 500, "Couldn't find video", err)
		return
	}

	// Make sure the authenticated user is the same user who owns the video
	if userID != dbVideo.UserID {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized", err)
		return
	}

	// Parse uploaded video file
	file, fileHeader, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, 500, "Couldn't parse video file", err)
		return
	}
	defer file.Close()

	// Validate the files media type
	mediaType, _, err := mime.ParseMediaType(fileHeader.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, 500, "Couldn't parse file type", err)
		return
	}

	// Make sure it's an mp4
	if mediaType != "video/mp4" {
		respondWithError(w, http.StatusNotAcceptable, "File must be mp4.", err)
		return
	}

	// Create a temp file
	tempFile, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		respondWithError(w, 500, "Couldn't create temp file", err)
		return
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Copy file to tempfile
	io.Copy(tempFile, file)

	// Reset tempFile's file pointer to beginning
	_, err = tempFile.Seek(0, io.SeekStart)
	if err != nil {
		respondWithError(w, 500, "Error seeking on tempFile", err)
		return
	}

	// Create file key
	randombytes := make([]byte, 16)
	_, err = rand.Read(randombytes)
	if err != nil {
		respondWithError(w, 500, "Couldn't be random", err)
		return
	}
	fileKey := fmt.Sprintf("%s.%s", hex.EncodeToString(randombytes), "mp4")

	// Put the object into S3
	_, err = cfg.s3Client.PutObject(r.Context(), &s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &fileKey,
		Body:        tempFile,
		ContentType: &mediaType,
	})
	if err != nil {
		respondWithError(w, 500, "Failed to put into S3 Bucket", err)
		return
	}

	// Update VideoURL to S3 URL
	videoURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", cfg.s3Bucket, cfg.s3Region, fileKey)
	dbVideo.VideoURL = &videoURL

	// Update in database
	err = cfg.db.UpdateVideo(dbVideo)
	if err != nil {
		respondWithError(w, 500, "Error updating database", err)
		return
	}

	// Confirmation
	respondWithJSON(w, http.StatusOK, dbVideo)
}
