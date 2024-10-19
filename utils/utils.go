package utils

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"crypto/rand"

	"github.com/marshyski/pal/data"
)

// TimeNow
func TimeNow(tz string) string {
	loc, _ := time.LoadLocation(tz)
	return time.Now().In(loc).Format(time.RFC3339)
}

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
func CmdRun(action data.ActionData, prefix string) (string, int, error) {
	startTime := time.Now()

	if action.Timeout == 0 {
		action.Timeout = 600
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(action.Timeout)*time.Second)
	defer cancel()

	var output []byte
	var err error

	cmdPrefix := append(strings.Split(prefix, " "), action.Cmd)

	// Retry loop
	for attempt := 0; attempt <= action.OnError.Retries; attempt++ {

		command := exec.CommandContext(ctx, cmdPrefix[0], cmdPrefix[1:]...) // #nosec G204
		output, err = command.Output()

		if err == nil {
			break // Command succeeded, exit the loop
		}

		// If it's not a context timeout, retry
		if ctx.Err() != context.DeadlineExceeded {
			if attempt < action.OnError.Retries {
				// TODO: DEBUG LOG
				// fmt.Printf("Command failed, retrying in %d seconds (attempt %d/%d)...\n", action.OnError.RetryInterval, attempt+1, action.OnError.Retries)
				time.Sleep(time.Duration(action.OnError.RetryInterval) * time.Second)
				continue
			}
		}

		errStr := fmt.Sprintf("error after %d retries in %d seconds : %s %s", action.OnError.Retries, int(time.Since(startTime).Seconds()), strings.TrimSpace(string(output)), err.Error())

		// If it's a context timeout or the maximum retries are reached, return the error
		return errStr, int(time.Since(startTime).Seconds()), errors.New(errStr)
	}

	return string(output), int(time.Since(startTime).Seconds()), nil
}

// HasAction verify action is not empty
func HasAction(action string, group []data.ActionData) (bool, data.ActionData) {

	for _, e := range group {
		if e.Action == action {
			return true, e
		}
	}

	return false, data.ActionData{}
}

// GetAuthHeader check if auth header is present and return header
func GetAuthHeader(action data.ActionData) (bool, string) {

	if action.AuthHeader != "" {
		return true, action.AuthHeader
	}

	return false, ""
}

// GetCmd returns cmd if not empty or return error
func GetCmd(action data.ActionData) (string, error) {

	if action.Cmd != "" {
		return action.Cmd, nil
	}

	return "", errors.New("error cmd is empty for action")

}

func MergeGroups(oldGroups, newGroups map[string][]data.ActionData) map[string][]data.ActionData {
	for group, newGroupData := range newGroups {
		if oldGroupData, ok := oldGroups[group]; ok {
			// Group exists in the old map, update its actions
			for _, newAction := range newGroupData {
				found := false
				for i, oldAction := range oldGroupData {
					if newAction.Action == oldAction.Action {
						// Update existing action
						oldGroups[group][i] = updateAction(oldAction, newAction)
						found = true
						break
					}
				}
				if !found {
					// Add new action
					oldGroups[group] = append(oldGroups[group], newAction)
				}
			}
		} else {
			// Group doesn't exist in the old map, add it
			oldGroups[group] = newGroupData
		}
	}

	return oldGroups
}

func updateAction(oldAction, newAction data.ActionData) data.ActionData {
	// Update fields (all except the excluded ones)
	oldAction.Group = newAction.Group
	oldAction.Desc = newAction.Desc
	oldAction.Background = newAction.Background
	oldAction.Concurrent = newAction.Concurrent
	oldAction.AuthHeader = newAction.AuthHeader
	oldAction.Output = newAction.Output
	oldAction.Timeout = newAction.Timeout
	oldAction.Cmd = newAction.Cmd
	oldAction.ResponseHeaders = newAction.ResponseHeaders
	oldAction.Crons = newAction.Crons
	oldAction.OnError = newAction.OnError
	oldAction.InputValidate = newAction.InputValidate
	oldAction.Tags = newAction.Tags

	return oldAction
}
