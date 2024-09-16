package utils

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"

	"crypto/rand"

	"github.com/marshyski/pal/data"
)

// FileExists
func FileExists(location string) bool {
	_, err := os.Stat(location)
	return err == nil
}

// GenSecret
func GenSecret() string {
	randomBytes := make([]byte, 32)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "OGv2P1_3nVs_j9"
	}

	secret := base64.URLEncoding.EncodeToString(randomBytes)[:15]

	return string(secret)
}

// CmdRun runs a shell command or script and returns output with error
func CmdRun(cmd string, timeoutSeconds int) (string, string, error) {
	if timeoutSeconds == 0 {
		// 600 seconds = 10 mins
		timeoutSeconds = 600
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	startTime := time.Now()

	// Create the command with the context
	command := exec.CommandContext(ctx, "/bin/sh", "-c", cmd)

	output, err := command.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", "0s", fmt.Errorf("command timed out after %d seconds", timeoutSeconds)
		}
		return "", "0s", err
	}

	duration := time.Since(startTime)
	durationSeconds := int(duration.Seconds())
	humanReadableDuration := fmt.Sprintf("%ds", durationSeconds)

	return string(output), humanReadableDuration, nil
}

// HasAction verify action is not empty
func HasAction(action string, group []data.GroupData) (bool, data.GroupData) {

	for _, e := range group {
		if e.Action == action {
			return true, e
		}
	}

	return false, data.GroupData{}
}

// GetAuthHeader check if auth header is present and return header
func GetAuthHeader(action data.GroupData) (bool, string) {

	if action.AuthHeader != "" {
		return true, action.AuthHeader
	}

	return false, ""
}

// GetCmd returns cmd if not empty or return error
func GetCmd(action data.GroupData) (string, error) {

	if action.Cmd != "" {
		return action.Cmd, nil
	}

	return "", errors.New("error cmd is empty for action")

}
