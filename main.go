package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net/http"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/go-playground/validator/v10"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo-contrib/session"
	echo "github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/marshyski/pal/config"
	"github.com/marshyski/pal/data"
	db "github.com/marshyski/pal/db"
	"github.com/marshyski/pal/routes"
	"github.com/marshyski/pal/ui"
	"github.com/marshyski/pal/utils"
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
	version    string
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
		actionsDir string
		timeoutInt int
	)

	flag.StringVar(&actionsDir, "d", "./actions", "Action definitions files directory location")
	flag.StringVar(&configFile, "c", "./pal.yml", "Set configuration file path location")
	flag.Usage = func() {
		fmt.Printf(`Usage: pal [options] <args>
  -c,	Set configuration file path location, default is ./pal.yml
  -d,	Set action definitions files directory location, default is ./actions

Example: pal -c ./pal.yml -d ./actions

Built On:       %s
Commit Hash:	%s
Version:        %s
Documentation:	https://github.com/marshyski/pal
`, builtOn, commitHash, version)
	}

	flag.Parse()

	// Setup Custom Configs
	err := config.InitConfig(configFile)
	if err != nil {
		log.Fatalln(err.Error())
	}

	groups := config.ReadConfig(actionsDir)

	for k, v := range groups {
		for i, e := range v {
			e.Group = k
			v[i] = e
		}
		groups[k] = v
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

	// Update old DB data with new values from actions yml files
	mergedGroups := utils.MergeGroups(dbc.GetGroups(), groups)

	err = db.DBC.PutGroups(mergedGroups)
	if err != nil {
		// TODO: DEBUG STATEMENT
		log.Println(err.Error())
	}

	e := echo.New()
	e.Debug = config.GetConfigBool("global_debug")
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

	if config.GetConfigBool("http_prometheus") {
		e.Use(echoprometheus.NewMiddleware("pal"))
		e.GET("/v1/pal/metrics", echoprometheus.NewHandler())
	}

	// Setup Non-UI Routes
	e.GET("/v1/pal/db/get", routes.GetDBGet)
	e.GET("/v1/pal/db/dump", routes.GetDBJSONDump)
	e.PUT("/v1/pal/db/put", routes.PutDBPut)
	e.DELETE("/v1/pal/db/delete", routes.DeleteDBDel)
	e.GET("/v1/pal/health", routes.GetHealth)
	e.GET("/v1/pal/crons", routes.GetCronsJSON)
	e.GET("/v1/pal/notifications", routes.GetNotifications)
	e.PUT("/v1/pal/notifications", routes.PutNotifications)
	e.GET("/v1/pal/run/:group/:action", routes.RunGroup)
	e.POST("/v1/pal/run/:group/:action", routes.RunGroup)
	e.GET("/v1/pal/cond/:group/:action", routes.GetCond)
	e.GET("/v1/pal/action", routes.GetAction)

	// Setup UI Routes Only If Basic Auth Isn't Empty
	if config.GetConfigUI().BasicAuth != "" && utils.FileExists(config.GetConfigUI().UploadDir) {
		uiFS, err := fs.Sub(ui.UIFiles, ".")
		if err != nil {
			log.Fatal(err)
		}

		// Setup The HTML Templates
		tmpl := template.Must(template.ParseFS(uiFS, "login.tmpl"))
		template.Must(tmpl.New("db.tmpl").ParseFS(uiFS, "db.tmpl"))
		template.Must(tmpl.New("crons.tmpl").ParseFS(uiFS, "crons.tmpl"))
		template.Must(tmpl.New("action.tmpl").ParseFS(uiFS, "action.tmpl"))
		template.Must(tmpl.New("notifications.tmpl").ParseFS(uiFS, "notifications.tmpl"))
		actionsFuncMap := template.FuncMap{
			"getData": func() map[string][]data.ActionData {
				return db.DBC.GetGroups()
			},
			"Notifications": func() int {
				return len(db.DBC.GetNotifications(""))
			},
		}
		template.Must(tmpl.New("actions.tmpl").Funcs(actionsFuncMap).ParseFS(uiFS, "actions.tmpl"))
		filesFuncMap := template.FuncMap{
			"fileSize": func(file fs.DirEntry) string {
				info, _ := file.Info()
				return humanize.Bytes(uint64(info.Size())) // #nosec G115
			},
			"fileModTime": func(file fs.DirEntry) string {
				info, _ := file.Info()
				return humanize.Time(info.ModTime())
			},
		}
		template.Must(tmpl.New("files.tmpl").Funcs(filesFuncMap).ParseFS(uiFS, "files.tmpl"))

		e.Renderer = &Template{
			templates: tmpl,
		}

		var store *sessions.CookieStore
		if config.GetConfigStr("http_session_secret") == "" {
			store = sessions.NewCookieStore([]byte(utils.GenSecret()))
		} else {
			store = sessions.NewCookieStore([]byte(config.GetConfigStr("http_session_secret")))
		}
		e.Use(session.Middleware(store))
		e.GET("/", routes.RedirectUI)
		e.GET("/v1/pal/ui/static/*", echo.WrapHandler(http.StripPrefix("/v1/pal/ui/static/", http.FileServer(http.FS(uiFS)))))
		e.GET("/favicon.ico", routes.GetFavicon)
		e.GET("/robots.txt", routes.GetRobots)
		e.GET("/v1/pal/ui", routes.GetActionsPage)
		e.GET("/v1/pal/ui/login", routes.GetLoginPage)
		e.POST("/v1/pal/ui/login", routes.PostLoginPage)
		e.GET("/v1/pal/ui/db", routes.GetDBPage)
		e.POST("/v1/pal/ui/db/put", routes.PostDBput)
		e.GET("/v1/pal/ui/db/delete", routes.GetDBdelete)
		e.GET("/v1/pal/ui/files", routes.GetFilesPage)
		e.POST("/v1/pal/ui/files/upload", routes.PostFilesUpload)
		e.GET("/v1/pal/ui/files/download/:file", routes.GetFilesDownload)
		e.GET("/v1/pal/ui/files/delete/:file", routes.GetFilesDelete)
		e.GET("/v1/pal/ui/notifications", routes.GetNotificationsPage)
		e.GET("/v1/pal/ui/crons", routes.GetCrons)
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
