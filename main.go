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

package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
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
	builtOn    string
	commitHash string
	version    string
)

func getCiphers() []uint16 {
	return []uint16{
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_RSA_WITH_AES_256_CBC_SHA,
	}
}

func getTLScurves() []tls.CurveID {
	return []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256}
}

// Template
type Template struct {
	templates *template.Template
}

// Render
func (t *Template) Render(w io.Writer, name string, data interface{}, _ echo.Context) error {
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
		configFile      string
		actionsDir      string
		validateActions bool
		timeoutInt      int
	)

	flag.StringVar(&actionsDir, "d", "./actions", "Action definitions files directory location")
	flag.StringVar(&configFile, "c", "./pal.yml", "Set configuration file path location")
	flag.BoolVar(&validateActions, "v", false, "Validate action YML files and exit")
	flag.Usage = func() {
		fmt.Fprintf(os.Stdout, `Usage: pal [options] <args>
  -c,	Set configuration file path location, default is ./pal.yml
  -d,	Set action definitions files directory location, default is ./actions
  -v,   Validate action YML files and exit, default is false

Example: pal -c ./pal.yml -d ./actions
	 pal -d ./actions -v

Built On:       %s
Commit Hash:	%s
Version:        %s (YYYY.mm.dd)
Documentation:	https://github.com/marshyski/pal
`, builtOn, commitHash, version)
	}

	flag.Parse()

	// Setup Custom Configs
	err := config.InitConfig(configFile)
	if err != nil {
		log.Println("error with server config file: "+configFile, err.Error())
	}

	config.SetActionsDir(actionsDir)
	config.SetConfigFile(configFile)
	config.SetVersion(version)

	groups := config.ReadConfig(actionsDir)

	if validateActions {
		log.Println("Actions validated")
		os.Exit(0)
	}

	// keep need it twice for init/now and ReloadActions
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
		defer log.Fatalln(err.Error())
	}

	// Setup BadgerDB
	dbc, err := db.Open()
	if err != nil {
		log.Println(err.Error())
	}

	defer dbc.Close()
	if err != nil {
		defer os.Exit(1)
	}

	err = routes.ReloadActions(groups)
	if err != nil {
		log.Println("error reloading actions")
	}
	config.SetActionsReload()

	e := echo.New()
	e.Debug = config.GetConfigBool("global_debug")
	e.HideBanner = true

	// Setup Echo Middlewares
	e.Pre(middleware.HTTPSRedirect())
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
	e.GET("/v1/pal/actions", routes.GetActions)
	e.GET("/v1/pal/action", routes.GetAction)

	// Setup UI Routes Only If Basic Auth Isn't Empty
	if config.GetConfigUI().BasicAuth != "" && utils.FileExists(config.GetConfigUI().UploadDir) {
		uiFS, err := fs.Sub(ui.UIFiles, ".")
		if err != nil {
			defer log.Fatal(err)
		}

		// Setup The HTML Templates
		tmpl := template.Must(template.ParseFS(uiFS, "login.tmpl"))
		template.Must(tmpl.New("db.tmpl").ParseFS(uiFS, "db.tmpl"))
		template.Must(tmpl.New("crons.tmpl").ParseFS(uiFS, "crons.tmpl"))
		template.Must(tmpl.New("action.tmpl").ParseFS(uiFS, "action.tmpl"))
		template.Must(tmpl.New("system.tmpl").ParseFS(uiFS, "system.tmpl"))
		template.Must(tmpl.New("notifications.tmpl").ParseFS(uiFS, "notifications.tmpl"))
		actionsFuncMap := template.FuncMap{
			"getData": func() map[string][]data.ActionData {
				return db.DBC.GetGroups()
			},
			"Username": func() string {
				return ""
			},
			"Refresh": func() string {
				return "off"
			},
			"TimeNow": func() string {
				return utils.TimeNow(config.GetConfigStr("global_timezone"))
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
		e.GET("/v1/pal/cond/:group/:action", routes.GetCond)
		e.GET("/v1/pal/ui", routes.GetActionsPage)
		e.GET("/v1/pal/ui/login", routes.GetLoginPage)
		e.POST("/v1/pal/ui/login", routes.PostLoginPage)
		e.GET("/v1/pal/ui/system", routes.GetSystemPage)
		e.GET("/v1/pal/ui/refresh", routes.GetRefreshPage)
		e.GET("/v1/pal/ui/system/reload", routes.GetReloadActions)
		e.GET("/v1/pal/ui/db", routes.GetDBPage)
		e.POST("/v1/pal/ui/db/put", routes.PostDBput)
		e.GET("/v1/pal/ui/db/delete", routes.GetDBdelete)
		e.GET("/v1/pal/ui/files", routes.GetFilesPage)
		e.POST("/v1/pal/ui/files/upload", routes.PostFilesUpload)
		e.GET("/v1/pal/ui/files/download/:file", routes.GetFilesDownload)
		e.GET("/v1/pal/ui/files/delete/:file", routes.GetFilesDelete)
		e.GET("/v1/pal/ui/notifications", routes.GetNotificationsPage)
		e.GET("/v1/pal/ui/notifications/delete", routes.GetDeleteNotifications)
		e.GET("/v1/pal/ui/crons", routes.GetCrons)
		e.GET("/v1/pal/ui/action/:group/:action", routes.GetActionPage)
		e.POST("/v1/pal/ui/action/:group/:action/run", routes.RunGroup)
		e.GET("/v1/pal/ui/action/:group/:action/run", routes.RunGroup)
		e.GET("/v1/pal/ui/action/:group/:action/reset_runs", routes.GetResetAction)
		e.GET("/v1/pal/ui/logout", routes.GetLogout)
	}

	// Setup HTTP Server
	transport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		fmt.Fprintf(os.Stderr, "error getting transport")
		defer os.Exit(1)
	}

	transport.MaxIdleConnsPerHost = 200
	transport.MaxConnsPerHost = 200

	tlsCfg := &tls.Config{
		MinVersion:               tls.VersionTLS12,
		CurvePreferences:         getTLScurves(),
		PreferServerCipherSuites: true,
		CipherSuites:             getCiphers(),
	}

	tcpVer := "tcp4"
	if config.GetConfigBool("http_ipv6") {
		tcpVer = "tcp6"
	}

	port, err := strconv.Atoi(strings.Split(config.GetConfigStr("http_listen"), ":")[1])
	if err != nil {
		e.Logger.Fatal(err)
	}

	listener, err := net.ListenTCP(tcpVer, &net.TCPAddr{
		IP:   net.ParseIP(strings.Split(config.GetConfigStr("http_listen"), ":")[0]),
		Port: port,
	})
	if err != nil {
		e.Logger.Fatal(err)
	}

	s := &http.Server{
		Addr:              config.GetConfigStr("http_listen"),
		Handler:           e.Server.Handler,
		ReadTimeout:       time.Duration(timeoutInt) * time.Minute,
		WriteTimeout:      time.Duration(timeoutInt) * time.Minute,
		IdleTimeout:       time.Duration(timeoutInt) * time.Minute,
		ReadHeaderTimeout: time.Duration(timeoutInt) * time.Minute,
		TLSConfig:         tlsCfg,
	}

	e.Logger.Fatal(s.ServeTLS(listener, config.GetConfigStr("http_cert"), config.GetConfigStr("http_key")))
}
