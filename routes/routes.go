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

package routes

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/go-co-op/gocron/v2"
	"github.com/go-playground/validator/v10"
	"github.com/goccy/go-json"
	"github.com/google/uuid"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	echo "github.com/labstack/echo/v4"
	"github.com/lnquy/cron"
	"github.com/marshyski/pal/config"
	"github.com/marshyski/pal/data"
	"github.com/marshyski/pal/db"
	"github.com/marshyski/pal/ui"
	"github.com/marshyski/pal/utils"
	"gopkg.in/yaml.v3"
)

const (
	httpStatusCodeCheck = 300
	httpClientTimeout   = 15
	runHistoryLimit     = 5
	errorAuth           = "error unauthorized"
	errorScript         = "error script fail"
	errorNotReady       = "error not ready"
	errorAction         = "error invalid action"
	errorGroup          = "error group invalid"
	errorCmdEmpty       = "error cmd is empty for action"
	favicon             = `<?xml version="1.0" encoding="UTF-8"?>
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="48" height="48">
<path d="M0 0 C15.84 0 31.68 0 48 0 C48 15.84 48 31.68 48 48 C32.16 48 16.32 48 0 48 C0 32.16 0 16.32 0 0 Z " fill="#060606" transform="translate(0,0)"/>
<path d="M0 0 C8.91 0 17.82 0 27 0 C27 1.32 27 2.64 27 4 C29.475 4.495 29.475 4.495 32 5 C32 7.97 32 10.94 32 14 C30.68 14 29.36 14 28 14 C27.67 15.65 27.34 17.3 27 19 C18.09 19 9.18 19 0 19 C0 12.73 0 6.46 0 0 Z " fill="#292929" transform="translate(8,7)"/>
<path d="M0 0 C8.91 0 17.82 0 27 0 C27 1.32 27 2.64 27 4 C18.09 4 9.18 4 0 4 C0 2.68 0 1.36 0 0 Z " fill="#E7E7E7" transform="translate(8,22)"/>
<path d="M0 0 C8.91 0 17.82 0 27 0 C27 0.99 27 1.98 27 3 C18.09 3 9.18 3 0 3 C0 2.01 0 1.02 0 0 Z " fill="#FCFCFC" transform="translate(8,7)"/>
<path d="M0 0 C3.63 0 7.26 0 11 0 C11.33 1.32 11.66 2.64 12 4 C8.04 4 4.08 4 0 4 C0 2.68 0 1.36 0 0 Z " fill="#E3E3E3" transform="translate(8,27)"/>
<path d="M0 0 C3.3 0 6.6 0 10 0 C10.33 1.32 10.66 2.64 11 4 C7.04 4 3.08 4 -1 4 C-0.67 2.68 -0.34 1.36 0 0 Z " fill="#E7E7E7" transform="translate(9,32)"/>
<path d="M0 0 C3.63 0 7.26 0 11 0 C11 1.32 11 2.64 11 4 C7.37 4 3.74 4 0 4 C0 2.68 0 1.36 0 0 Z " fill="#E8E8E8" transform="translate(29,17)"/>
<path d="M0 0 C3.63 0 7.26 0 11 0 C11 1.32 11 2.64 11 4 C7.37 4 3.74 4 0 4 C0 2.68 0 1.36 0 0 Z " fill="#E8E8E8" transform="translate(8,17)"/>
<path d="M0 0 C3.96 0 7.92 0 12 0 C11.67 1.32 11.34 2.64 11 4 C7.7 4 4.4 4 1 4 C0.67 2.68 0.34 1.36 0 0 Z " fill="#E9E9E9" transform="translate(28,12)"/>
<path d="M0 0 C3.96 0 7.92 0 12 0 C11.67 1.32 11.34 2.64 11 4 C7.7 4 4.4 4 1 4 C0.67 2.68 0.34 1.36 0 0 Z " fill="#E9E9E9" transform="translate(8,12)"/>
<path d="M0 0 C3.96 0 7.92 0 12 0 C12 0.99 12 1.98 12 3 C8.04 3 4.08 3 0 3 C0 2.01 0 1.02 0 0 Z " fill="#F3F3F3" transform="translate(8,38)"/>
</svg>`
)

var (
	sched    gocron.Scheduler
	validate = validator.New(validator.WithRequiredStructEnabled())
)

func checkBasicAuth(c echo.Context) bool {
	username, password, ok := c.Request().BasicAuth()
	validUser := strings.Split(config.GetConfigUI().BasicAuth, " ")[0]
	validPass := strings.Split(config.GetConfigUI().BasicAuth, " ")[1]
	if !ok {
		return false
	}

	return username == validUser && password == validPass
}

// lock sets the lock for blocking requests until cmd has finished
func lock(group, action string, lockState bool) {
	resData := db.DBC.GetGroupAction(group, action)

	resData.Lock = lockState
	db.DBC.PutGroupAction(group, resData)
}

func condDisable(group, action string, disabled bool) {
	resData := db.DBC.GetGroupAction(group, action)

	resData.Disabled = disabled
	db.DBC.PutGroupAction(group, resData)
	if disabled {
		sched.RemoveByTags(group + action)
	} else {
		if len(resData.Crons) > 0 {
			for _, c := range resData.Crons {
				if validateInput(c, "cron") == nil {
					var cronDesc string
					exprDesc, err := cron.NewDescriptor()
					if err == nil {
						cronDesc, err = exprDesc.ToDescription(c, cron.Locale_en)
						if err != nil {
							cronDesc = ""
						}
					}

					_, err = sched.NewJob(
						gocron.CronJob(c, false),
						gocron.NewTask(cronTask, resData),
						gocron.WithName(group+"/"+action),
						gocron.WithTags(cronDesc, group+action),
					)

					if err != nil {
						// TODOD: log error
						return
					}
				}
			}
		}
		return
	}
}

// logError time, error, id, uri fields
func logError(reqid, uri string, e error) {
	fmt.Fprintf(os.Stdout, "%s\n", fmt.Sprintf(`{"time":"%s","error":"%s","id":"%s","uri":"%s"}`,
		utils.TimeNow(config.GetConfigStr("global_timezone")), e.Error(), reqid, uri))
}

// RunGroup is the main route for triggering a command
//
//nolint:gocyclo // TODO: Clean up high complexity
func RunGroup(c echo.Context) error {
	// check if group from URL is not empty
	group := c.Param("group")
	if group == "" {
		return c.String(http.StatusBadRequest, errorGroup)
	}

	resData := db.DBC.GetGroupActions(group)

	action := c.Param("action")
	if action == "" {
		return c.String(http.StatusBadRequest, errorAction)
	}

	actionPresent, actionData := utils.HasAction(action, resData)

	if !actionPresent {
		return c.String(http.StatusBadRequest, errorAction)
	}

	// set global http resp headers
	if len(config.GetConfigResponseHeaders()) > 0 {
		for _, v := range config.GetConfigResponseHeaders() {
			if strings.ToLower(v.Header) != "access-control-allow-origin" {
				c.Response().Header().Set(v.Header, v.Value)
			}
		}
	}

	// set action http resp headers
	if len(actionData.ResponseHeaders) > 0 {
		for _, v := range actionData.ResponseHeaders {
			if strings.ToLower(v.Header) != "access-control-allow-origin" {
				c.Response().Header().Set(v.Header, v.Value)
			}
		}
	}

	auth, authHeader := utils.GetAuthHeader(actionData)

	// Check if auth header is present and if the header is correct
	if auth {
		pass := false
		if strings.HasPrefix(c.Request().RequestURI, "/v1/pal/ui") {
			if !sessionValid(c) && !checkBasicAuth(c) {
				return c.Redirect(http.StatusSeeOther, "/v1/pal/ui/login")
			}
			pass = true
		}
		for k, v := range c.Request().Header {
			header := strings.Join([]string{k, v[0]}, " ")
			if header == authHeader {
				pass = true
			}
		}

		if !pass {
			return c.String(http.StatusUnauthorized, errorAuth)
		}
	}

	// Return last output don't rerun or count as a "run"
	if c.QueryParam("last_output") == "true" {
		if actionData.Output {
			lastOutput := utils.GetLastOutput(actionData)
			return c.String(http.StatusOK, lastOutput)
		}
		return c.String(http.StatusBadRequest, "error output not enabled")
	}

	if c.QueryParam("last_success") == "true" {
		if actionData.Output {
			return c.String(http.StatusOK, actionData.LastSuccessOutput)
		}
		return c.String(http.StatusBadRequest, "error output not enabled")
	}

	if c.QueryParam("last_failure") == "true" {
		if actionData.Output {
			return c.String(http.StatusOK, actionData.LastFailureOutput)
		}
		return c.String(http.StatusBadRequest, "error output not enabled")
	}

	if actionData.Disabled {
		return c.String(http.StatusBadRequest, "error action is disabled")
	}

	cmdOrig := actionData.Cmd

	var input string

	if c.Request().Method == http.MethodPost {
		bodyBytes, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "error reading request body in post run")
		}
		input = string(bodyBytes)
	} else {
		input = c.QueryParam("input")
	}

	if input == "" {
		input = actionData.Input
	}

	input = strings.TrimSpace(input)

	err := validateInput(input, actionData.InputValidate)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "error with input validation: "+err.Error())
	}

	req, err := requestJSON(c, input)
	if err != nil {
		req = ""
	}

	actionData.Cmd = cmdString(actionData, input, req)

	// Check if action wants to block the request to one
	if !actionData.Concurrent {
		if actionData.Lock {
			return c.String(http.StatusTooManyRequests, errorNotReady)
		}

		lock(group, action, true)
	}

	if actionData.Background {
		go func() {
			var (
				cmdOutput string
				duration  string
				err       error
			)
			if actionData.Image != "" {
				cmdOutput, duration, err = utils.CmdRunContainerized(actionData, config.GetConfigStr("global_podman_socket"), config.GetConfigStr("global_cmd_prefix"), config.GetConfigStr("global_working_dir"))
			} else {
				cmdOutput, duration, err = utils.CmdRun(actionData, config.GetConfigStr("global_cmd_prefix"), config.GetConfigStr("global_working_dir"))
			}

			actionData.Cmd = cmdOrig
			if err != nil {
				if !actionData.Concurrent {
					lock(group, action, false)
				}
				actionData.Status = "error"
				actionData.RunCount++
				actionData.LastDuration = duration
				actionData.LastRan = utils.TimeNow(config.GetConfigStr("global_timezone"))
				actionData.LastFailure = actionData.LastRan
				if actionData.Output {
					actionData.LastFailureOutput = err.Error() + " " + cmdOutput
				}
				mergeGroup(actionData)
				registerActionDB(actionData, actionData.LastFailureOutput, input)
				logError("", "", err)
				if actionData.OnError.Notification != "" {
					notification := actionData.OnError.Notification
					notification = strings.ReplaceAll(notification, "$PAL_GROUP", actionData.Group)
					notification = strings.ReplaceAll(notification, "$PAL_ACTION", actionData.Action)
					notification = strings.ReplaceAll(notification, "$PAL_INPUT", input)
					notification = strings.ReplaceAll(notification, "$PAL_STATUS", actionData.Status)
					if actionData.Output {
						notification = strings.ReplaceAll(notification, "$PAL_OUTPUT", actionData.LastFailureOutput)
					}
					err := putNotifications(data.Notification{Group: group, Action: action, Status: actionData.Status, Notification: notification})
					if err != nil {
						logError("", "", err)
					}
				}
				sendWebhookNotifications(actionData, actionData.LastFailureOutput, input)
				for _, e := range actionData.OnError.Run {
					go func() {
						errorGroup := e.Group
						errorAction := e.Action
						errorInput := e.Input
						errorInput = strings.ReplaceAll(errorInput, "$PAL_GROUP", actionData.Group)
						errorInput = strings.ReplaceAll(errorInput, "$PAL_ACTION", actionData.Action)
						if actionData.Output {
							errorInput = strings.ReplaceAll(errorInput, "$PAL_OUTPUT", actionData.LastFailureOutput)
						}
						errorInput = strings.ReplaceAll(errorInput, "$PAL_STATUS", actionData.Status)

						runBackground(errorGroup, errorAction, errorInput)
					}()
				}
				return
			}
			actionData.Status = "success"
			actionData.RunCount++
			actionData.LastDuration = duration
			actionData.LastRan = utils.TimeNow(config.GetConfigStr("global_timezone"))
			actionData.LastSuccess = actionData.LastRan
			if actionData.Output {
				actionData.LastSuccessOutput = cmdOutput
			}
			mergeGroup(actionData)
			registerActionDB(actionData, actionData.LastSuccessOutput, input)
			if actionData.OnSuccess.Notification != "" {
				notification := actionData.OnSuccess.Notification
				notification = strings.ReplaceAll(notification, "$PAL_GROUP", actionData.Group)
				notification = strings.ReplaceAll(notification, "$PAL_ACTION", actionData.Action)
				notification = strings.ReplaceAll(notification, "$PAL_INPUT", input)
				notification = strings.ReplaceAll(notification, "$PAL_STATUS", actionData.Status)
				if actionData.Output {
					notification = strings.ReplaceAll(notification, "$PAL_OUTPUT", cmdOutput)
				}
				err := putNotifications(data.Notification{Group: group, Action: action, Status: actionData.Status, Notification: notification})
				if err != nil {
					logError("", "", err)
				}
			}
			sendWebhookNotifications(actionData, actionData.LastSuccessOutput, input)
			for _, e := range actionData.OnSuccess.Run {
				go func() {
					successGroup := e.Group
					successAction := e.Action
					successInput := e.Input
					successInput = strings.ReplaceAll(successInput, "$PAL_GROUP", actionData.Group)
					successInput = strings.ReplaceAll(successInput, "$PAL_ACTION", actionData.Action)
					successInput = strings.ReplaceAll(successInput, "$PAL_STATUS", actionData.Status)
					if actionData.Output {
						successInput = strings.ReplaceAll(successInput, "$PAL_OUTPUT", cmdOutput)
					}
					runBackground(successGroup, successAction, successInput)
				}()
			}
		}()

		if !actionData.Concurrent {
			lock(group, action, false)
		}

		return c.String(http.StatusOK, "running in background")
	}

	var (
		cmdOutput string
		duration  string
	)
	if actionData.Image != "" {
		cmdOutput, duration, err = utils.CmdRunContainerized(actionData, config.GetConfigStr("global_podman_socket"), config.GetConfigStr("global_cmd_prefix"), config.GetConfigStr("global_working_dir"))
	} else {
		cmdOutput, duration, err = utils.CmdRun(actionData, config.GetConfigStr("global_cmd_prefix"), config.GetConfigStr("global_working_dir"))
	}
	actionData.Cmd = cmdOrig
	if err != nil {
		if !actionData.Concurrent {
			lock(group, action, false)
		}
		actionData.Status = "error"
		actionData.RunCount++
		actionData.LastDuration = duration
		actionData.LastRan = utils.TimeNow(config.GetConfigStr("global_timezone"))
		actionData.LastFailure = actionData.LastRan
		if actionData.Output {
			actionData.LastFailureOutput = err.Error() + " " + cmdOutput
		}
		mergeGroup(actionData)
		registerActionDB(actionData, actionData.LastFailureOutput, input)
		logError(c.Response().Header().Get(echo.HeaderXRequestID), c.Request().RequestURI, errors.New(errorScript+" "+err.Error()))
		if actionData.OnError.Notification != "" {
			notification := actionData.OnError.Notification
			notification = strings.ReplaceAll(notification, "$PAL_GROUP", actionData.Group)
			notification = strings.ReplaceAll(notification, "$PAL_ACTION", actionData.Action)
			notification = strings.ReplaceAll(notification, "$PAL_INPUT", input)
			notification = strings.ReplaceAll(notification, "$PAL_STATUS", actionData.Status)
			if actionData.Output {
				notification = strings.ReplaceAll(notification, "$PAL_OUTPUT", actionData.LastFailureOutput)
			}
			err := putNotifications(data.Notification{Group: group, Action: action, Status: actionData.Status, Notification: notification})
			if err != nil {
				logError(c.Response().Header().Get(echo.HeaderXRequestID), c.Request().RequestURI, err)
			}
		}
		sendWebhookNotifications(actionData, actionData.LastFailureOutput, input)
		for _, e := range actionData.OnError.Run {
			go func() {
				errorGroup := e.Group
				errorAction := e.Action
				errorInput := e.Input
				errorInput = strings.ReplaceAll(errorInput, "$PAL_GROUP", actionData.Group)
				errorInput = strings.ReplaceAll(errorInput, "$PAL_ACTION", actionData.Action)
				errorInput = strings.ReplaceAll(errorInput, "$PAL_STATUS", actionData.Status)
				if actionData.Output {
					errorInput = strings.ReplaceAll(errorInput, "$PAL_OUTPUT", actionData.LastFailureOutput)
				}
				runBackground(errorGroup, errorAction, errorInput)
			}()
		}
		return c.String(http.StatusInternalServerError, err.Error())
	}

	// Unlock if blocking is enabled
	if !actionData.Concurrent {
		lock(group, action, false)
	}

	if actionData.Output {
		actionData.Status = "success"
		actionData.RunCount++
		actionData.LastDuration = duration
		actionData.LastRan = utils.TimeNow(config.GetConfigStr("global_timezone"))
		actionData.LastSuccess = actionData.LastRan
		if actionData.Output {
			actionData.LastSuccessOutput = cmdOutput
		}
		mergeGroup(actionData)
		registerActionDB(actionData, actionData.LastSuccessOutput, input)
		if actionData.OnSuccess.Notification != "" {
			notification := actionData.OnSuccess.Notification
			notification = strings.ReplaceAll(notification, "$PAL_GROUP", actionData.Group)
			notification = strings.ReplaceAll(notification, "$PAL_ACTION", actionData.Action)
			notification = strings.ReplaceAll(notification, "$PAL_INPUT", input)
			notification = strings.ReplaceAll(notification, "$PAL_STATUS", actionData.Status)
			if actionData.Output {
				notification = strings.ReplaceAll(notification, "$PAL_OUTPUT", cmdOutput)
			}
			err := putNotifications(data.Notification{Group: group, Action: action, Status: actionData.Status, Notification: notification})
			if err != nil {
				logError("", "", err)
			}
		}
		sendWebhookNotifications(actionData, actionData.LastSuccessOutput, input)
		for _, e := range actionData.OnSuccess.Run {
			go func() {
				successGroup := e.Group
				successAction := e.Action
				successInput := e.Input
				successInput = strings.ReplaceAll(successInput, "$PAL_GROUP", actionData.Group)
				successInput = strings.ReplaceAll(successInput, "$PAL_ACTION", actionData.Action)
				successInput = strings.ReplaceAll(successInput, "$PAL_STATUS", actionData.Status)
				if actionData.Output {
					successInput = strings.ReplaceAll(successInput, "$PAL_OUTPUT", cmdOutput)
				}
				runBackground(successGroup, successAction, successInput)
			}()
		}
		return c.String(http.StatusOK, cmdOutput)
	}

	actionData.Status = "success"
	actionData.RunCount++
	actionData.LastDuration = duration
	actionData.LastRan = utils.TimeNow(config.GetConfigStr("global_timezone"))
	actionData.LastSuccess = actionData.LastRan
	if actionData.Output {
		actionData.LastSuccessOutput = cmdOutput
	}
	mergeGroup(actionData)
	registerActionDB(actionData, actionData.LastSuccessOutput, input)
	return c.String(http.StatusOK, "done")
}

func GetHealth(c echo.Context) error {
	return c.String(http.StatusOK, "ok")
}

func GetCond(c echo.Context) error {
	if !sessionValid(c) && !checkBasicAuth(c) {
		return c.JSON(http.StatusUnauthorized, data.GenericResponse{Err: "Unauthorized no valid session or basic auth."})
	}

	disable := c.QueryParam("disable")

	state := disable == "true"

	group := c.Param("group")
	action := c.Param("action")

	condDisable(group, action, state)

	return c.Redirect(http.StatusSeeOther, "/v1/pal/ui")
}

func GetNotifications(c echo.Context) error {
	if !sessionValid(c) && !checkBasicAuth(c) {
		return c.JSON(http.StatusUnauthorized, data.GenericResponse{Err: "Unauthorized no valid session or basic auth."})
	}

	return c.JSON(http.StatusOK, db.DBC.GetNotifications(c.QueryParam("group")))
}

func PutNotifications(c echo.Context) error {
	if !sessionValid(c) && !checkBasicAuth(c) {
		return c.JSON(http.StatusUnauthorized, data.GenericResponse{Err: "Unauthorized no valid session or basic auth."})
	}

	notification := new(data.Notification)
	if err := c.Bind(notification); err != nil {
		return c.JSON(http.StatusInternalServerError, data.GenericResponse{Err: err.Error()})
	}

	if err := c.Validate(notification); err != nil {
		return c.JSON(http.StatusBadRequest, data.GenericResponse{Err: err.Error()})
	}

	err := putNotifications(*notification)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, data.GenericResponse{Err: err.Error()})
	}

	return c.JSON(http.StatusOK, data.GenericResponse{Message: "Created notification"})
}

func GetDeleteNotifications(c echo.Context) error {
	if !sessionValid(c) && !checkBasicAuth(c) {
		return c.JSON(http.StatusUnauthorized, data.GenericResponse{Err: "Unauthorized no valid session or basic auth."})
	}

	err := db.DBC.DeleteNotifications()
	if err != nil {
		return err
	}

	return c.Redirect(http.StatusSeeOther, "/v1/pal/ui/notifications")
}

func GetNotificationsPage(c echo.Context) error {
	if !sessionValid(c) && !checkBasicAuth(c) {
		return c.Redirect(http.StatusSeeOther, "/v1/pal/ui/login")
	}

	notificationID := c.QueryParam("notification_received")
	var notifications []data.Notification

	// delete notification from slice
	if notificationID != "" {
		for _, e := range db.DBC.GetNotifications("") {
			if e.ID != notificationID {
				notifications = append(notifications, e)
			}
		}

		err := db.DBC.PutNotifications(notifications)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, data.GenericResponse{Err: err.Error()})
		}
		return c.Redirect(http.StatusSeeOther, "/v1/pal/ui/notifications")
	}

	for _, e := range db.DBC.GetNotifications("") {
		parsedTime, err := time.Parse(time.RFC3339, e.NotificationRcv)
		if err == nil {
			e.NotificationRcv = humanize.Time(parsedTime)
		}
		notifications = append(notifications, e)
	}

	uiData := struct {
		NotificationsList []data.Notification
		Notifications     int
	}{
		NotificationsList: notifications,
		Notifications:     len(db.DBC.GetNotifications("")),
	}

	return c.Render(http.StatusOK, "notifications.tmpl", uiData)
}

func GetCronsJSON(c echo.Context) error {
	if !sessionValid(c) && !checkBasicAuth(c) {
		return c.JSON(http.StatusUnauthorized, data.GenericResponse{Err: "Unauthorized no valid session or basic auth."})
	}

	group := c.QueryParam("group")

	action := c.QueryParam("action")

	name := group + "/" + action

	scheds := []data.Crons{}

	for _, e := range sched.Jobs() {
		if name == e.Name() && c.QueryParam("run") == "now" {
			err := e.RunNow()
			if err != nil {
				return c.JSON(http.StatusInternalServerError, data.GenericResponse{Err: err.Error()})
			}
			return c.JSON(http.StatusOK, data.GenericResponse{Message: "running"})
		}

		nextrun, _ := e.NextRun()
		group := strings.Split(e.Name(), "/")[0]
		action := strings.Split(e.Name(), "/")[1]
		actionData := db.DBC.GetGroupAction(group, action)
		lastRan, _ := time.Parse(time.RFC3339, actionData.LastRan)

		scheds = append(scheds, data.Crons{
			Group:        group,
			Action:       action,
			NextRun:      nextrun,
			LastRan:      lastRan,
			LastDuration: actionData.LastDuration,
			Status:       actionData.Status,
		})
	}

	return c.JSON(http.StatusOK, scheds)
}

func GetCrons(c echo.Context) error {
	if !sessionValid(c) && !checkBasicAuth(c) {
		return c.Redirect(http.StatusSeeOther, "/v1/pal/ui/login")
	}

	type crons struct {
		RunHistory   []data.RunHistory
		CronDesc     string
		Group        string
		Action       string
		NextRun      string
		LastRan      string
		LastDuration string
	}

	scheds := []crons{}

	for _, e := range sched.Jobs() {
		nextrun, _ := e.NextRun()
		group := strings.Split(e.Name(), "/")[0]
		action := strings.Split(e.Name(), "/")[1]
		actionData := db.DBC.GetGroupAction(group, action)
		parsedTime, err := time.Parse(time.RFC3339, actionData.LastRan)
		if err == nil {
			actionData.LastRan = humanize.Time(parsedTime)
		}

		for runIndex, run := range actionData.RunHistory {
			parsedTime, err = time.Parse(time.RFC3339, run.Ran)
			if err == nil {
				actionData.RunHistory[runIndex].Ran = humanize.Time(parsedTime)
			}
		}

		scheds = append(scheds, crons{
			Group:        strings.Split(e.Name(), "/")[0],
			Action:       strings.Split(e.Name(), "/")[1],
			NextRun:      humanize.Time(nextrun),
			RunHistory:   actionData.RunHistory,
			LastRan:      actionData.LastRan,
			LastDuration: actionData.LastDuration,
			CronDesc:     e.Tags()[0],
		})
	}

	uiData := struct {
		Schedules     []crons
		Notifications int
	}{
		Schedules:     scheds,
		Notifications: len(db.DBC.GetNotifications("")),
	}

	return c.Render(http.StatusOK, "crons.tmpl", uiData)
}

func GetDBGet(c echo.Context) error {
	if !sessionValid(c) && !checkBasicAuth(c) {
		return c.String(http.StatusUnauthorized, errorAuth)
	}

	key := c.QueryParam("key")

	if key == "" {
		return echo.NewHTTPError(http.StatusNotFound, "error key query param empty")
	}

	if len(config.GetConfigResponseHeaders()) > 0 {
		for _, v := range config.GetConfigResponseHeaders() {
			if strings.ToLower(v.Header) != "access-control-allow-origin" {
				c.Response().Header().Set(v.Header, v.Value)
			}
		}
	}

	dbSet, err := db.DBC.Get(key)
	if err != nil {
		return c.String(http.StatusNotFound, "error value not found with key: "+key)
	}

	return c.String(http.StatusOK, dbSet.Value)
}

func GetDBJSONDump(c echo.Context) error {
	if !sessionValid(c) && !checkBasicAuth(c) {
		return c.String(http.StatusUnauthorized, errorAuth)
	}

	if len(config.GetConfigResponseHeaders()) > 0 {
		for _, v := range config.GetConfigResponseHeaders() {
			if strings.ToLower(v.Header) != "access-control-allow-origin" {
				c.Response().Header().Set(v.Header, v.Value)
			}
		}
	}

	return c.JSON(http.StatusOK, db.DBC.Dump())
}

func PutDBPut(c echo.Context) error {
	if !sessionValid(c) && !checkBasicAuth(c) {
		return c.String(http.StatusUnauthorized, errorAuth)
	}

	key := c.QueryParam("key")
	var secret bool
	if c.QueryParam("secret") == "true" {
		secret = true
	}

	if key == "" {
		return echo.NewHTTPError(http.StatusNotFound, "error key query param empty")
	}

	if len(config.GetConfigResponseHeaders()) > 0 {
		for _, v := range config.GetConfigResponseHeaders() {
			if strings.ToLower(v.Header) != "access-control-allow-origin" {
				c.Response().Header().Set(v.Header, v.Value)
			}
		}
	}

	bodyBytes, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "error reading request body in db put")
	}

	dbSet := data.DBSet{
		Key:    key,
		Value:  string(bodyBytes),
		Secret: secret,
	}
	err = db.DBC.Put(dbSet)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "error db put for key: "+key)
	}

	return c.String(http.StatusCreated, "success")
}

func PostDBput(c echo.Context) error {
	if !sessionValid(c) && !checkBasicAuth(c) {
		return c.Redirect(http.StatusSeeOther, "/v1/pal/ui/login")
	}

	key := c.FormValue("key")
	value := c.FormValue("value")
	var secret bool
	if c.FormValue("secret") == "on" {
		secret = true
	}

	if key == "" {
		return echo.NewHTTPError(http.StatusNotFound, "error key query param empty")
	}

	dbSet := data.DBSet{
		Key:    key,
		Value:  value,
		Secret: secret,
	}

	err := db.DBC.Put(dbSet)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "error db put for key: "+key)
	}

	return c.Redirect(http.StatusFound, "/v1/pal/ui/db")
}

func registerActionDB(actionData data.ActionData, output, input string) {
	key := actionData.Register.Key
	if key == "" {
		return
	}
	key = strings.ReplaceAll(key, "$PAL_GROUP", actionData.Group)
	key = strings.ReplaceAll(key, "$PAL_ACTION", actionData.Action)
	key = strings.ReplaceAll(key, "$PAL_INPUT", input)
	key = strings.ReplaceAll(key, "$PAL_STATUS", actionData.Status)
	if actionData.Output {
		key = strings.ReplaceAll(key, "$PAL_OUTPUT", output)
	}
	value := actionData.Register.Value
	value = strings.ReplaceAll(value, "$PAL_GROUP", actionData.Group)
	value = strings.ReplaceAll(value, "$PAL_ACTION", actionData.Action)
	value = strings.ReplaceAll(value, "$PAL_INPUT", input)
	value = strings.ReplaceAll(value, "$PAL_STATUS", actionData.Status)
	if actionData.Output {
		value = strings.ReplaceAll(value, "$PAL_OUTPUT", output)
	}

	dbSet := data.DBSet{
		Key:    key,
		Value:  value,
		Secret: actionData.Register.Secret,
	}

	err := db.DBC.Put(dbSet)
	if err != nil {
		log.Println("registerActionDB error DB PUT: " + err.Error())
	}
}

func DeleteDBDel(c echo.Context) error {
	if !sessionValid(c) && !checkBasicAuth(c) {
		return c.String(http.StatusUnauthorized, errorAuth)
	}

	key := c.QueryParam("key")
	if key == "" {
		return echo.NewHTTPError(http.StatusNotFound, "error key query param empty")
	}

	if len(config.GetConfigResponseHeaders()) > 0 {
		for _, v := range config.GetConfigResponseHeaders() {
			if strings.ToLower(v.Header) != "access-control-allow-origin" {
				if strings.ToLower(v.Header) != "access-control-allow-origin" {
					c.Response().Header().Set(v.Header, v.Value)
				}
			}
		}
	}

	err := db.DBC.Delete(key)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "error db put for key: "+key)
	}

	return c.String(http.StatusOK, "success")
}

func GetDBdelete(c echo.Context) error {
	if !sessionValid(c) && !checkBasicAuth(c) {
		return c.Redirect(http.StatusSeeOther, "/v1/pal/ui/login")
	}

	key := c.QueryParam("key")
	if key == "" {
		return echo.NewHTTPError(http.StatusNotFound, "error key query param empty")
	}

	err := db.DBC.Delete(key)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "error db put for key: "+key)
	}

	return c.Redirect(http.StatusTemporaryRedirect, "/v1/pal/ui/db")
}

func PostFilesUpload(c echo.Context) error {
	if !sessionValid(c) && !checkBasicAuth(c) {
		return c.Redirect(http.StatusSeeOther, "/v1/pal/ui/login")
	}

	// Multipart form
	form, err := c.MultipartForm()
	if err != nil {
		return err
	}
	files := form.File["files"]

	for _, file := range files {
		// Source
		src, err := file.Open()
		if err != nil {
			return err
		}
		defer src.Close()

		// Destination
		dst, err := os.Create(config.GetConfigUI().UploadDir + "/" + file.Filename)
		if err != nil {
			return err
		}
		defer dst.Close()

		// Copy
		if _, err = io.Copy(dst, src); err != nil {
			return err
		}
	}

	return c.HTML(http.StatusOK, fmt.Sprintf("<!DOCTYPE html><html><head><meta http-equiv='refresh' content='1; url=/v1/pal/ui/files' /><title>Redirecting...</title></head><body><h2>Successfully uploaded %d files. You will be redirected to <a href='/v1/pal/ui/files'>/v1/pal/ui/files</a> in 1 seconds...</h2></body></html>", len(files)))
}

func GetLogout(c echo.Context) error {
	sess, err := session.Get("session", c)
	if err != nil {
		return c.HTML(http.StatusUnauthorized, `<!DOCTYPE html><html><head><meta http-equiv="refresh" content="1; url=/v1/pal/ui"><title>Redirecting...</title></head><body><h2>You will be redirected to /v1/pal/ui in 1 seconds...</h2></body></html>`)
	}
	sess.Options = &sessions.Options{
		Path:     "/v1/pal",
		MaxAge:   -1,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
	sess.Values["authenticated"] = false
	if err := sess.Save(c.Request(), c.Response()); err != nil {
		return c.HTML(http.StatusUnauthorized, `<!DOCTYPE html><html><head><meta http-equiv="refresh" content="1; url=/v1/pal/ui"><title>Redirecting...</title></head><body><h2>You will be redirected to /v1/pal/ui in 1 seconds...</h2></body></html>`)
	}

	return c.HTML(http.StatusUnauthorized, `<!DOCTYPE html><html><head><meta http-equiv="refresh" content="1; url=/v1/pal/ui"><title>Redirecting...</title></head><body><h2>You will be redirected to /v1/pal/ui in 1 seconds...</h2></body></html>`)
}

func GetRobots(c echo.Context) error {
	return c.String(http.StatusOK, `User-agent: *
Disallow: /`)
}

func GetDBPage(c echo.Context) error {
	if !sessionValid(c) && !checkBasicAuth(c) {
		return c.Redirect(http.StatusSeeOther, "/v1/pal/ui/login")
	}

	uiData := struct {
		Dump          []data.DBSet
		Notifications int
	}{
		Dump:          db.DBC.Dump(),
		Notifications: len(db.DBC.GetNotifications("")),
	}

	return c.Render(http.StatusOK, "db.tmpl", uiData)
}

func GetSystemPage(c echo.Context) error {
	if !sessionValid(c) && !checkBasicAuth(c) {
		return c.Redirect(http.StatusSeeOther, "/v1/pal/ui/login")
	}

	uiData := struct {
		Configs       map[string]string
		Notifications int
	}{}
	uiData.Configs = make(map[string]string)
	var actionsReload string
	parsedTime, err := time.Parse(time.RFC3339, config.GetConfigStr("global_actions_reload"))
	if err == nil {
		actionsReload = humanize.Time(parsedTime)
	}

	uiData.Configs["global_timezone"] = config.GetConfigStr("global_timezone")
	uiData.Configs["global_version"] = config.GetConfigStr("global_version")
	uiData.Configs["global_working_dir"] = config.GetConfigStr("global_working_dir")
	uiData.Configs["global_debug"] = strconv.FormatBool(config.GetConfigBool("global_debug"))
	uiData.Configs["global_cmd_prefix"] = config.GetConfigStr("global_cmd_prefix")
	uiData.Configs["global_actions_dir"] = config.GetConfigStr("global_actions_dir")
	uiData.Configs["global_config_file"] = config.GetConfigStr("global_config_file")
	uiData.Configs["global_container_cmd"] = config.GetConfigStr("global_container_cmd")
	uiData.Configs["global_actions_reload"] = actionsReload
	uiData.Configs["http_timeout_min"] = strconv.Itoa(config.GetConfigInt("http_timeout_min"))
	uiData.Configs["http_body_limit"] = config.GetConfigStr("http_body_limit")
	uiData.Configs["http_max_age"] = strconv.Itoa(config.GetConfigInt("http_max_age"))
	uiData.Configs["http_upload_dir"] = config.GetConfigStr("http_upload_dir")
	uiData.Configs["http_prometheus"] = strconv.FormatBool(config.GetConfigBool("http_prometheus"))
	uiData.Configs["http_ipv6"] = strconv.FormatBool(config.GetConfigBool("http_ipv6"))
	uiData.Configs["http_headers"] = fmt.Sprint(config.GetConfigResponseHeaders())
	uiData.Configs["notifications_store_max"] = strconv.Itoa(config.GetConfigInt("notifications_store_max"))

	uiData.Notifications = len(db.DBC.GetNotifications(""))

	return c.Render(http.StatusOK, "system.tmpl", uiData)
}

func GetActions(c echo.Context) error {
	if !sessionValid(c) && !checkBasicAuth(c) {
		return c.Redirect(http.StatusSeeOther, "/v1/pal/ui/login")
	}
	res := db.DBC.GetGroups()
	var actionsSlice []data.ActionData
	for _, actions := range res {
		actionsSlice = append(actionsSlice, actions...)
	}

	return c.JSON(http.StatusOK, actionsSlice)
}

func GetActionsPage(c echo.Context) error {
	if !sessionValid(c) && !checkBasicAuth(c) {
		return c.Redirect(http.StatusSeeOther, "/v1/pal/ui/login")
	}

	groupMap := make(map[string][]data.ActionData)
	group := c.QueryParam("group")
	if group != "" {
		groupMap[group] = db.DBC.GetGroupActions(group)
		for i, action := range groupMap[group] {
			parsedTime, err := time.Parse(time.RFC3339, action.LastRan)
			if err == nil {
				action.LastRan = humanize.Time(parsedTime)
			}
			parsedTime, err = time.Parse(time.RFC3339, action.LastSuccess)
			if err == nil {
				action.LastSuccess = humanize.Time(parsedTime)
			}
			parsedTime, err = time.Parse(time.RFC3339, action.LastFailure)
			if err == nil {
				action.LastFailure = humanize.Time(parsedTime)
			}
			groupMap[group][i] = action
		}
	} else {
		res := db.DBC.GetGroups()

		for groupKey, groupData := range res {
			groupMap[groupKey] = make([]data.ActionData, len(groupData))
			for i, action := range groupData {
				parsedTime, err := time.Parse(time.RFC3339, action.LastRan)
				if err == nil {
					action.LastRan = humanize.Time(parsedTime)
				}
				parsedTime, err = time.Parse(time.RFC3339, action.LastSuccess)
				if err == nil {
					action.LastSuccess = humanize.Time(parsedTime)
				}
				parsedTime, err = time.Parse(time.RFC3339, action.LastFailure)
				if err == nil {
					action.LastFailure = humanize.Time(parsedTime)
				}
				for runIndex, run := range action.RunHistory {
					parsedTime, err = time.Parse(time.RFC3339, run.Ran)
					if err == nil {
						action.RunHistory[runIndex].Ran = humanize.Time(parsedTime)
					}
				}
				groupMap[groupKey][i] = action
			}
		}
	}

	sess, err := session.Get("session", c)
	if err != nil {
		return err
	}

	username, ok := sess.Values["username"].(string)
	if !ok {
		username = ""
	}

	refresh, ok := sess.Values["refresh"].(string)
	if !ok {
		refresh = ""
	}

	// Parse the timestamp string using the RFC3339 layout
	parsedTime, err := time.Parse(time.RFC3339, utils.TimeNow(config.GetConfigStr("global_timezone")))
	if err != nil {
		return err
	}

	tmpl, err := template.New("actions.tmpl").Funcs(template.FuncMap{
		"getData": func() map[string][]data.ActionData {
			return groupMap
		},
		"Username": func() string {
			return username
		},
		"Refresh": func() string {
			return refresh
		},
		"TimeNow": func() string {
			return parsedTime.Format("Monday, January 2, 2006 at 3:04 PM")
		},
		"Notifications": func() int {
			return len(db.DBC.GetNotifications(""))
		},
	}).ParseFS(ui.UIFiles, "actions.tmpl")
	if err != nil {
		return err
	}

	return tmpl.Execute(c.Response(), nil)
}

func GetActionPage(c echo.Context) error {
	if !sessionValid(c) && !checkBasicAuth(c) {
		return c.Redirect(http.StatusSeeOther, "/v1/pal/ui/login")
	}
	group := c.Param("group")
	if group == "" {
		return c.String(http.StatusBadRequest, errorGroup)
	}
	action := c.Param("action")
	if action == "" {
		return c.String(http.StatusBadRequest, errorAction)
	}

	disable := c.QueryParam("disable")
	if disable != "" {
		state := false

		if disable == "true" {
			state = true
		}

		condDisable(group, action, state)
	}

	res := db.DBC.GetGroupAction(group, action)

	uiData := struct {
		ActionMap     map[string]data.ActionData
		Notifications int
	}{}

	for runIndex, run := range res.RunHistory {
		parsedTime, err := time.Parse(time.RFC3339, run.Ran)
		if err == nil {
			res.RunHistory[runIndex].Ran = humanize.Time(parsedTime)
		}
	}

	uiData.ActionMap = map[string]data.ActionData{
		group: res,
	}

	uiData.Notifications = len(db.DBC.GetNotifications(""))

	return c.Render(http.StatusOK, "action.tmpl", uiData)
}

func GetResetAction(c echo.Context) error {
	if !sessionValid(c) && !checkBasicAuth(c) {
		return c.Redirect(http.StatusSeeOther, "/v1/pal/ui/login")
	}

	group := c.Param("group")
	if group == "" {
		return c.String(http.StatusBadRequest, errorGroup)
	}

	action := c.Param("action")
	if action == "" {
		return c.String(http.StatusBadRequest, errorAction)
	}

	res := db.DBC.GetGroupAction(group, action)
	res.RunCount = 0
	db.DBC.PutGroupAction(group, res)

	return c.Redirect(http.StatusSeeOther, "/v1/pal/ui/action/"+group+"/"+action)
}

func Yaml(c echo.Context, code int, i interface{}) error {
	c.Response().Status = code
	c.Response().Header().Set(echo.HeaderContentType, "text/yaml")
	return yaml.NewEncoder(c.Response()).Encode(i)
}

func GetAction(c echo.Context) error {
	if !sessionValid(c) && !checkBasicAuth(c) {
		return c.JSON(http.StatusUnauthorized, data.GenericResponse{Err: "Unauthorized no valid session or basic auth."})
	}

	group := c.QueryParam("group")
	if group == "" {
		return c.JSON(http.StatusBadRequest, data.GenericResponse{Err: errorGroup})
	}

	action := c.QueryParam("action")
	if action == "" {
		return c.JSON(http.StatusBadRequest, data.GenericResponse{Err: errorAction})
	}

	yaml := c.QueryParam("yml")

	disable := c.QueryParam("disabled")
	if disable != "" {
		state := false

		if disable == "true" {
			state = true
		}

		condDisable(group, action, state)
		return c.JSON(http.StatusOK, data.GenericResponse{Message: fmt.Sprintf("changed action state %s/%s disabled to %s", group, action, disable)})
	}

	resMap := db.DBC.GetGroupAction(group, action)

	if resMap.Action == action {
		resMap.AuthHeader = "hidden"
		if yaml == "true" {
			return Yaml(c, http.StatusOK, resMap)
		}
		return c.JSONPretty(http.StatusOK, resMap, "  ")
	}

	return c.JSON(http.StatusOK, data.ActionData{})
}

func GetFilesPage(c echo.Context) error {
	if !sessionValid(c) && !checkBasicAuth(c) {
		return c.Redirect(http.StatusSeeOther, "/v1/pal/ui/login")
	}

	// Parse the template from the string
	tmpl, err := template.New("files.tmpl").Funcs(template.FuncMap{
		"fileSize": func(file fs.DirEntry) string {
			info, _ := file.Info()
			return humanize.Bytes(uint64(info.Size())) // #nosec G115
		},
		"fileModTime": func(file fs.DirEntry) string {
			info, _ := file.Info()
			return humanize.Time(info.ModTime())
		},
	}).ParseFS(ui.UIFiles, "files.tmpl")
	if err != nil {
		return err
	}

	dirPath := config.GetConfigUI().UploadDir

	// Read directory contents
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "error reading directory: "+dirPath)
	}

	// Prepare data for the template
	uiData := struct {
		Notifications int
		Files         []fs.DirEntry
	}{
		Notifications: len(db.DBC.GetNotifications("")),
		Files:         files,
	}

	// Render the template to the response
	return tmpl.Execute(c.Response(), uiData)
}

func PostLoginPage(c echo.Context) error {
	if c.FormValue("username") == strings.Split(config.GetConfigUI().BasicAuth, " ")[0] && c.FormValue("password") == strings.Split(config.GetConfigUI().BasicAuth, " ")[1] {
		sess, err := session.Get("session", c)
		if err != nil {
			return err
		}
		sess.Options = &sessions.Options{
			Path: "/v1/pal",
			// 3600 seconds = 1 hour
			MaxAge:   config.GetConfigInt("http_max_age"),
			Secure:   true,
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
		}
		sess.Values["authenticated"] = true
		sess.Values["username"] = c.FormValue("username")
		sess.Values["refresh"] = "off"
		if err := sess.Save(c.Request(), c.Response()); err != nil {
			return err
		}
		return c.Redirect(http.StatusSeeOther, "/v1/pal/ui")
	}
	return c.Redirect(http.StatusFound, "/v1/pal/ui/login")
}

func GetLoginPage(c echo.Context) error {
	return c.Render(http.StatusOK, "login.tmpl", nil)
}

func GetFilesDownload(c echo.Context) error {
	if !sessionValid(c) && !checkBasicAuth(c) {
		return c.Redirect(http.StatusSeeOther, "/v1/pal/ui/login")
	}

	file := c.Param("file")

	if strings.Contains(file, "/") || strings.Contains(file, "\\") || strings.Contains(file, "..") {
		return echo.NewHTTPError(http.StatusInternalServerError, "error invalid file name: "+file)
	}

	path := config.GetConfigUI().UploadDir + "/" + file
	absPath, err := filepath.Abs(path)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "error file not in path "+path)
	}

	return c.File(absPath)
}

func GetFavicon(c echo.Context) error {
	return c.Blob(http.StatusOK, "image/svg+xml", []byte(favicon))
}

func GetFilesDelete(c echo.Context) error {
	if !sessionValid(c) && !checkBasicAuth(c) {
		return c.Redirect(http.StatusSeeOther, "/v1/pal/ui/login")
	}
	file := c.Param("file")

	path := config.GetConfigUI().UploadDir + "/" + file
	absPath, err := filepath.Abs(path)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "error deleting file file path "+path)
	}

	err = os.Remove(absPath)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "error deleting file os remove"+file)
	}
	return c.Redirect(http.StatusTemporaryRedirect, "/v1/pal/ui/files")
}

func GetReloadActions(c echo.Context) error {
	if !sessionValid(c) && !checkBasicAuth(c) {
		return c.Redirect(http.StatusSeeOther, "/v1/pal/ui/login")
	}
	groups := config.ReadConfig(config.GetConfigStr("global_actions_dir"))
	err := ReloadActions(groups)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "error reloading actions "+err.Error())
	}
	config.SetActionsReload()
	return c.Redirect(http.StatusTemporaryRedirect, "/v1/pal/ui/system")
}

func RedirectUI(c echo.Context) error {
	return c.Redirect(http.StatusSeeOther, "/v1/pal/ui")
}

func GetRefreshPage(c echo.Context) error {
	if !sessionValid(c) && !checkBasicAuth(c) {
		return c.Redirect(http.StatusSeeOther, "/v1/pal/ui/login")
	}

	interv := c.QueryParam("set")

	err := validateInput(interv, "lt=10")
	if err != nil {
		return err
	}

	sess, err := session.Get("session", c)
	if err != nil {
		return err
	}

	sess.Options = &sessions.Options{
		Path: "/v1/pal",
		// 3600 seconds = 1 hour
		MaxAge:   config.GetConfigInt("http_max_age"),
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}

	sess.Values["authenticated"] = true
	sess.Values["refresh"] = interv

	if err := sess.Save(c.Request(), c.Response()); err != nil {
		return err
	}

	return c.Redirect(http.StatusSeeOther, "/v1/pal/ui")
}

func sessionValid(c echo.Context) bool {
	sess, err := session.Get("session", c)
	if err != nil {
		return false
	}
	auth, ok := sess.Values["authenticated"]
	if !ok {
		return false
	}

	val, ok := auth.(bool)
	if !ok {
		return false
	}
	return val
}

func cronTask(res data.ActionData) string {
	actionsData := db.DBC.GetGroupAction(res.Group, res.Action)

	if actionsData.Disabled {
		return "error action disabled"
	}

	cmdOrig := actionsData.Cmd
	actionsData.Cmd = cmdString(actionsData, "", "")
	timeNow := utils.TimeNow(config.GetConfigStr("global_timezone"))
	var (
		cmdOutput string
		duration  string
		err       error
	)
	if res.Image != "" {
		cmdOutput, duration, err = utils.CmdRunContainerized(res, config.GetConfigStr("global_podman_socket"), config.GetConfigStr("global_cmd_prefix"), config.GetConfigStr("global_working_dir"))
	} else {
		cmdOutput, duration, err = utils.CmdRun(res, config.GetConfigStr("global_cmd_prefix"), config.GetConfigStr("global_working_dir"))
	}

	actionsData.Cmd = cmdOrig
	if err != nil {
		actionsData.Status = "error"
		actionsData.RunCount++
		actionsData.LastDuration = duration
		actionsData.LastRan = timeNow
		actionsData.LastFailure = timeNow
		if actionsData.Output {
			actionsData.LastFailureOutput = err.Error() + " " + cmdOutput
		}
		registerActionDB(actionsData, actionsData.LastFailureOutput, "")
		mergeGroup(res)
		logError("", "", err)
		if actionsData.OnError.Notification != "" {
			notification := actionsData.OnError.Notification
			notification = strings.ReplaceAll(notification, "$PAL_GROUP", actionsData.Group)
			notification = strings.ReplaceAll(notification, "$PAL_ACTION", actionsData.Action)
			notification = strings.ReplaceAll(notification, "$PAL_INPUT", actionsData.Input)
			notification = strings.ReplaceAll(notification, "$PAL_STATUS", actionsData.Status)
			if actionsData.Output {
				notification = strings.ReplaceAll(notification, "$PAL_OUTPUT", actionsData.LastFailureOutput)
			}
			err := putNotifications(data.Notification{Group: actionsData.Group, Action: actionsData.Action, Status: actionsData.Status, Notification: notification})
			if err != nil {
				logError("", "", err)
			}
		}
		sendWebhookNotifications(actionsData, actionsData.LastFailureOutput, "")
		for _, e := range actionsData.OnError.Run {
			go func() {
				errorGroup := e.Group
				errorAction := e.Action
				errorInput := e.Input
				errorInput = strings.ReplaceAll(errorInput, "$PAL_GROUP", actionsData.Group)
				errorInput = strings.ReplaceAll(errorInput, "$PAL_ACTION", actionsData.Action)
				if actionsData.Output {
					errorInput = strings.ReplaceAll(errorInput, "$PAL_OUTPUT", actionsData.LastFailureOutput)
				}
				errorInput = strings.ReplaceAll(errorInput, "$PAL_STATUS", actionsData.Status)
				runBackground(errorGroup, errorAction, errorInput)
			}()
		}
		return err.Error()
	}

	// TODO: Log here
	// fmt.Printf("%s\n", fmt.Sprintf(`{"time":"%s","group":"%s","job_success":%t}`, timeNow, actionsData.Group+"/"+actionsData.Action, true))

	actionsData.Status = "success"
	actionsData.RunCount++
	actionsData.LastDuration = duration
	actionsData.LastRan = timeNow
	actionsData.LastSuccess = timeNow
	if actionsData.Output {
		actionsData.LastSuccessOutput = cmdOutput
	}
	registerActionDB(actionsData, actionsData.LastFailureOutput, "")
	mergeGroup(actionsData)
	if actionsData.OnSuccess.Notification != "" {
		notification := actionsData.OnSuccess.Notification
		notification = strings.ReplaceAll(notification, "$PAL_GROUP", actionsData.Group)
		notification = strings.ReplaceAll(notification, "$PAL_ACTION", actionsData.Action)
		notification = strings.ReplaceAll(notification, "$PAL_INPUT", actionsData.Input)
		notification = strings.ReplaceAll(notification, "$PAL_STATUS", actionsData.Status)
		if actionsData.Output {
			notification = strings.ReplaceAll(notification, "$PAL_OUTPUT", cmdOutput)
		}
		err := putNotifications(data.Notification{Group: actionsData.Group, Action: actionsData.Action, Status: actionsData.Status, Notification: notification})
		if err != nil {
			logError("", "", err)
		}
	}
	sendWebhookNotifications(actionsData, actionsData.LastSuccessOutput, "")
	for _, e := range actionsData.OnSuccess.Run {
		go func() {
			successGroup := e.Group
			successAction := e.Action
			successInput := e.Input
			successInput = strings.ReplaceAll(successInput, "$PAL_GROUP", actionsData.Group)
			successInput = strings.ReplaceAll(successInput, "$PAL_ACTION", actionsData.Action)
			successInput = strings.ReplaceAll(successInput, "$PAL_STATUS", actionsData.Status)
			if actionsData.Output {
				successInput = strings.ReplaceAll(successInput, "$PAL_OUTPUT", cmdOutput)
			}
			runBackground(successGroup, successAction, successInput)
		}()
	}
	return cmdOutput
}

func CronStart(r map[string][]data.ActionData) error {
	loc, err := time.LoadLocation(config.GetConfigStr("global_timezone"))
	if err != nil {
		return err
	}

	sched, err = gocron.NewScheduler(gocron.WithLocation(loc))
	if err != nil {
		return err
	}

	for k, v := range r {
		for _, e := range v {
			if len(e.Crons) > 0 {
				for _, c := range e.Crons {
					if validateInput(c, "cron") == nil {
						var cronDesc string
						exprDesc, err := cron.NewDescriptor()
						if err == nil {
							cronDesc, err = exprDesc.ToDescription(c, cron.Locale_en)
							if err != nil {
								cronDesc = ""
							}
						}
						_, err = sched.NewJob(
							gocron.CronJob(c, false),
							gocron.NewTask(cronTask, e),
							gocron.WithName(k+"/"+e.Action),
							gocron.WithTags(cronDesc, e.Group+e.Action),
						)
						if err != nil {
							return err
						}
					}
				}
			}
		}
	}

	sched.Start()

	return nil
}

func mergeGroup(action data.ActionData) {
	action = addRun(action)
	// Dont use PutGroupAction instead, set Action field in ActionData
	groupsData := db.DBC.GetGroups()
	if v, ok := groupsData[action.Group]; ok {
		for i, e := range v {
			if e.Action == action.Action {
				v[i] = action
				groupsData[action.Group] = v
				err := db.DBC.PutGroups(groupsData)
				if err != nil {
					// TODO: DEBUG STATEMENT
					log.Println(err.Error())
				}
				return
			}
		}
	}
}

func addRun(action data.ActionData) data.ActionData {
	run := data.RunHistory{
		Ran:      action.LastRan,
		Duration: action.LastDuration,
		Status:   action.Status,
	}

	action.RunHistory = append([]data.RunHistory{run}, action.RunHistory...)

	// more than 5 items, remove the last one the oldest
	if len(action.RunHistory) > runHistoryLimit {
		// create a new slice containing all elements except the last one.
		action.RunHistory = action.RunHistory[:len(action.RunHistory)-1]
	}

	return action
}

func putNotifications(notification data.Notification) error {
	notifications := db.DBC.GetNotifications("")

	if len(notifications) > config.GetConfigInt("notifications_store_max") {
		notifications = notifications[1:]
	}

	notification.ID = uuid.NewString()
	notification.NotificationRcv = utils.TimeNow(config.GetConfigStr("global_timezone"))

	notifications = append([]data.Notification{notification}, notifications...)

	return db.DBC.PutNotifications(notifications)
}

func sendWebhookNotifications(actionData data.ActionData, output, input string) {
	webhooks := config.GetConfigWebHooks()

	var webhookNames []string
	if actionData.Status == "error" {
		webhookNames = actionData.OnError.Webhook
	} else {
		webhookNames = actionData.OnSuccess.Webhook
	}

	// Iterate over each configured webhook
	for _, webhook := range webhooks {
		for _, name := range webhookNames {
			if name == webhook.Name {
				notification := webhook.Body
				notification = strings.ReplaceAll(notification, "$PAL_GROUP", actionData.Group)
				notification = strings.ReplaceAll(notification, "$PAL_ACTION", actionData.Action)
				notification = strings.ReplaceAll(notification, "$PAL_INPUT", input)
				notification = strings.ReplaceAll(notification, "$PAL_STATUS", actionData.Status)
				if actionData.Output {
					notification = strings.ReplaceAll(notification, "$PAL_OUTPUT", output)
				}

				log.Printf("Sending webhook notification to: %s\n", webhook.Name)

				ctx, cancel := context.WithTimeout(context.Background(), httpClientTimeout*time.Second)
				defer cancel()

				// Create a new request using the method, URL, and passed-in body
				req, err := http.NewRequestWithContext(ctx, webhook.Method, webhook.URL, bytes.NewBufferString(notification))
				if err != nil {
					log.Printf("Error creating webhook request for %s: %v", webhook.Name, err)
					continue // Move to the next webhook
				}

				// Add all configured headers to the request
				for _, h := range webhook.Headers {
					req.Header.Set(h.Header, h.Value)
				}

				transport := &http.Transport{
					// The core of the solution is in TLSClientConfig
					TLSClientConfig: &tls.Config{
						// This line disables certificate verification
						InsecureSkipVerify: webhook.Insecure,
					},
				}

				client := &http.Client{
					Transport: transport,
					Timeout:   httpClientTimeout * time.Second,
				}

				// Execute the request
				resp, err := client.Do(req)
				if err != nil {
					log.Printf("Error sending webhook request to %s: %v", webhook.Name, err)
					continue // Move to the next webhook
				}

				bodyBytes, err := io.ReadAll(resp.Body)
				if err != nil {
					log.Printf("Error reading webhook response body: %v", err)
					continue
				}

				// It's important to close the response body to free up connections
				defer resp.Body.Close()
				if resp.StatusCode < httpStatusCodeCheck {
					log.Printf("Successfully sent webhook request to %s, Status: %s\n", webhook.Name, resp.Status)
				} else {
					log.Printf("Error sending webhook request to %s, Status: %s, Body: %s\n", webhook.Name, resp.Status, bodyBytes)
				}
			}
		}
	}
}

func validateInput(input, inputValidate string) error {
	return validate.Var(input, inputValidate)
}

func requestJSON(c echo.Context, input string) (string, error) {
	type RequestData struct {
		Method      string              `json:"method"`
		URL         string              `json:"url"`
		Headers     map[string]string   `json:"headers"`
		QueryParams map[string][]string `json:"query_params"`
		Body        string              `json:"body"`
	}

	requestData := RequestData{
		Method:      c.Request().Method,
		URL:         c.Request().URL.String(),
		Headers:     make(map[string]string),
		QueryParams: c.QueryParams(),
		Body:        input,
	}

	for key, values := range c.Request().Header {
		requestData.Headers[key] = strings.Join(values, ", ")
	}

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return "", err
	}

	return string(jsonData), nil
}

func cmdString(actionData data.ActionData, input, req string) string {
	var cmd string
	var sudo string
	if actionData.Container.Image != "" {
		containerCmd := config.GetConfigStr("global_container_cmd")
		if actionData.Container.Sudo {
			sudo = "sudo"
		}
		envVars := fmt.Sprintf("-e PAL_UPLOAD_DIR='%s' -e PAL_GROUP='%s' -e PAL_ACTION='%s' -e PAL_INPUT='%s' -e PAL_REQUEST='%s'", config.GetConfigStr("http_upload_dir"), actionData.Group, actionData.Action, input, req)
		cmd = fmt.Sprintf("%s %s run --rm %s %s %s %s '%s'", sudo, containerCmd, envVars, actionData.Container.Options, actionData.Container.Image, config.GetConfigStr("global_cmd_prefix"), actionData.Cmd)
	} else {
		cmdArg := fmt.Sprintf("export PAL_UPLOAD_DIR='%s'; export PAL_GROUP='%s'; export PAL_ACTION='%s'; export PAL_INPUT='%s'; export PAL_REQUEST='%s';", config.GetConfigStr("http_upload_dir"), actionData.Group, actionData.Action, input, req)
		cmd = strings.Join([]string{cmdArg, actionData.Cmd}, " ")
	}
	// TODO: Add Debug cmd output
	return cmd
}

func runBackground(group, action, input string) {
	actionData := db.DBC.GetGroupAction(group, action)
	origCmd := actionData.Cmd
	actionData.Cmd = cmdString(actionData, input, "")
	var (
		cmdOutput string
		duration  string
		err       error
	)
	if actionData.Image != "" {
		cmdOutput, duration, err = utils.CmdRunContainerized(actionData, config.GetConfigStr("global_podman_socket"), config.GetConfigStr("global_cmd_prefix"), config.GetConfigStr("global_working_dir"))
	} else {
		cmdOutput, duration, err = utils.CmdRun(actionData, config.GetConfigStr("global_cmd_prefix"), config.GetConfigStr("global_working_dir"))
	}

	actionData.Cmd = origCmd
	if err != nil {
		if !actionData.Concurrent {
			lock(actionData.Group, actionData.Action, false)
		}
		actionData.Status = "error"
		actionData.RunCount++
		actionData.LastDuration = duration
		actionData.LastRan = utils.TimeNow(config.GetConfigStr("global_timezone"))
		actionData.LastFailure = actionData.LastRan
		if actionData.Output {
			actionData.LastFailureOutput = err.Error() + " " + cmdOutput
		}
		mergeGroup(actionData)
		registerActionDB(actionData, actionData.LastFailureOutput, input)
		logError("", "", err)
		if actionData.OnError.Notification != "" {
			notification := actionData.OnError.Notification
			notification = strings.ReplaceAll(notification, "$PAL_GROUP", actionData.Group)
			notification = strings.ReplaceAll(notification, "$PAL_ACTION", actionData.Action)
			notification = strings.ReplaceAll(notification, "$PAL_INPUT", input)
			notification = strings.ReplaceAll(notification, "$PAL_STATUS", actionData.Status)
			if actionData.Output {
				notification = strings.ReplaceAll(notification, "$PAL_OUTPUT", actionData.LastFailureOutput)
			}
			err := putNotifications(data.Notification{Group: actionData.Group, Action: actionData.Action, Status: actionData.Status, Notification: notification})
			if err != nil {
				logError("", "", err)
			}
		}
		sendWebhookNotifications(actionData, actionData.LastFailureOutput, input)
		for _, e := range actionData.OnError.Run {
			go func() {
				errorGroup := e.Group
				errorAction := e.Action
				errorInput := e.Input
				errorInput = strings.ReplaceAll(errorInput, "$PAL_GROUP", actionData.Group)
				errorInput = strings.ReplaceAll(errorInput, "$PAL_ACTION", actionData.Action)
				if actionData.Output {
					errorInput = strings.ReplaceAll(errorInput, "$PAL_OUTPUT", actionData.LastFailureOutput)
				}
				errorInput = strings.ReplaceAll(errorInput, "$PAL_STATUS", actionData.Status)
				runBackground(errorGroup, errorAction, errorInput)
			}()
		}
		return
	}
	actionData.Status = "success"
	actionData.RunCount++
	actionData.LastDuration = duration
	actionData.LastRan = utils.TimeNow(config.GetConfigStr("global_timezone"))
	actionData.LastSuccess = actionData.LastRan
	if actionData.Output {
		actionData.LastSuccessOutput = cmdOutput
	}
	mergeGroup(actionData)
	registerActionDB(actionData, actionData.LastSuccessOutput, input)
	if actionData.OnSuccess.Notification != "" {
		notification := actionData.OnSuccess.Notification
		notification = strings.ReplaceAll(notification, "$PAL_GROUP", actionData.Group)
		notification = strings.ReplaceAll(notification, "$PAL_ACTION", actionData.Action)
		notification = strings.ReplaceAll(notification, "$PAL_INPUT", input)
		notification = strings.ReplaceAll(notification, "$PAL_STATUS", actionData.Status)
		if actionData.Output {
			notification = strings.ReplaceAll(notification, "$PAL_OUTPUT", cmdOutput)
		}
		err := putNotifications(data.Notification{Group: group, Action: actionData.Action, Status: actionData.Status, Notification: notification})
		if err != nil {
			logError("", "", err)
		}
	}
	sendWebhookNotifications(actionData, actionData.LastSuccessOutput, input)
	for _, e := range actionData.OnSuccess.Run {
		go func() {
			successGroup := e.Group
			successAction := e.Action
			successInput := e.Input
			successInput = strings.ReplaceAll(successInput, "$PAL_GROUP", actionData.Group)
			successInput = strings.ReplaceAll(successInput, "$PAL_ACTION", actionData.Action)
			if actionData.Output {
				successInput = strings.ReplaceAll(successInput, "$PAL_OUTPUT", cmdOutput)
			}
			successInput = strings.ReplaceAll(successInput, "$PAL_STATUS", actionData.Status)
			runBackground(successGroup, successAction, successInput)
		}()
	}

	if !actionData.Concurrent {
		lock(actionData.Group, actionData.Action, false)
	}
}

func ReloadActions(groups map[string][]data.ActionData) error {
	actionIndex := make(map[string]map[string]*data.ActionData)
	for groupName, actions := range groups {
		actionIndex[groupName] = make(map[string]*data.ActionData)
		for i := range actions {
			action := &actions[i]
			action.Group = groupName
			if len(action.RunHistory) == 0 {
				run := data.RunHistory{
					Ran:      "",
					Duration: "",
					Status:   "",
				}
				for range 5 {
					action.RunHistory = append(action.RunHistory, run)
				}
			}
			actionIndex[groupName][action.Action] = action
		}
	}

	processTriggerRules := func(originAction *data.ActionData, rules []data.Run, condition string) {
		for _, rule := range rules {
			targetAction, found := actionIndex[rule.Group][rule.Action]
			if !found {
				// log.Printf("Warning: Target action '%s' in group '%s' not found for trigger from action '%s'.",
				// 	rule.Action, rule.Group, originAction.Action)
				continue // Skip this trigger if the target doesn't exist.
			}

			trigger := data.Triggers{
				OriginGroup:      originAction.Group,
				OriginAction:     originAction.Action,
				TriggerGroup:     rule.Group,
				TriggerAction:    rule.Action,
				TriggerCondition: condition,
				TriggerInput:     rule.Input,
			}

			originAction.Triggers = append(originAction.Triggers, trigger)
			targetAction.Triggers = append(targetAction.Triggers, trigger)
		}
	}

	for _, groupMap := range actionIndex {
		for _, action := range groupMap {
			processTriggerRules(action, action.OnSuccess.Run, "success")
			processTriggerRules(action, action.OnError.Run, "error")
		}
	}

	mergedGroups := utils.MergeGroups(db.DBC.GetGroups(), groups)
	return db.DBC.PutGroups(mergedGroups)
}
