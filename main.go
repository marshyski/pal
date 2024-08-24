package main

import (
	"crypto/tls"
	"errors"
	"flag"
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
	curves      = []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256}
	resourceMap = cmap.New()
)

// resourceData struct for target data of a resource
type resourceData struct {
	Background bool   `yaml:"background"`
	Target     string `yaml:"target"`
	Concurrent bool   `yaml:"concurrent"`
	AuthHeader string `yaml:"auth_header"`
	Output     bool   `yaml:"output"`
	Cmd        string `yaml:"cmd"`
	Lock       bool
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

	data, _ := resourceMap.Get(resource)
	resData := data.([]resourceData)

	for i, e := range resData {
		if e.Target == target {
			resData[i].Lock = lockState
			resourceMap.Set(resource, resData)
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
func logError(resource, target, e string) {

	lock(resource, target, false)
	log.Println(resource, target, e)
}

// getResource is the main route for triggering a command
func getResource(c echo.Context) error {

	// check if resource from URL is not empty
	resource := c.Param("resource")
	if resource == "" {
		return c.String(http.StatusBadRequest, errorResource)
	}

	// check if resource is present in concurrent map
	if !resourceMap.Has(resource) {
		return c.String(http.StatusBadRequest, errorResource)
	}

	resMap, _ := resourceMap.Get(resource)

	resData := resMap.([]resourceData)

	target := c.QueryParam("target")

	targetPresent, targetData := hasTarget(target, resData)

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
		logError(resource, target, err.Error())
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
				logError(resource, target, err.Error())
			}
		}()

		if !targetData.Concurrent {
			lock(resource, target, false)
		}

		return c.String(http.StatusOK, "running in background")
	}

	cmdOutput, err := cmdRun(cmd)
	if err != nil {
		logError(resource, target, errorScript+" "+err.Error())
		if !targetData.Concurrent {
			lock(resource, target, false)
		}
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

func main() {

	var (
		configFile string
		listenAddr string
		timeoutInt int
	)

	flag.StringVar(&configFile, "c", "./pal.yml", "Configuration file location")
	flag.StringVar(&listenAddr, "l", "127.0.0.1:8443", "Set listening address and port")
	flag.IntVar(&timeoutInt, "t", 10, "Set HTTP timeout by minutes")
	flag.Parse()

	deployFile, err := os.ReadFile(filepath.Clean(configFile))
	if err != nil {
		log.Fatalln(err.Error())
	}

	resources := make(map[string][]resourceData)

	err = yaml.Unmarshal(deployFile, &resources)
	if err != nil {
		log.Fatalln(err.Error())
	}

	for k, v := range resources {
		resourceMap.Set(k, v)
	}

	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Recover())
	e.Use(middleware.Secure())
	e.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		Timeout: time.Duration(timeoutInt) * time.Minute,
	}))

	e.GET("/v1/pal/:resource", getResource)

	http.DefaultTransport.(*http.Transport).MaxIdleConnsPerHost = 200
	http.DefaultTransport.(*http.Transport).MaxConnsPerHost = 200

	tlsCfg := &tls.Config{
		MinVersion:               tls.VersionTLS12,
		CurvePreferences:         curves,
		PreferServerCipherSuites: true,
		CipherSuites:             ciphers,
	}

	s := &http.Server{
		Addr:              listenAddr,
		Handler:           e.Server.Handler,
		ReadTimeout:       time.Duration(timeoutInt) * time.Minute,
		WriteTimeout:      time.Duration(timeoutInt) * time.Minute,
		IdleTimeout:       time.Duration(timeoutInt) * time.Minute,
		ReadHeaderTimeout: time.Duration(timeoutInt) * time.Minute,
		MaxHeaderBytes:    1 << 20,
		TLSConfig:         tlsCfg,
	}

	e.Logger.Fatal(s.ListenAndServeTLS("./localhost.pem", "./localhost.key"))
}
