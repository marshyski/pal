package main

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	echo "github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	cmap "github.com/orcaman/concurrent-map"
	"github.com/perlogix/pal/config"
	db "github.com/perlogix/pal/db"
	"github.com/perlogix/pal/ui"
	"github.com/perlogix/pal/utils"
	"gopkg.in/yaml.v3"
)

const (
	errorAuth     = "error unauthorized"
	errorScript   = "error script fail"
	errorNotReady = "error not ready"
	errorTarget   = "error invalid target"
	errorResource = "error resource invalid"
	errorCmdEmpty = "error cmd is empty for target"
)

var (
	ciphers = []uint16{
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_RSA_WITH_AES_256_CBC_SHA,
	}
	curves    = []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256}
	configMap = cmap.New()
	dbc       = &db.DB{}
	store     = sessions.NewCookieStore([]byte(utils.GenSecret()))
)

type responseHeaders struct {
	Header string `yaml:"header"`
	Value  string `yaml:"value"`
}

// resourceData struct for target data of a resource
type resourceData struct {
	Background      bool              `yaml:"background"`
	Target          string            `yaml:"target"`
	Concurrent      bool              `yaml:"concurrent"`
	AuthHeader      string            `yaml:"auth_header"`
	Output          bool              `yaml:"output"`
	Cmd             string            `yaml:"cmd"`
	ResponseHeaders []responseHeaders `yaml:"response_headers"`
	ContentType     string            `yaml:"content_type"`
	Lock            bool
}

func dbAuthCheck(headers map[string][]string) bool {
	pass := false
	for k, v := range headers {
		header := strings.Join([]string{k, v[0]}, " ")
		if header == config.GetConfigStr("db_auth_header") {
			pass = true
		}
	}

	return pass
}

type Template struct {
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

// hasTarget verify target is not empty
func hasTarget(target string, resource []resourceData) (bool, resourceData) {

	for _, e := range resource {
		if e.Target == target {
			return true, e
		}
	}

	return false, resourceData{}
}

// getAuthHeader check if auth header is present and return header
func getAuthHeader(target resourceData) (bool, string) {

	if target.AuthHeader != "" {
		return true, target.AuthHeader
	}

	return false, ""
}

// getCmd returns cmd if not empty or return error
func getCmd(target resourceData) (string, error) {

	if target.Cmd != "" {
		return target.Cmd, nil
	}

	return "", errors.New(errorCmdEmpty)
}

// lock sets the lock for blocking requests until cmd has finished
func lock(resource, target string, lockState bool) {

	data, _ := configMap.Get(resource)
	resData := data.([]resourceData)

	for i, e := range resData {
		if e.Target == target {
			resData[i].Lock = lockState
			configMap.Set(resource, resData)
			return
		}
	}
}

// cmdRun runs a shell command or script and returns output with error
func cmdRun(cmd string) (string, error) {

	output, err := exec.Command("/bin/sh", "-c", cmd).Output()
	if err != nil {
		return "", err
	}

	return string(output), nil
}

// logError sets lock to unlocked and logs error
func logError(uri, e string) {
	t := time.Now()
	logMessage := fmt.Sprintf(`{"time":"%s","error":"%s","uri":"%s"}`, t.Format(time.RFC3339), e, uri)
	fmt.Println(logMessage)
}

// runResource is the main route for triggering a command
func runResource(c echo.Context) error {

	// check if resource from URL is not empty
	resource := c.Param("resource")
	if resource == "" {
		return c.String(http.StatusBadRequest, errorResource)
	}

	// check if resource is present in concurrent map
	if !configMap.Has(resource) {
		return c.String(http.StatusBadRequest, errorResource)
	}

	resMap, _ := configMap.Get(resource)

	resData := resMap.([]resourceData)

	uri := c.Request().RequestURI

	target := c.QueryParam("target")

	if target == "" {
		target = c.Param("target")
	}

	targetPresent, targetData := hasTarget(target, resData)

	// set custom headers
	if len(targetData.ResponseHeaders) > 0 {
		for _, v := range targetData.ResponseHeaders {
			c.Response().Header().Set(v.Header, v.Value)
		}
	}

	if !targetPresent {
		return c.String(http.StatusBadRequest, errorTarget)
	}

	auth, authHeader := getAuthHeader(targetData)

	// Check if auth header is present and if the header is correct
	if auth {
		if c.Param("target") == "" {
			pass := false
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
		if !sessionValid(c) {
			return c.Redirect(http.StatusSeeOther, "/v1/pal/ui/login")
		}
	}

	cmd, err := getCmd(targetData)
	if err != nil {
		logError(uri, err.Error())
		return c.String(http.StatusInternalServerError, errorCmdEmpty)
	}

	var arg string

	if c.Request().Method == "POST" {
		bodyBytes, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "error reading request body in post run")
		}
		arg = string(bodyBytes)
	} else {
		arg = c.QueryParam("arg")
	}

	// Prepare cmd prefix with argument variable to be passed into cmd
	cmdArg := strings.Join([]string{"export ARG='", arg, "';"}, "")

	// Build cmd with arg prefix
	cmd = strings.Join([]string{cmdArg, cmd}, " ")

	// Check if target wants to block the request to one
	if !targetData.Concurrent {
		if targetData.Lock {
			return c.String(http.StatusTooManyRequests, errorNotReady)
		}

		lock(resource, target, true)
	}

	if targetData.Background {
		go func() {
			_, err := cmdRun(cmd)
			if err != nil {
				if !targetData.Concurrent {
					lock(resource, target, false)
				}
				logError(uri, err.Error())
			}
		}()

		if !targetData.Concurrent {
			lock(resource, target, false)
		}

		return c.String(http.StatusOK, "running in background")
	}

	cmdOutput, err := cmdRun(cmd)
	if err != nil {
		if !targetData.Concurrent {
			lock(resource, target, false)
		}
		logError(uri, errorScript+" "+err.Error())
		return c.String(http.StatusInternalServerError, errorScript)
	}

	// Unlock if blocking is enabled
	if !targetData.Concurrent {
		lock(resource, target, false)
	}

	if targetData.Output {
		switch targetData.ContentType {
		case "application/json":
			var jsonData interface{}
			err := json.Unmarshal([]byte(cmdOutput), &jsonData)
			if err != nil {
				logError(uri, err.Error())
			}
			return c.JSON(http.StatusOK, jsonData)
		case "text/html":
			return c.HTML(http.StatusOK, cmdOutput)
		default:
			return c.String(http.StatusOK, cmdOutput)
		}
	}

	return c.String(http.StatusOK, "done")
}

func getHealth(c echo.Context) error {
	return c.String(http.StatusOK, "ok")
}

func getDBGet(c echo.Context) error {
	pass := dbAuthCheck(c.Request().Header)

	if !pass {
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

	val, err := dbc.Get(key)
	if err != nil {
		return c.String(http.StatusNotFound, "error vlaue not found with key: "+key)
	}

	return c.String(http.StatusOK, val)
}

func getDBJSONDump(c echo.Context) error {
	pass := dbAuthCheck(c.Request().Header)

	if !pass {
		return c.String(http.StatusUnauthorized, errorAuth)
	}

	if len(config.GetConfigResponseHeaders()) > 0 {
		for _, v := range config.GetConfigResponseHeaders() {
			c.Response().Header().Set(v.Header, v.Value)
		}
	}

	return c.JSON(http.StatusOK, dbc.Dump())
}

func putDBPut(c echo.Context) error {
	pass := dbAuthCheck(c.Request().Header)

	if !pass {
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

	err = dbc.Put(key, string(bodyBytes))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "error db put for key: "+key)
	}

	return c.String(http.StatusCreated, "success")
}

func postDBput(c echo.Context) error {
	if !sessionValid(c) {
		return c.Redirect(http.StatusSeeOther, "/v1/pal/ui/login")
	}

	key := c.FormValue("key")
	value := c.FormValue("value")

	if key == "" {
		return echo.NewHTTPError(http.StatusNotFound, "error key query param empty")
	}

	err := dbc.Put(key, value)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "error db put for key: "+key)
	}

	return c.Redirect(302, "/v1/pal/ui/db")
}

func deleteDBDel(c echo.Context) error {
	pass := dbAuthCheck(c.Request().Header)

	if !pass {
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

	err := dbc.Delete(key)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "error db put for key: "+key)
	}

	return c.String(http.StatusOK, "success")
}

func getDBdelete(c echo.Context) error {
	if !sessionValid(c) {
		return c.Redirect(http.StatusSeeOther, "/v1/pal/ui/login")
	}

	key := c.QueryParam("key")
	if key == "" {
		return echo.NewHTTPError(http.StatusNotFound, "error key query param empty")
	}

	err := dbc.Delete(key)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "error db put for key: "+key)
	}

	return c.Redirect(http.StatusTemporaryRedirect, "/v1/pal/ui/db")
}

func postFilesUpload(c echo.Context) error {
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

func getLogout(c echo.Context) error {
	sess, err := session.Get("session", c)
	if err != nil {
		return err
	}
	sess.Options = &sessions.Options{
		Path:     "/v1/pal",
		MaxAge:   -1,
		Secure:   true,
		HttpOnly: true,
	}
	sess.Values["authenticated"] = false
	if err := sess.Save(c.Request(), c.Response()); err != nil {
		return err
	}
	return c.HTML(http.StatusUnauthorized, `<!DOCTYPE html><html><head><meta http-equiv="refresh" content="3; url=/v1/pal/ui"><title>Redirecting...</title></head><body><h2>You will be redirected to /v1/pal/ui in 3 seconds...</h2></body></html>`)
}

func getRobots(c echo.Context) error {
	return c.String(http.StatusOK, `User-agent: *
Disallow: /`)
}

func getMainCSS(c echo.Context) error {
	return c.Blob(http.StatusOK, "text/css", []byte(ui.MainCSS))
}

func getMainJS(c echo.Context) error {
	return c.Blob(http.StatusOK, "text/javascript", []byte(ui.MainJS))
}

func getDBPage(c echo.Context) error {
	if !sessionValid(c) {
		return c.Redirect(http.StatusSeeOther, "/v1/pal/ui/login")
	}
	data := dbc.Dump()
	return c.Render(http.StatusOK, "db.html", data)
}

func getResourcesPage(c echo.Context) error {
	if !sessionValid(c) {
		return c.Redirect(http.StatusSeeOther, "/v1/pal/ui/login")
	}
	data, _ := configMap.Get("resources")
	return c.Render(http.StatusOK, "resources.html", data.(map[string][]resourceData))
}

func getResourcePage(c echo.Context) error {
	if !sessionValid(c) {
		return c.Redirect(http.StatusSeeOther, "/v1/pal/ui/login")
	}
	resource := c.Param("resource")
	if resource == "" {
		return c.String(http.StatusBadRequest, errorResource)
	}
	target := c.Param("target")
	if target == "" {
		return c.String(http.StatusBadRequest, errorResource)
	}

	data, _ := configMap.Get(("resources"))
	data2 := make(map[string]resourceData)

	for key, value := range data.(map[string][]resourceData) {
		for _, e := range value {
			if key == resource && e.Target == target {
				data2[resource] = e
				return c.Render(http.StatusOK, "resource.html", data2)
			}
		}
	}

	return c.Render(http.StatusOK, "resource.html", data2)
}

func getFilesPage(c echo.Context) error {
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

func postLoginPage(c echo.Context) error {
	if c.FormValue("username") == strings.Split(config.GetConfigUI().BasicAuth, " ")[0] && c.FormValue("password") == strings.Split(config.GetConfigUI().BasicAuth, " ")[1] {
		sess, err := session.Get("session", c)
		if err != nil {
			return err
		}
		sess.Options = &sessions.Options{
			Path:     "/v1/pal",
			MaxAge:   86400 * 1,
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

func getLoginPage(c echo.Context) error {
	return c.HTML(http.StatusOK, ui.LoginPage)
}

func getFilesDownload(c echo.Context) error {
	if !sessionValid(c) {
		return c.Redirect(http.StatusSeeOther, "/v1/pal/ui/login")
	}
	return c.File(config.GetConfigUI().UploadDir + "/" + c.Param("file"))
}

func getFilesDelete(c echo.Context) error {
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

func main() {

	var (
		configFile string
		defFile    string
		timeoutInt int
	)

	flag.StringVar(&defFile, "d", "./pal-defs.yml", "Definitions file location")
	flag.StringVar(&configFile, "c", "./pal.yml", "Configuration file location")
	flag.Parse()

	err := config.InitConfig(configFile)
	if err != nil {
		log.Fatalln(err.Error())
	}

	defs, err := os.ReadFile(filepath.Clean(defFile))
	if err != nil {
		log.Fatalln(err.Error())
	}

	resources := make(map[string][]resourceData)

	err = yaml.Unmarshal(defs, &resources)
	if err != nil {
		log.Fatalln(err.Error())
	}

	configMap.Set("resources", resources)

	for k, v := range resources {
		configMap.Set(k, v)
	}

	dbc, err = db.Open()
	if err != nil {
		log.Fatalln(err.Error())
	}

	defer dbc.Close()

	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())
	e.Use(middleware.Logger())
	e.Use(middleware.Secure())

	if config.GetConfigInt("http_timeout_min") > 0 {
		e.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
			Timeout: time.Duration(config.GetConfigInt("http_timeout_min")) * time.Minute,
		}))
	}

	if len(config.GetConfigArray("http_cors_allow_origins")) > 0 {
		e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
			AllowOrigins: config.GetConfigArray("http_cors_allow_origins"),
		}))
	}

	if config.GetConfigStr("http_body_limit") != "" {
		e.Use(middleware.BodyLimit(config.GetConfigStr("http_body_limit")))
	}

	tmpl := template.Must(template.New("db.html").Parse(ui.DBpage))

	template.Must(tmpl.New("resources.html").Parse(ui.ResourcesPage))
	template.Must(tmpl.New("resource.html").Parse(ui.ResourcePage))

	renderer := &Template{
		templates: tmpl,
	}
	e.Renderer = renderer

	e.GET("/v1/pal/db/get", getDBGet)
	e.GET("/v1/pal/db/dump", getDBJSONDump)
	e.PUT("/v1/pal/db/put", putDBPut)
	e.DELETE("/v1/pal/db/delete", deleteDBDel)
	e.GET("/v1/pal/health", getHealth)
	e.GET("/v1/pal/run/:resource", runResource)
	e.POST("/v1/pal/run/:resource", runResource)

	if config.GetConfigUI().BasicAuth != "" && utils.FileExists(config.GetConfigUI().UploadDir) {
		e.Use(session.Middleware(store))
		e.GET("/robots.txt", getRobots)
		e.GET("/v1/pal/ui", getResourcesPage)
		e.GET("/v1/pal/ui/login", getLoginPage)
		e.POST("/v1/pal/ui/login", postLoginPage)
		e.GET("/v1/pal/ui/main.css", getMainCSS)
		e.GET("/v1/pal/ui/main.js", getMainJS)
		e.GET("/v1/pal/ui/db", getDBPage)
		e.POST("/v1/pal/ui/db/put", postDBput)
		e.GET("/v1/pal/ui/db/delete", getDBdelete)
		e.GET("/v1/pal/ui/files", getFilesPage)
		e.POST("/v1/pal/ui/files/upload", postFilesUpload)
		e.GET("/v1/pal/ui/files/download/:file", getFilesDownload)
		e.GET("/v1/pal/ui/files/delete/:file", getFilesDelete)
		e.GET("/v1/pal/ui/resource/:resource/:target", getResourcePage)
		e.POST("/v1/pal/ui/resource/:resource/:target/run", runResource)
		e.GET("/v1/pal/ui/logout", getLogout)
	}

	http.DefaultTransport.(*http.Transport).MaxIdleConnsPerHost = 200
	http.DefaultTransport.(*http.Transport).MaxConnsPerHost = 200

	tlsCfg := &tls.Config{
		MinVersion:               tls.VersionTLS12,
		CurvePreferences:         curves,
		PreferServerCipherSuites: true,
		CipherSuites:             ciphers,
	}

	s := &http.Server{
		Addr:              config.GetConfigStr("http_listen"),
		Handler:           e.Server.Handler,
		ReadTimeout:       time.Duration(timeoutInt) * time.Minute,
		WriteTimeout:      time.Duration(timeoutInt) * time.Minute,
		IdleTimeout:       time.Duration(timeoutInt) * time.Minute,
		ReadHeaderTimeout: time.Duration(timeoutInt) * time.Minute,
		MaxHeaderBytes:    1 << 20,
		TLSConfig:         tlsCfg,
	}

	e.Logger.Fatal(s.ListenAndServeTLS(config.GetConfigStr("http_cert"), config.GetConfigStr("http_key")))
}
