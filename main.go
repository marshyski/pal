package main

import (
	"crypto/subtle"
	"crypto/tls"
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

	echo "github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	cmap "github.com/orcaman/concurrent-map"
	"github.com/perlogix/pal/config"
	db "github.com/perlogix/pal/db"
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
	Lock            bool
}

// type bcryptValid struct {
// 	Hash     string `json:"hash"`
// 	Password string `json:"password"`
// }

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
func cmdRun(cmd string) ([]byte, error) {

	output, err := exec.Command("/bin/sh", "-c", cmd).Output()
	if err != nil {
		return nil, err
	}

	return output, nil
}

// logError sets lock to unlocked and logs error
func logError(uri, e string) {
	t := time.Now()
	logMessage := fmt.Sprintf(`{"time":"%s","error":"%s","uri":"%s"}`, t.Format(time.RFC3339), e, uri)
	fmt.Println(logMessage)
}

// getResource is the main route for triggering a command
func getResource(c echo.Context) error {

	uri := c.Request().RequestURI

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

	target := c.QueryParam("target")

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

	cmd, err := getCmd(targetData)
	if err != nil {
		logError(uri, err.Error())
		return c.String(http.StatusInternalServerError, errorCmdEmpty)
	}

	// Prepare cmd prefix with argument variable to be passed into cmd
	cmdArg := strings.Join([]string{"export ARG=", c.QueryParam("arg"), ";"}, "")

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
		return c.String(http.StatusOK, string(cmdOutput))
	}

	return c.String(http.StatusOK, "done")
}

func getHealth(c echo.Context) error {
	return c.String(http.StatusOK, "ok")
}

func skipAuth(c echo.Context) bool {
	return c.Path() != "/v1/pal/upload"
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

// func getBcrypt(c echo.Context) error {
// 	bodyBytes, err := io.ReadAll(c.Request().Body)
// 	if err != nil {
// 		return echo.NewHTTPError(http.StatusBadRequest, "error reading request body in getBcrypt")
// 	}

// 	// Generate the bcrypt hash
// 	hash, err := bcrypt.GenerateFromPassword(bodyBytes, bcrypt.DefaultCost)
// 	if err != nil {
// 		return echo.NewHTTPError(http.StatusInternalServerError, "error generating hash")
// 	}

// 	return c.String(http.StatusOK, string(hash))
// }

// func postBcrypt(c echo.Context) error {
// 	var req bcryptValid
// 	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
// 		return echo.NewHTTPError(http.StatusBadRequest, "error invalid JSON request")
// 	}

// 	// Compare the password with the hash
// 	err := bcrypt.CompareHashAndPassword([]byte(req.Hash), []byte(req.Password))
// 	if err != nil {
// 		return c.String(http.StatusBadRequest, "invalid")
// 	}

// 	return c.String(http.StatusOK, "valid")
// }

func postUpload(c echo.Context) error {
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
		dst, err := os.Create(config.GetConfigUpload().Dir + "/" + file.Filename)
		if err != nil {
			return err
		}
		defer dst.Close()

		// Copy
		if _, err = io.Copy(dst, src); err != nil {
			return err
		}

	}

	return c.HTML(http.StatusOK, fmt.Sprintf("<p>Uploaded successfully %d files. <a href='/v1/pal/upload'>Click here to go back.</a></p>", len(files)))

}

func getUpload(c echo.Context) error {
	tmplString := `
<!DOCTYPE html>
<html>
<head>
<title>Upload</title>
<style>
body {
  font-family: sans-serif;
  padding: 20px;
}
h1 {
  margin-bottom: 15px;
}
form {
  margin-bottom: 30px;
}
input[type="file"] {
  margin-bottom: 10px;
}
ul {
  list-style: none;
  padding: 0;
}
li {
  margin-bottom: 5px;
}
a {
  text-decoration: none;
  color: #007bff;
}
</style>
</head>
<body>
<h1>Upload</h1>
<form action="/v1/pal/upload" method="post" enctype="multipart/form-data">
    <input type="file" name="files" multiple><br><br>
    <input type="submit" value="Submit">
</form>
<h1>Directory Listing</h1>
<ul>
{{range .Files}}
	<li>📄 <a href="/v1/pal/upload/{{.Name}}">{{.Name}}</a></li>
{{end}}
</ul>
</body>
</html>
`

	// Parse the template from the string
	tmpl := template.Must(template.New("directoryListing").Parse(tmplString))

	dirPath := config.GetConfigUpload().Dir

	// Read directory contents
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Error reading directory: "+dirPath)
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

	for k, v := range resources {
		configMap.Set(k, v)
	}

	dbc, err = db.Open()
	if err != nil {
		log.Fatalln(err.Error())
	}

	defer dbc.Close()

	timeoutInt = config.GetConfigInt("http_timeout_min")

	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())
	e.Use(middleware.Logger())
	e.Use(middleware.BodyLimit(config.GetConfigStr("http_body_limit")))
	// e.Use(middleware.CORSWithConfig{middleware.CORSConfig{
	// 	AllowOrigins: config.GetConfigArray(),
	// }))

	e.Use(middleware.Secure())
	e.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		Timeout: time.Duration(timeoutInt) * time.Minute,
	}))

	e.GET("/v1/pal/db/get", getDBGet)
	e.PUT("/v1/pal/db/put", putDBPut)
	e.DELETE("/v1/pal/db/delete", deleteDBDel)
	// e.POST("/v1/pal/bcrypt/gen", getBcrypt)
	// e.POST("/v1/pal/bcrypt/compare", postBcrypt)
	e.GET("/v1/pal/health", getHealth)
	e.GET("/v1/pal/run/:resource", getResource)

	if config.GetConfigUpload().Enable {
		if config.GetConfigUpload().BasicAuth != "" {
			e.Use(middleware.BasicAuthWithConfig(middleware.BasicAuthConfig{
				// Skip authentication for some routes that do not require authentication
				Skipper: skipAuth,
				Validator: func(username, password string, c echo.Context) (bool, error) {
					// Be careful to use constant time comparison to prevent timing attacks
					if subtle.ConstantTimeCompare([]byte(strings.ToLower(username)), []byte(strings.ToLower(strings.Split(config.GetConfigUpload().BasicAuth, " ")[0]))) == 1 &&
						subtle.ConstantTimeCompare([]byte(password), []byte(strings.Split(config.GetConfigUpload().BasicAuth, " ")[1])) == 1 {
						return true, nil
					}
					return false, nil
				},
			}))
		} else {
			log.Println("WARNING upload isn't protected by BasicAuth")
		}
		e.GET("/v1/pal/upload", getUpload)
		e.Static("/v1/pal/upload", config.GetConfigUpload().Dir)
		e.POST("/v1/pal/upload", postUpload)
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
