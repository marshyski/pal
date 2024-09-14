package routes

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
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
	cmap "github.com/orcaman/concurrent-map"
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
	RouteMap  = cmap.New()
	Schedules *[]gocron.Job
	validate  = validator.New(validator.WithRequiredStructEnabled())
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

	res, _ := RouteMap.Get(group)
	resData := res.([]data.GroupData)

	for i, e := range resData {
		if e.Action == action {
			resData[i].Lock = lockState
			RouteMap.Set(group, resData)
			return
		}
	}
}

// logError time, error, id, uri fields
func logError(c echo.Context, e error) {
	fmt.Printf("%s\n", fmt.Sprintf(`{"time":"%s","error":"%s","id":"%s","uri":"%s"}`,
		time.Now().Format(time.RFC3339), e.Error(), c.Response().Header().Get(echo.HeaderXRequestID), c.Request().RequestURI))
}

// RunGroup is the main route for triggering a command
func RunGroup(c echo.Context) error {

	// check if group from URL is not empty
	group := c.Param("group")
	if group == "" {
		return c.String(http.StatusBadRequest, errorGroup)
	}

	// check if group is present in concurrent map
	if !RouteMap.Has(group) {
		return c.String(http.StatusBadRequest, errorGroup)
	}

	resMap, _ := RouteMap.Get(group)

	resData := resMap.([]data.GroupData)

	action := c.Param("action")
	if action == "" {
		return c.String(http.StatusBadRequest, errorAction)
	}

	actionPresent, actionData := utils.HasAction(action, resData)

	if !actionPresent {
		return c.String(http.StatusBadRequest, errorAction)
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
			switch actionData.ContentType {
			case "application/json":
				var jsonData interface{}
				err := json.Unmarshal([]byte(actionData.LastOutput), &jsonData)
				if err != nil {
					return c.String(http.StatusOK, actionData.LastOutput)
				}
				return c.JSON(http.StatusOK, jsonData)
			case "text/html":
				return c.HTML(http.StatusOK, actionData.LastOutput)
			default:
				return c.String(http.StatusOK, actionData.LastOutput)
			}
		}
		return c.String(http.StatusBadRequest, "error output not enabled")
	}

	cmd := actionData.Cmd

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
	cmd = strings.Join([]string{cmdArg, cmd}, " ")

	// Check if action wants to block the request to one
	if !actionData.Concurrent {
		if actionData.Lock {
			return c.String(http.StatusTooManyRequests, errorNotReady)
		}

		lock(group, action, true)
	}

	if actionData.Background {
		go func() {
			cmdOutput, err := utils.CmdRun(cmd)
			if err != nil {
				if !actionData.Concurrent {
					lock(group, action, false)
				}
				actionData.Status = "error"
				actionData.LastRan = time.Now().Format(time.RFC3339)
				actionData.LastOutput = err.Error()
				mergeGroup(group, actionData)
				logError(c, err)
				if actionData.OnError.Notification != "" {
					err := putNotifications(data.Notification{Group: group, Notification: actionData.OnError.Notification})
					if err != nil {
						logError(c, err)
					}
				}

			}
			actionData.Status = "success"
			actionData.LastRan = time.Now().Format(time.RFC3339)
			actionData.LastOutput = cmdOutput
			mergeGroup(group, actionData)
		}()

		if !actionData.Concurrent {
			lock(group, action, false)
		}

		time.Sleep(20 * time.Millisecond)

		return c.String(http.StatusOK, "running in background")
	}

	cmdOutput, err := utils.CmdRun(cmd)
	if err != nil {
		if !actionData.Concurrent {
			lock(group, action, false)
		}
		actionData.Status = "error"
		actionData.LastRan = time.Now().Format(time.RFC3339)
		actionData.LastOutput = err.Error()
		mergeGroup(group, actionData)
		logError(c, errors.New(errorScript+" "+err.Error()))
		if actionData.OnError.Notification != "" {
			err := putNotifications(data.Notification{Group: group, Notification: actionData.OnError.Notification})
			if err != nil {
				logError(c, err)
			}
		}
		return c.String(http.StatusInternalServerError, errorScript)
	}

	// Unlock if blocking is enabled
	if !actionData.Concurrent {
		lock(group, action, false)
	}

	if actionData.Output {
		actionData.Status = "success"
		actionData.LastRan = time.Now().Format(time.RFC3339)
		actionData.LastOutput = cmdOutput
		mergeGroup(group, actionData)
		switch actionData.ContentType {
		case "application/json":
			var jsonData interface{}
			err := json.Unmarshal([]byte(cmdOutput), &jsonData)
			if err != nil {
				return c.String(http.StatusOK, cmdOutput)
			}
			return c.JSON(http.StatusOK, jsonData)
		case "text/html":
			return c.HTML(http.StatusOK, cmdOutput)
		default:
			return c.String(http.StatusOK, cmdOutput)
		}
	}

	actionData.Status = "success"
	actionData.LastRan = time.Now().Format(time.RFC3339)
	// skip LastOutput
	mergeGroup(group, actionData)
	return c.String(http.StatusOK, "done")
}

func GetHealth(c echo.Context) error {
	return c.String(http.StatusOK, "ok")
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

func GetSchedulesJSON(c echo.Context) error {
	if !sessionValid(c) {
		if !authHeaderCheck(c.Request().Header) {
			return c.JSON(http.StatusUnauthorized, data.GenericResponse{Err: "Unauthorized no valid session or auth header present."})
		}
	}
	name := c.QueryParam("name")
	run := c.QueryParam("run")

	scheds := []data.Schedules{}

	for _, e := range *Schedules {
		if name == e.Name() && run == "now" {
			err := e.RunNow()
			if err != nil {
				return c.JSON(http.StatusInternalServerError, data.GenericResponse{Err: err.Error()})
			}
			return c.JSON(http.StatusOK, data.GenericResponse{Message: "running"})
		}

		lastrun, _ := e.LastRun()
		nextrun, _ := e.NextRun()

		scheds = append(scheds, data.Schedules{
			Name:    e.Name(),
			NextRun: nextrun,
			LastRan: lastrun,
		})
	}

	return c.JSON(http.StatusOK, scheds)
}

func GetSchedules(c echo.Context) error {
	if !sessionValid(c) {
		return c.Redirect(http.StatusSeeOther, "/v1/pal/ui/login")
	}

	scheds := []data.Schedules{}

	for _, e := range *Schedules {
		lastrun, _ := e.LastRun()
		nextrun, _ := e.NextRun()

		scheds = append(scheds, data.Schedules{
			Name:    e.Name(),
			NextRun: nextrun,
			LastRan: lastrun,
		})
	}

	return c.Render(http.StatusOK, "schedules.html", scheds)

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
	res, _ := RouteMap.Get("groups")
	return c.Render(http.StatusOK, "actions.html", res.(map[string][]data.GroupData))
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
		return c.String(http.StatusBadRequest, errorGroup)
	}

	res, _ := RouteMap.Get(("groups"))
	data2 := make(map[string]data.GroupData)

	for key, value := range res.(map[string][]data.GroupData) {
		for _, e := range value {
			if key == group && e.Action == action {
				data2[group] = e
				return c.Render(http.StatusOK, "action.html", data2)
			}
		}
	}

	return c.Render(http.StatusOK, "action.html", data2)
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

func cronTask(group string, res data.GroupData) string {
	cmdOutput, err := utils.CmdRun(res.Cmd)
	timeNow := time.Now().Format(time.RFC3339)
	if err != nil {
		res.Status = "error"
		res.LastRan = timeNow
		res.LastOutput = cmdOutput
		mergeGroup(group, res)
		return err.Error()
	}

	fmt.Printf("%s\n", fmt.Sprintf(`{"time":"%s","group":"%s","job_success":%t}`, timeNow, group+"/"+res.Action, true))

	res.Status = "success"
	res.LastRan = timeNow
	res.LastOutput = cmdOutput
	mergeGroup(group, res)
	return cmdOutput
}

func CronStart(r map[string][]data.GroupData) error {
	var sched gocron.Scheduler

	loc, err := time.LoadLocation(config.GetConfigStr("http_schedule_tz"))
	if err != nil {
		return err
	}

	sched, err = gocron.NewScheduler(gocron.WithLocation(loc))
	if err != nil {
		return err
	}

	for k, v := range r {
		for _, e := range v {
			if e.Schedule != "" {
				_, err := sched.NewJob(
					gocron.CronJob(e.Schedule, false),
					gocron.NewTask(cronTask, k, e),
					gocron.WithName(k+"/"+e.Action),
				)
				if err != nil {
					return err
				}
			}
		}
	}

	sched.Start()
	jobs := sched.Jobs()
	Schedules = &jobs

	return nil
}

func mergeGroup(group string, action data.GroupData) {
	groups, _ := RouteMap.Get("groups")
	groupsData := groups.(map[string][]data.GroupData)
	if v, ok := groupsData[group]; ok {
		for i, e := range v {
			if e.Action == action.Action {
				v[i] = action
				groupsData[group] = v
				RouteMap.Set("groups", groupsData)
				return
			}
		}
	}
}

func putNotifications(notification data.Notification) error {
	notifications := db.DBC.GetNotifications("")

	if len(notifications) > 100 {
		notifications = notifications[1:]
	}

	var timeStr string

	if config.GetConfigStr("http_schedule_tz") != "" {
		loc, err := time.LoadLocation(config.GetConfigStr("http_schedule_tz"))
		if err != nil {
			return err
		}

		timeStr = time.Now().In(loc).Format(time.RFC3339)
	} else {
		timeStr = time.Now().Format(time.RFC3339)
	}

	notification.NotificationRcv = timeStr

	notifications = append(notifications, notification)

	return db.DBC.PutNotifications(notifications)
}

func validateInput(input, inputValidate string) error {
	if inputValidate == "" {
		return nil
	}

	return validate.Var(input, inputValidate)
}
