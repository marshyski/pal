package utils

import (
	"encoding/base64"
	"errors"
	"os"
	"os/exec"

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
func CmdRun(cmd string) (string, error) {
	output, err := exec.Command("/bin/sh", "-c", cmd).Output()
	if err != nil {
		return "", err
	}

	return string(output), nil
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
