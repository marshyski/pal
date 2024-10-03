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
	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	echo "github.com/labstack/echo/v4"
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
					if e.Cron != "" && validateInput(e.Cron, "cron") == nil {
						_, err := sched.NewJob(
							gocron.CronJob(e.Cron, false),
							gocron.NewTask(cronTask, resData[group][i]),
							gocron.WithName(group+"/"+action),
							gocron.WithTags(group+action),
						)

						if err != nil {
							// TODOD: log error
							return
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
func logError(c echo.Context, e error) {
	fmt.Printf("%s\n", fmt.Sprintf(`{"time":"%s","error":"%s","id":"%s","uri":"%s"}`,
		utils.TimeNow(config.GetConfigStr("global_timezone")), e.Error(), c.Response().Header().Get(echo.HeaderXRequestID), c.Request().RequestURI))
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

	// Prepare cmd prefix with inputument variable to be passed into cmd
	cmdArg := strings.Join([]string{"export INPUT='", input, "';"}, "")

	// Build cmd with input prefix
	cmd := strings.Join([]string{cmdArg, cmdOrig}, " ")
	actionData.Cmd = cmd

	// Check if action wants to block the request to one
	if !actionData.Concurrent {
		if actionData.Lock {
			return c.String(http.StatusTooManyRequests, errorNotReady)
		}

		lock(group, action, true)
	}

	if actionData.Background {
		go func() {
			cmdOutput, duration, err := utils.CmdRun(actionData, config.GetConfigStr("global_cmd_prefix"))
			if err != nil {
				if !actionData.Concurrent {
					lock(group, action, false)
				}
				actionData.Cmd = cmdOrig
				actionData.Status = "error"
				actionData.LastDuration = duration
				actionData.LastRan = utils.TimeNow(config.GetConfigStr("global_timezone"))
				if actionData.Output {
					actionData.LastOutput = err.Error()
				}
				mergeGroup(actionData)
				logError(c, err)
				if actionData.OnError.Notification != "" {
					err := putNotifications(data.Notification{Group: group, Notification: actionData.OnError.Notification})
					if err != nil {
						logError(c, err)
					}
				}

			}
			actionData.Cmd = cmdOrig
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

		time.Sleep(20 * time.Millisecond)

		return c.String(http.StatusOK, "running in background")
	}

	cmdOutput, duration, err := utils.CmdRun(actionData, config.GetConfigStr("global_cmd_prefix"))
	if err != nil {
		if !actionData.Concurrent {
			lock(group, action, false)
		}
		actionData.Cmd = cmdOrig
		actionData.Status = "error"
		actionData.LastDuration = duration
		actionData.LastRan = utils.TimeNow(config.GetConfigStr("global_timezone"))
		if actionData.Output {
			actionData.LastOutput = err.Error()
		}
		mergeGroup(actionData)
		logError(c, errors.New(errorScript+" "+err.Error()))
		if actionData.OnError.Notification != "" {
			err := putNotifications(data.Notification{Group: group, Notification: actionData.OnError.Notification})
			if err != nil {
				logError(c, err)
			}
		}
		return c.String(http.StatusInternalServerError, err.Error())
	}

	// Unlock if blocking is enabled
	if !actionData.Concurrent {
		lock(group, action, false)
	}

	if actionData.Output {
		actionData.Cmd = cmdOrig
		actionData.Status = "success"
		actionData.LastDuration = duration
		actionData.LastRan = utils.TimeNow(config.GetConfigStr("global_timezone"))
		if actionData.Output {
			actionData.LastOutput = cmdOutput
		}
		mergeGroup(actionData)
		return c.String(http.StatusOK, cmdOutput)
	}

	actionData.Cmd = cmdOrig
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
		if !authHeaderCheck(c.Request().Header) {
			return c.JSON(http.StatusUnauthorized, data.GenericResponse{Err: "Unauthorized no valid session or auth header present."})
		}
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

	// delete notification from slice
	if notificationID != "" {

		var notifications []data.Notification
		for _, e := range db.DBC.GetNotifications("") {
			if e.NotificationRcv != notificationID {
				notifications = append(notifications, e)
			}
		}

		err := db.DBC.PutNotifications(notifications)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, data.GenericResponse{Err: err.Error()})
		}
		return c.Render(http.StatusOK, "notifications.html", notifications)
	}

	return c.Render(http.StatusOK, "notifications.html", db.DBC.GetNotifications(""))

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

	scheds := []data.Crons{}

	for _, e := range sched.Jobs() {
		lastrun, _ := e.LastRun()
		nextrun, _ := e.NextRun()

		scheds = append(scheds, data.Crons{
			Group:   strings.Split(e.Name(), "/")[0],
			Action:  strings.Split(e.Name(), "/")[1],
			NextRun: nextrun,
			LastRan: lastrun,
		})
	}

	return c.Render(http.StatusOK, "crons.html", scheds)

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

	return c.HTML(http.StatusOK, fmt.Sprintf("<!DOCTYPE html><html><head><meta http-equiv='refresh' content='5; url=/v1/pal/ui/files' /><title>Redirecting...</title></head><body><h2>Successfully uploaded %d files. You will be redirected to <a href='/v1/pal/ui/files'>/v1/pal/ui/files</a> in 5 seconds...</h2></body></html>", len(files)))

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

func GetMainCSS(c echo.Context) error {
	return c.Blob(http.StatusOK, "text/css", []byte(ui.MainCSS))
}

func GetMainJS(c echo.Context) error {
	return c.Blob(http.StatusOK, "text/javascript", []byte(ui.MainJS))
}

func GetDBPage(c echo.Context) error {
	if !sessionValid(c) {
		return c.Redirect(http.StatusSeeOther, "/v1/pal/ui/login")
	}
	data := db.DBC.Dump()
	return c.Render(http.StatusOK, "db.html", data)
}

func GetActionsPage(c echo.Context) error {
	if !sessionValid(c) {
		return c.Redirect(http.StatusSeeOther, "/v1/pal/ui/login")
	}

	res := db.DBC.GetGroups()
	res2 := make(map[string][]data.ActionData)

	for group, groupData := range res {
		res2[group] = make([]data.ActionData, len(groupData))
		for i, data := range groupData {
			parsedTime, err := time.Parse(time.RFC3339, data.LastRan)
			if err == nil {
				data.LastRan = humanize.Time(parsedTime)
			}
			res2[group][i] = data
		}
	}

	funcMap := template.FuncMap{
		"getData": func() map[string][]data.ActionData {
			return res2
		},
	}

	return template.Must(template.New("actions.html").Funcs(funcMap).Parse(ui.ActionsPage)).Execute(c.Response(), nil)
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

	for _, e := range resMap[group] {
		if e.Action == action {
			return c.Render(http.StatusOK, "action.html", map[string]data.ActionData{
				group: e,
			})
		}
	}

	return c.Render(http.StatusOK, "action.html", map[string]data.ActionData{})
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
	tmpl := template.Must(template.New("directoryListing").Funcs(template.FuncMap{
		"fileSize": func(file fs.DirEntry) string {
			info, _ := file.Info()
			return humanize.Bytes(uint64(info.Size()))
		},
		"fileModTime": func(file fs.DirEntry) string {
			info, _ := file.Info()
			return humanize.Time(info.ModTime())
		},
	}).Parse(ui.FilesPage))

	dirPath := config.GetConfigUI().UploadDir

	// Read directory contents
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "error reading directory: "+dirPath)
	}

	// Prepare data for the template
	data := struct {
		Files []fs.DirEntry
	}{
		Files: files,
	}

	// Render the template to the response
	return tmpl.Execute(c.Response(), data)
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
	return c.HTML(http.StatusOK, ui.LoginPage)
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

	cmdOutput, duration, err := utils.CmdRun(res, config.GetConfigStr("global_cmd_prefix"))
	timeNow := utils.TimeNow(config.GetConfigStr("global_timezone"))
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
			if e.Cron != "" && validateInput(e.Cron, "cron") == nil {
				_, err := sched.NewJob(
					gocron.CronJob(e.Cron, false),
					gocron.NewTask(cronTask, e),
					gocron.WithName(k+"/"+e.Action),
					gocron.WithTags(k+e.Action),
				)
				if err != nil {
					return err
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
