package config

import (
	"errors"
	"log"
	"os"
	"path/filepath"

	"github.com/go-playground/validator/v10"
	"github.com/marshyski/pal/data"
	"github.com/marshyski/pal/utils"
	cmap "github.com/orcaman/concurrent-map"
	"gopkg.in/yaml.v3"
)

var (
	configMap = cmap.New()
)

func validateDefs(res map[string][]data.ActionData) bool {
	validate := validator.New(validator.WithRequiredStructEnabled())

	for _, v := range res {
		for _, e := range v {
			err := validate.Struct(e)
			if err != nil {
				log.Println(err)
				return false
			}
		}
	}

	return true
}

func ReadConfig(dir string) map[string][]data.ActionData {
	files, err := os.ReadDir(dir)
	if err != nil {
		log.Fatalln("Error reading directory:", err)
	}

	groups := make(map[string][]data.ActionData)

	for _, file := range files {
		if filepath.Ext(file.Name()) == ".yml" {
			fileLoc := filepath.Clean(dir + "/" + file.Name())
			defs, err := os.ReadFile(fileLoc)
			if err != nil {
				log.Println("Error reading file:", err)
				continue // Skip to the next file
			}

			var groupData map[string][]data.ActionData
			err = yaml.Unmarshal(defs, &groupData)
			if err != nil {
				log.Println("Error unmarshaling YAML:", err)
				continue
			}

			if validateDefs(groupData) {
				for k, v := range groupData {
					groups[k] = v
				}
			}
		}
	}
	return groups
}

func InitConfig(location string) error {
	if !utils.FileExists(location) {
		return errors.New("error file does not exist: " + location)
	}

	configFile, err := os.ReadFile(filepath.Clean(location))
	if err != nil {
		return err
	}

	config := &data.Config{}

	err = yaml.Unmarshal(configFile, config)
	if err != nil {
		return err
	}

	// Validate the configuration
	validate := validator.New(validator.WithRequiredStructEnabled())

	err = validate.Struct(config)
	if err != nil {
		log.Println(err)
		log.Fatalln("error panic " + location + " is invalid")
	}

	var upload_dir string
	upload_dir, err = filepath.Abs(config.HTTP.UploadDir)
	if err != nil {
		upload_dir = ""
	}

	configMap.Set("global_debug", config.Global.Debug)
	configMap.Set("http_prometheus", config.HTTP.Prometheus)
	configMap.Set("http_cert", config.HTTP.Cert)
	configMap.Set("http_key", config.HTTP.Key)
	configMap.Set("http_listen", config.HTTP.Listen)
	configMap.Set("http_timeout_min", config.HTTP.TimeoutMin)
	configMap.Set("http_body_limit", config.HTTP.BodyLimit)
	configMap.Set("http_cors_allow_origins", config.HTTP.CorsAllowOrigins)
	configMap.Set("http_session_secret", config.HTTP.SessionSecret)
	configMap.Set("http_ui", config.HTTP.UI)
	configMap.Set("http_upload_dir", upload_dir)
	configMap.Set("http_auth_header", config.HTTP.AuthHeader)
	configMap.Set("db_path", config.DB.Path)
	configMap.Set("db_encrypt_key", config.DB.EncryptKey)
	configMap.Set("db_headers", config.DB.ResponseHeaders)
	// Set default value for notifications.max to 100
	if config.Notifications.Max == 0 {
		configMap.Set("notifications_max", 100)
	} else {
		configMap.Set("notifications_max", config.Notifications.Max)
	}
	// Set default value for global.cmdprefix to sh
	if config.Global.CmdPrefix == "" {
		configMap.Set("global_cmd_prefix", "/bin/sh -c")
	} else {
		configMap.Set("global_cmd_prefix", config.Global.CmdPrefix)
	}
	// Set default timezone to UTC if empty
	if config.Global.Timezone == "" {
		configMap.Set("global_timezone", "UTC")
	} else {
		configMap.Set("global_timezone", config.Global.Timezone)
	}

	return nil
}

func GetConfigStr(key string) string {
	val, _ := configMap.Get(key)
	return val.(string)
}

func GetConfigBool(key string) bool {
	val, _ := configMap.Get(key)
	return val.(bool)
}

func GetConfigArray(key string) []string {
	val, _ := configMap.Get(key)
	return val.([]string)
}

func GetConfigInt(key string) int {
	val, _ := configMap.Get(key)
	return val.(int)
}

func GetConfigResponseHeaders() []data.ResponseHeaders {
	val, _ := configMap.Get("db_headers")
	return val.([]data.ResponseHeaders)
}

func GetConfigUI() data.UI {
	val, _ := configMap.Get("http_ui")
	return val.(data.UI)
}
