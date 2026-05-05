package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
)

func getVideoAspectRatio(filePath string) (string, error) {
	log.Printf("Starting Get Video Aspect Ratio func...")
	log.Printf("File Path: %s", filePath)
	// Create command to get aspect ratio
	aspectRatioCmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)

	// Create place to capture info
	cmdResult := bytes.Buffer{}
	cmdError := bytes.Buffer{}

	// Tell cmd to save to buffer
	aspectRatioCmd.Stdout = &cmdResult
	aspectRatioCmd.Stderr = &cmdError

	log.Printf("Buffer created and assigned")

	// Run the command
	err := aspectRatioCmd.Run()
	if err != nil {
		return "", fmt.Errorf("Error running command: %v", cmdError)
	}

	// Create struct for json
	type videoInfo struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	}
	type streams struct {
		Streams []videoInfo `json:"streams"`
	}
	var vidInfo streams

	log.Printf("Raw Data: %v", cmdResult)

	// Unmarshal data
	err = json.Unmarshal(cmdResult.Bytes(), &vidInfo)
	if err != nil {
		return "", fmt.Errorf("Error unmarshalling data: %v", err)
	}
	log.Printf("vidInfo: %v", vidInfo)

	// Determinde ratio
	var ratio string
	ratioNum := float64(vidInfo.Streams[0].Width) / float64(vidInfo.Streams[0].Height)

	if ratioNum <= 1.778 && ratioNum >= 1.777 {
		ratio = "16:9"
	} else if ratioNum >= 0.5625 && ratioNum <= 0.563 {
		ratio = "9:16"
	} else {
		ratio = "other"
	}

	return ratio, nil
}
