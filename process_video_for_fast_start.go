package main

import (
	"fmt"
	"os/exec"
)

func processVideoForFastStart(filePath string) (string, error) {
	// Create file path for output
	outputPath := fmt.Sprintf("%s.processing", filePath)

	// Create command
	fastStartCmd := exec.Command("ffmpeg", "-i", filePath, "-c", "copy", "-movflags", "faststart", "-f", "mp4", outputPath)

	// Run the command
	err := fastStartCmd.Run()
	if err != nil {
		return "", fmt.Errorf("Error running fast start command: %s", err)
	}

	// Return output file path
	return outputPath, nil
}
