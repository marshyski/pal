package utils

import (
	"encoding/base64"
	"errors"
	"os"
	"os/exec"

	"crypto/rand"

	"github.com/perlogix/pal/data"
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

// HasTarget verify target is not empty
func HasTarget(target string, resource []data.ResourceData) (bool, data.ResourceData) {

	for _, e := range resource {
		if e.Target == target {
			return true, e
		}
	}

	return false, data.ResourceData{}
}

// GetAuthHeader check if auth header is present and return header
func GetAuthHeader(target data.ResourceData) (bool, string) {

	if target.AuthHeader != "" {
		return true, target.AuthHeader
	}

	return false, ""
}

// GetCmd returns cmd if not empty or return error
func GetCmd(target data.ResourceData) (string, error) {

	if target.Cmd != "" {
		return target.Cmd, nil
	}

	return "", errors.New("error cmd is empty for target")

}
