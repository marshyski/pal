package routes

import (
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
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
)

const (
	errorAuth     = "error unauthorized"
	errorScript   = "error script fail"
	errorNotReady = "error not ready"
	errorAction   = "error invalid action"
	errorGroup    = "error group invalid"
	errorCmdEmpty = "error cmd is empty for action"
)

var (
	sched    gocron.Scheduler
	validate = validator.New(validator.WithRequiredStructEnabled())
)

func authHeaderCheck(headers map[string][]string) bool {
	pass := false
	for k, v := range headers {
		header := strings.Join([]string{k, v[0]}, " ")
		if header == config.GetConfigStr("http_auth_header") {
			pass = true
		}
	}

	return pass
}

// lock sets the lock for blocking requests until cmd has finished
func lock(group, action string, lockState bool) {

	resData := db.DBC.GetGroupActions(group)

	for i, e := range resData {
		if e.Action == action {
			resData[i].Lock = lockState
			db.DBC.PutGroupActions(group, resData)
			return
		}
	}
}

func condDisable(group, action string, disabled bool) {

	resData := db.DBC.GetGroups()

	if v, ok := resData[group]; ok {
		for i, e := range v {
			if e.Action == action {
				resData[group][i].Disabled = disabled
				err := db.DBC.PutGroups(resData)
				if err != nil {
					// TODO: DEBUG STATEMENT
					log.Println(err.Error())
				}
				if disabled {
					sched.RemoveByTags(group + action)
				} else {
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
									gocron.NewTask(cronTask, resData[group][i]),
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
		}
	}
}

func getCond(group, action string) bool {
	resData := db.DBC.GetGroups()

	if v, ok := resData[group]; ok {
		for _, e := range v {
			if e.Action == action {
				return e.Disabled
			}
		}
	}

	return false
}

// logError time, error, id, uri fields
func logError(reqid, uri string, e error) {
	fmt.Printf("%s\n", fmt.Sprintf(`{"time":"%s","error":"%s","id":"%s","uri":"%s"}`,
		utils.TimeNow(config.GetConfigStr("global_timezone")), e.Error(), reqid, uri))
}

// RunGroup is the main route for triggering a command
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

	if actionData.Disabled {
		return c.String(http.StatusBadRequest, "error action is disabled")
	}

	// set custom headers
	if len(actionData.ResponseHeaders) > 0 {
		for _, v := range actionData.ResponseHeaders {
			c.Response().Header().Set(v.Header, v.Value)
		}
	}

	auth, authHeader := utils.GetAuthHeader(actionData)

	// Check if auth header is present and if the header is correct
	if auth {
		pass := false
		if strings.HasPrefix(c.Request().RequestURI, "/v1/pal/ui") {
			if !sessionValid(c) {
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
			return c.String(http.StatusOK, actionData.LastOutput)
		}
		return c.String(http.StatusBadRequest, "error output not enabled")
	}

	cmdOrig := actionData.Cmd

	var input string

	if c.Request().Method == "POST" {
		bodyBytes, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "error reading request body in post run")
		}
		input = string(bodyBytes)
	} else {
		input = c.QueryParam("input")
	}

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
			cmdOutput, duration, err := utils.CmdRun(actionData, config.GetConfigStr("global_cmd_prefix"), config.GetConfigStr("global_working_dir"))
			actionData.Cmd = cmdOrig
			if err != nil {
				if !actionData.Concurrent {
					lock(group, action, false)
				}
				actionData.Status = "error"
				actionData.LastDuration = duration
				actionData.LastRan = utils.TimeNow(config.GetConfigStr("global_timezone"))
				if actionData.Output {
					actionData.LastOutput = err.Error()
				}
				mergeGroup(actionData)
				logError("", "", err)
				if actionData.OnError.Notification != "" {
					notification := actionData.OnError.Notification
					notification = strings.ReplaceAll(notification, "$PAL_GROUP", actionData.Group)
					notification = strings.ReplaceAll(notification, "$PAL_ACTION", actionData.Action)
					notification = strings.ReplaceAll(notification, "$PAL_INPUT", input)
					if actionData.Output {
						notification = strings.ReplaceAll(notification, "$PAL_OUTPUT", actionData.LastOutput)
					}
					err := putNotifications(data.Notification{Group: group, Notification: notification})
					if err != nil {
						logError("", "", err)
					}
				}
				return
			}
			actionData.Status = "success"
			actionData.LastDuration = duration
			actionData.LastRan = utils.TimeNow(config.GetConfigStr("global_timezone"))
			if actionData.Output {
				actionData.LastOutput = cmdOutput
			}
			mergeGroup(actionData)
		}()

		if !actionData.Concurrent {
			lock(group, action, false)
		}

		return c.String(http.StatusOK, "running in background")
	}

	cmdOutput, duration, err := utils.CmdRun(actionData, config.GetConfigStr("global_cmd_prefix"), config.GetConfigStr("global_working_dir"))
	actionData.Cmd = cmdOrig
	if err != nil {
		if !actionData.Concurrent {
			lock(group, action, false)
		}
		actionData.Status = "error"
		actionData.LastDuration = duration
		actionData.LastRan = utils.TimeNow(config.GetConfigStr("global_timezone"))
		if actionData.Output {
			actionData.LastOutput = err.Error()
		}
		mergeGroup(actionData)
		logError(c.Response().Header().Get(echo.HeaderXRequestID), c.Request().RequestURI, errors.New(errorScript+" "+err.Error()))
		if actionData.OnError.Notification != "" {
			notification := actionData.OnError.Notification
			notification = strings.ReplaceAll(notification, "$PAL_GROUP", actionData.Group)
			notification = strings.ReplaceAll(notification, "$PAL_ACTION", actionData.Action)
			notification = strings.ReplaceAll(notification, "$PAL_INPUT", input)
			if actionData.Output {
				notification = strings.ReplaceAll(notification, "$PAL_OUTPUT", actionData.LastOutput)
			}
			err := putNotifications(data.Notification{Group: group, Notification: notification})
			if err != nil {
				logError(c.Response().Header().Get(echo.HeaderXRequestID), c.Request().RequestURI, err)
			}
		}
		return c.String(http.StatusInternalServerError, err.Error())
	}

	// Unlock if blocking is enabled
	if !actionData.Concurrent {
		lock(group, action, false)
	}

	if actionData.Output {
		actionData.Status = "success"
		actionData.LastDuration = duration
		actionData.LastRan = utils.TimeNow(config.GetConfigStr("global_timezone"))
		if actionData.Output {
			actionData.LastOutput = cmdOutput
		}
		mergeGroup(actionData)
		return c.String(http.StatusOK, cmdOutput)
	}

	actionData.Status = "success"
	actionData.LastDuration = duration
	actionData.LastRan = utils.TimeNow(config.GetConfigStr("global_timezone"))
	if actionData.Output {
		actionData.LastOutput = cmdOutput
	}
	mergeGroup(actionData)
	return c.String(http.StatusOK, "done")
}

func GetHealth(c echo.Context) error {
	return c.String(http.StatusOK, "ok")
}

func GetCond(c echo.Context) error {
	if !sessionValid(c) {
		return c.JSON(http.StatusUnauthorized, data.GenericResponse{Err: "Unauthorized no valid session or auth header present."})
	}

	disable := c.QueryParam("disable")

	state := false

	if disable == "true" {
		state = true
	}

	group := c.Param("group")
	action := c.Param("action")

	condDisable(group, action, state)

	return c.Redirect(http.StatusSeeOther, "/v1/pal/ui")
}

func GetNotifications(c echo.Context) error {
	if !sessionValid(c) {
		if !authHeaderCheck(c.Request().Header) {
			return c.JSON(http.StatusUnauthorized, data.GenericResponse{Err: "Unauthorized no valid session or auth header present."})
		}
	}

	return c.JSON(http.StatusOK, db.DBC.GetNotifications(c.QueryParam("group")))
}

func PutNotifications(c echo.Context) error {
	if !sessionValid(c) {
		if !authHeaderCheck(c.Request().Header) {
			return c.JSON(http.StatusUnauthorized, data.GenericResponse{Err: "Unauthorized no valid session or auth header present."})
		}
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

func GetNotificationsPage(c echo.Context) error {
	if !sessionValid(c) {
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
	if !sessionValid(c) {
		if !authHeaderCheck(c.Request().Header) {
			return c.JSON(http.StatusUnauthorized, data.GenericResponse{Err: "Unauthorized no valid session or auth header present."})
		}
	}

	group := c.QueryParam("group")
	if group == "" {
		return c.JSON(http.StatusInternalServerError, data.GenericResponse{Err: "missing group query parameter"})
	}

	action := c.QueryParam("action")
	if action == "" {
		return c.JSON(http.StatusInternalServerError, data.GenericResponse{Err: "missing action query parameter"})
	}

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

		lastrun, _ := e.LastRun()
		nextrun, _ := e.NextRun()

		scheds = append(scheds, data.Crons{
			Group:   group,
			Action:  action,
			NextRun: nextrun,
			LastRan: lastrun,
		})
	}

	return c.JSON(http.StatusOK, scheds)
}

func GetCrons(c echo.Context) error {
	if !sessionValid(c) {
		return c.Redirect(http.StatusSeeOther, "/v1/pal/ui/login")
	}

	type crons struct {
		CronDesc string
		Group    string
		Action   string
		NextRun  string
		LastRan  string
	}

	scheds := []crons{}

	for _, e := range sched.Jobs() {
		lastrun, _ := e.LastRun()
		nextrun, _ := e.NextRun()

		scheds = append(scheds, crons{
			Group:    strings.Split(e.Name(), "/")[0],
			Action:   strings.Split(e.Name(), "/")[1],
			NextRun:  humanize.Time(nextrun),
			LastRan:  humanize.Time(lastrun),
			CronDesc: e.Tags()[0],
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
	if !authHeaderCheck(c.Request().Header) {
		return c.String(http.StatusUnauthorized, errorAuth)
	}

	key := c.QueryParam("key")

	if key == "" {
		return echo.NewHTTPError(http.StatusNotFound, "error key query param empty")
	}

	if len(config.GetConfigResponseHeaders()) > 0 {
		for _, v := range config.GetConfigResponseHeaders() {
			c.Response().Header().Set(v.Header, v.Value)
		}
	}

	val, err := db.DBC.Get(key)
	if err != nil {
		return c.String(http.StatusNotFound, "error vlaue not found with key: "+key)
	}

	return c.String(http.StatusOK, val)
}

func GetDBJSONDump(c echo.Context) error {
	if !authHeaderCheck(c.Request().Header) {
		return c.String(http.StatusUnauthorized, errorAuth)
	}

	if len(config.GetConfigResponseHeaders()) > 0 {
		for _, v := range config.GetConfigResponseHeaders() {
			c.Response().Header().Set(v.Header, v.Value)
		}
	}

	return c.JSON(http.StatusOK, db.DBC.Dump())
}

func PutDBPut(c echo.Context) error {
	if !authHeaderCheck(c.Request().Header) {
		return c.String(http.StatusUnauthorized, errorAuth)
	}

	key := c.QueryParam("key")

	if key == "" {
		return echo.NewHTTPError(http.StatusNotFound, "error key query param empty")
	}

	if len(config.GetConfigResponseHeaders()) > 0 {
		for _, v := range config.GetConfigResponseHeaders() {
			c.Response().Header().Set(v.Header, v.Value)
		}
	}

	bodyBytes, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "error reading request body in db put")
	}

	err = db.DBC.Put(key, string(bodyBytes))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "error db put for key: "+key)
	}

	return c.String(http.StatusCreated, "success")
}

func PostDBput(c echo.Context) error {
	if !sessionValid(c) {
		return c.Redirect(http.StatusSeeOther, "/v1/pal/ui/login")
	}

	key := c.FormValue("key")
	value := c.FormValue("value")

	if key == "" {
		return echo.NewHTTPError(http.StatusNotFound, "error key query param empty")
	}

	err := db.DBC.Put(key, value)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "error db put for key: "+key)
	}

	return c.Redirect(302, "/v1/pal/ui/db")
}

func DeleteDBDel(c echo.Context) error {
	if !authHeaderCheck(c.Request().Header) {
		return c.String(http.StatusUnauthorized, errorAuth)
	}

	key := c.QueryParam("key")
	if key == "" {
		return echo.NewHTTPError(http.StatusNotFound, "error key query param empty")
	}

	if len(config.GetConfigResponseHeaders()) > 0 {
		for _, v := range config.GetConfigResponseHeaders() {
			c.Response().Header().Set(v.Header, v.Value)
		}
	}

	err := db.DBC.Delete(key)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "error db put for key: "+key)
	}

	return c.String(http.StatusOK, "success")
}

func GetDBdelete(c echo.Context) error {
	if !sessionValid(c) {
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
	if !sessionValid(c) {
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

	return c.HTML(http.StatusOK, fmt.Sprintf("<!DOCTYPE html><html><head><meta http-equiv='refresh' content='3; url=/v1/pal/ui/files' /><title>Redirecting...</title></head><body><h2>Successfully uploaded %d files. You will be redirected to <a href='/v1/pal/ui/files'>/v1/pal/ui/files</a> in 3 seconds...</h2></body></html>", len(files)))

}

func GetLogout(c echo.Context) error {
	sess, err := session.Get("session", c)
	if err != nil {
		return c.HTML(http.StatusUnauthorized, `<!DOCTYPE html><html><head><meta http-equiv="refresh" content="3; url=/v1/pal/ui"><title>Redirecting...</title></head><body><h2>You will be redirected to /v1/pal/ui in 3 seconds...</h2></body></html>`)
	}
	sess.Options = &sessions.Options{
		Path:     "/v1/pal",
		MaxAge:   -1,
		Secure:   true,
		HttpOnly: true,
	}
	sess.Values["authenticated"] = false
	if err := sess.Save(c.Request(), c.Response()); err != nil {
		return c.HTML(http.StatusUnauthorized, `<!DOCTYPE html><html><head><meta http-equiv="refresh" content="3; url=/v1/pal/ui"><title>Redirecting...</title></head><body><h2>You will be redirected to /v1/pal/ui in 3 seconds...</h2></body></html>`)
	}

	return c.HTML(http.StatusUnauthorized, `<!DOCTYPE html><html><head><meta http-equiv="refresh" content="3; url=/v1/pal/ui"><title>Redirecting...</title></head><body><h2>You will be redirected to /v1/pal/ui in 3 seconds...</h2></body></html>`)
}

func GetRobots(c echo.Context) error {
	return c.String(http.StatusOK, `User-agent: *
Disallow: /`)
}

func GetDBPage(c echo.Context) error {
	if !sessionValid(c) {
		return c.Redirect(http.StatusSeeOther, "/v1/pal/ui/login")
	}

	uiData := struct {
		Dump          map[string]string
		Notifications int
	}{
		Dump:          db.DBC.Dump(),
		Notifications: len(db.DBC.GetNotifications("")),
	}

	return c.Render(http.StatusOK, "db.tmpl", uiData)
}

func GetActionsPage(c echo.Context) error {
	if !sessionValid(c) {
		return c.Redirect(http.StatusSeeOther, "/v1/pal/ui/login")
	}

	res := db.DBC.GetGroups()
	res2 := make(map[string][]data.ActionData)

	for group, groupData := range res {
		res2[group] = make([]data.ActionData, len(groupData))
		for i, action := range groupData {
			parsedTime, err := time.Parse(time.RFC3339, action.LastRan)
			if err == nil {
				action.LastRan = humanize.Time(parsedTime)
			}
			res2[group][i] = action
		}
	}

	tmpl, err := template.New("actions.tmpl").Funcs(template.FuncMap{
		"getData": func() map[string][]data.ActionData {
			return res2
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
	if !sessionValid(c) {
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

	resMap := db.DBC.GetGroups()

	uiData := struct {
		ActionMap     map[string]data.ActionData
		Notifications int
	}{}

	for _, e := range resMap[group] {
		if e.Action == action {
			uiData.ActionMap = map[string]data.ActionData{
				group: e,
			}
			uiData.Notifications = len(db.DBC.GetNotifications(""))
			return c.Render(http.StatusOK, "action.tmpl", uiData)
		}
	}

	return c.Render(http.StatusOK, "action.tmpl", uiData)
}

func GetAction(c echo.Context) error {
	if !sessionValid(c) {
		if !authHeaderCheck(c.Request().Header) {
			return c.JSON(http.StatusUnauthorized, data.GenericResponse{Err: "Unauthorized no valid session or auth header present."})
		}
	}

	group := c.QueryParam("group")
	if group == "" {
		return c.JSON(http.StatusBadRequest, data.GenericResponse{Err: errorGroup})
	}

	action := c.QueryParam("action")
	if action == "" {
		return c.JSON(http.StatusBadRequest, data.GenericResponse{Err: errorAction})
	}

	disable := c.QueryParam("disabled")
	if disable != "" {
		state := false

		if disable == "true" {
			state = true
		}

		condDisable(group, action, state)
		return c.JSON(http.StatusOK, data.GenericResponse{Message: fmt.Sprintf("changed action state %s/%s disabled to %s", group, action, disable)})
	}

	resMap := db.DBC.GetGroups()

	for _, e := range resMap[group] {
		if e.Action == action {
			return c.JSONPretty(http.StatusOK, e, "  ")
		}
	}

	return c.JSON(http.StatusOK, data.ActionData{})
}

func GetFilesPage(c echo.Context) error {
	if !sessionValid(c) {
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
			MaxAge:   3600,
			Secure:   true,
			HttpOnly: true,
		}
		sess.Values["authenticated"] = true
		if err := sess.Save(c.Request(), c.Response()); err != nil {
			return err
		}
		return c.Redirect(http.StatusSeeOther, "/v1/pal/ui")
	}
	return c.Redirect(302, "/v1/pal/ui/login")
}

func GetLoginPage(c echo.Context) error {
	return c.Render(http.StatusOK, "login.tmpl", nil)
}

func GetFilesDownload(c echo.Context) error {
	if !sessionValid(c) {
		return c.Redirect(http.StatusSeeOther, "/v1/pal/ui/login")
	}
	return c.File(config.GetConfigUI().UploadDir + "/" + c.Param("file"))
}

func GetFavicon(c echo.Context) error {
	iconStr := `<?xml version="1.0" encoding="UTF-8"?>
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

	return c.Blob(http.StatusOK, "image/svg+xml", []byte(iconStr))
}

func GetFilesDelete(c echo.Context) error {
	if !sessionValid(c) {
		return c.Redirect(http.StatusSeeOther, "/v1/pal/ui/login")
	}
	file := c.Param("file")
	err := os.Remove(config.GetConfigUI().UploadDir + "/" + file)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "error deleting file "+file)
	}
	return c.Redirect(http.StatusTemporaryRedirect, "/v1/pal/ui/files")
}

func RedirectUI(c echo.Context) error {
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

	return auth.(bool)
}

func cronTask(res data.ActionData) string {
	if getCond(res.Group, res.Action) {
		return "error action disabled"
	}

	cmdOrig := res.Cmd
	res.Cmd = cmdString(res, "", "")
	timeNow := utils.TimeNow(config.GetConfigStr("global_timezone"))
	cmdOutput, duration, err := utils.CmdRun(res, config.GetConfigStr("global_cmd_prefix"), config.GetConfigStr("global_working_dir"))
	res.Cmd = cmdOrig
	if err != nil {
		res.Status = "error"
		res.LastDuration = duration
		res.LastRan = timeNow
		if res.Output {
			res.LastOutput = cmdOutput
		}
		mergeGroup(res)
		return err.Error()
	}

	fmt.Printf("%s\n", fmt.Sprintf(`{"time":"%s","group":"%s","job_success":%t}`, timeNow, res.Group+"/"+res.Action, true))

	res.Status = "success"
	res.LastDuration = duration
	res.LastRan = timeNow
	if res.Output {
		res.LastOutput = cmdOutput
	}
	mergeGroup(res)
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

func putNotifications(notification data.Notification) error {
	notifications := db.DBC.GetNotifications("")

	if len(notifications) > config.GetConfigInt("notifications_max") {
		notifications = notifications[1:]
	}

	notification.ID = uuid.NewString()
	notification.NotificationRcv = utils.TimeNow(config.GetConfigStr("global_timezone"))

	notifications = append([]data.Notification{notification}, notifications...)

	return db.DBC.PutNotifications(notifications)
}

func validateInput(input, inputValidate string) error {
	if inputValidate == "" {
		return nil
	}

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
