package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	echo "github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/marshyski/pal/config"
	"github.com/marshyski/pal/data"
	db "github.com/marshyski/pal/db"
	"github.com/marshyski/pal/routes"
	"github.com/marshyski/pal/ui"
	"github.com/marshyski/pal/utils"
	"gopkg.in/yaml.v3"
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
	curves     = []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256}
	builtOn    string
	commitHash string
)

// Template
type Template struct {
	templates *template.Template
}

// Render
func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

type CustomValidator struct {
	validator *validator.Validate
}

func (cv *CustomValidator) Validate(i interface{}) error {
	if err := cv.validator.Struct(i); err != nil {
		// Optionally, you could return the error to give each route more control over the status code
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return nil
}

func main() {
	// Setup CLI Args
	var (
		configFile string
		actionFile string
		timeoutInt int
	)

	flag.StringVar(&actionFile, "a", "./pal-actions.yml", "Action definitions file location")
	flag.StringVar(&configFile, "c", "./pal.yml", "Set configuration file path location")
	flag.Usage = func() {
		fmt.Printf(`Usage: pal [options] <args>
  -a,	Set action definitions file path location, default is ./pal-actions.yml
  -c,	Set configuration file path location, default is ./pal.yml

Example: pal -a ./pal-actions.yml -c ./pal.yml

Built On:       %s
Commit Hash:	%s
Documentation:	https://github.com/marshyski/pal
`, builtOn, commitHash)
	}

	flag.Parse()

	// Setup Custom Configs
	err := config.InitConfig(configFile)
	if err != nil {
		log.Fatalln(err.Error())
	}

	defs, err := os.ReadFile(filepath.Clean(actionFile))
	if err != nil {
		log.Fatalln(err.Error())
	}

	groups := make(map[string][]data.GroupData)

	err = yaml.Unmarshal(defs, &groups)
	if err != nil {
		log.Fatalln(err.Error())
	}

	config.ValidateDefs(groups)

	routes.RouteMap.Set("groups", groups)

	for k, v := range groups {
		routes.RouteMap.Set(k, v)
	}

	// Setup Scheduled Cron Type Cmds
	err = routes.CronStart(groups)
	if err != nil {
		log.Fatalln(err.Error())
	}

	// Setup BadgerDB
	dbc, err := db.Open()
	if err != nil {
		log.Fatalln(err.Error())
	}

	defer dbc.Close()

	e := echo.New()
	e.HideBanner = true

	// Setup Echo Middlewares
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())
	e.Use(middleware.Logger())
	e.Use(middleware.Secure())
	e.Validator = &CustomValidator{validator: validator.New()}

	// Setup YAML HTTP Configurations
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

	// Setup The HTML Templates
	tmpl := template.Must(template.New("db.html").Parse(ui.DBpage))
	template.Must(tmpl.New("schedules.html").Parse(ui.SchedulesPage))
	template.Must(tmpl.New("actions.html").Parse(ui.ActionsPage))
	template.Must(tmpl.New("action.html").Parse(ui.ActionPage))
	template.Must(tmpl.New("notifications.html").Parse(ui.NotificationsPage))

	renderer := &Template{
		templates: tmpl,
	}
	e.Renderer = renderer

	// Setup Non-UI Routes
	e.GET("/v1/pal/db/get", routes.GetDBGet)
	e.GET("/v1/pal/db/dump", routes.GetDBJSONDump)
	e.PUT("/v1/pal/db/put", routes.PutDBPut)
	e.DELETE("/v1/pal/db/delete", routes.DeleteDBDel)
	e.GET("/v1/pal/health", routes.GetHealth)
	e.GET("/v1/pal/schedules", routes.GetSchedulesJSON)
	e.GET("/v1/pal/notifications", routes.GetNotifications)
	e.PUT("/v1/pal/notifications", routes.PutNotifications)
	e.GET("/v1/pal/run/:group/:action", routes.RunGroup)
	e.POST("/v1/pal/run/:group/:action", routes.RunGroup)
	e.GET("/v1/pal/cond/:group/:action", routes.GetCond)

	// Setup UI Routes Only If Basic Auth Isn't Empty
	if config.GetConfigUI().BasicAuth != "" && utils.FileExists(config.GetConfigUI().UploadDir) {
		var store *sessions.CookieStore
		if config.GetConfigStr("http_session_secret") == "" {
			store = sessions.NewCookieStore([]byte(utils.GenSecret()))
		} else {
			store = sessions.NewCookieStore([]byte(config.GetConfigStr("http_session_secret")))
		}
		e.Use(session.Middleware(store))
		e.GET("/robots.txt", routes.GetRobots)
		e.GET("/v1/pal/ui", routes.GetActionsPage)
		e.GET("/v1/pal/ui/login", routes.GetLoginPage)
		e.POST("/v1/pal/ui/login", routes.PostLoginPage)
		e.GET("/v1/pal/ui/main.css", routes.GetMainCSS)
		e.GET("/v1/pal/ui/main.js", routes.GetMainJS)
		e.GET("/v1/pal/ui/db", routes.GetDBPage)
		e.POST("/v1/pal/ui/db/put", routes.PostDBput)
		e.GET("/v1/pal/ui/db/delete", routes.GetDBdelete)
		e.GET("/v1/pal/ui/files", routes.GetFilesPage)
		e.POST("/v1/pal/ui/files/upload", routes.PostFilesUpload)
		e.GET("/v1/pal/ui/files/download/:file", routes.GetFilesDownload)
		e.GET("/v1/pal/ui/files/delete/:file", routes.GetFilesDelete)
		e.GET("/v1/pal/ui/notifications", routes.GetNotificationsPage)
		e.GET("/v1/pal/ui/schedules", routes.GetSchedules)
		e.GET("/v1/pal/ui/action/:group/:action", routes.GetActionPage)
		e.POST("/v1/pal/ui/action/:group/:action/run", routes.RunGroup)
		e.GET("/v1/pal/ui/action/:group/:action/run", routes.RunGroup)
		e.GET("/v1/pal/ui/logout", routes.GetLogout)
	}

	// Setup HTTP Server
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
