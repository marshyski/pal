// SPDX-License-Identifier: AGPL-3.0-only
// pal - github.com/marshyski/pal
// Copyright (C) 2024-2025  github.com/marshyski

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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

const (
	randBytes = 32
	day       = 24
	minute    = 60
)

// TimeNow
func TimeNow(tz string) string {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return time.Now().UTC().Format(time.RFC3339)
	}
	return time.Now().In(loc).Format(time.RFC3339)
}

// FileExists
func FileExists(location string) bool {
	_, err := os.Stat(location)
	return err == nil
}

// GenSecret
func GenSecret() string {
	randomBytes := make([]byte, randBytes)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "OGv2P1_3nVs_j9"
	}

	secret := base64.URLEncoding.EncodeToString(randomBytes)[:15]

	return secret
}

// CmdRun runs a shell command or script and returns output with error
func CmdRun(action data.ActionData, prefix, workingDir string) (string, string, error) {
	startTime := time.Now()

	if action.Timeout == 0 {
		action.Timeout = 600
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(action.Timeout)*time.Second)
	defer cancel()

	var output []byte
	var err error

	var cmdPrefix []string
	if action.CmdPrefix != "" {
		cmdPrefix = append(strings.Split(action.CmdPrefix, " "), action.Cmd)
	} else {
		cmdPrefix = append(strings.Split(prefix, " "), action.Cmd)
	}

	// Retry loop
	for attempt := 0; attempt <= action.OnError.Retries; attempt++ {
		command := exec.CommandContext(ctx, cmdPrefix[0], cmdPrefix[1:]...) // #nosec G204
		command.Dir = workingDir
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
		return errStr, fmtDuration(int(time.Since(startTime).Seconds())), errors.New(errStr)
	}

	return strings.TrimSpace(string(output)), fmtDuration(int(time.Since(startTime).Seconds())), nil
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

	// Remove old groups/actions that don't exist in newGroups
	for group, oldGroupData := range oldGroups {
		if newGroupData, ok := newGroups[group]; ok {
			// Group exists in newGroups, check actions
			var updatedActions []data.ActionData
			for _, oldAction := range oldGroupData {
				found := false
				for _, newAction := range newGroupData {
					if oldAction.Action == newAction.Action {
						updatedActions = append(updatedActions, oldAction)
						found = true
						break
					}
				}
				if !found {
					// Action not found in newGroupData delete it
					// Create a new slice excluding the oldAction
					var newOldGroupData []data.ActionData
					for _, oa := range oldGroupData {
						if oa.Action != oldAction.Action {
							newOldGroupData = append(newOldGroupData, oa)
						}
					}
					oldGroups[group] = newOldGroupData // Update with the new slice
				}
			}
			oldGroups[group] = updatedActions
		} else {
			// Group doesn't exist in newGroups, remove it
			delete(oldGroups, group)
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
	oldAction.Container = newAction.Container
	oldAction.Cmd = newAction.Cmd
	oldAction.ResponseHeaders = newAction.ResponseHeaders
	oldAction.Crons = newAction.Crons
	oldAction.OnError = newAction.OnError
	oldAction.OnSuccess = newAction.OnSuccess
	oldAction.Input = newAction.Input
	oldAction.InputValidate = newAction.InputValidate
	oldAction.Triggers = newAction.Triggers

	return oldAction
}

func GetLastOutput(action data.ActionData) string {
	if action.Status == "success" {
		return action.LastSuccessOutput
	}
	return action.LastFailureOutput
}

func fmtDuration(seconds int) string {
	duration := time.Duration(seconds) * time.Second
	days := int(duration.Hours() / day)
	hours := int(duration.Hours()) % day
	minutes := int(duration.Minutes()) % minute
	seconds = int(duration.Seconds()) % minute

	var parts []string
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
	}
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if minutes > 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes))
	}
	if seconds > 0 {
		parts = append(parts, fmt.Sprintf("%ds", seconds))
	} else {
		parts = append(parts, "0s")
	}

	return strings.Join(parts, "")
}
