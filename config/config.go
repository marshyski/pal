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

package config

import (
	"errors"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/go-playground/validator/v10"
	"github.com/marshyski/pal/data"
	"github.com/marshyski/pal/utils"
	cmap "github.com/orcaman/concurrent-map"
	"gopkg.in/yaml.v3"
)

const (
	defaultNotifications = 100
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
		log.Fatalln("error reading directory: "+dir, err)
	}

	groups := make(map[string][]data.ActionData)

	for _, file := range files {
		if filepath.Ext(file.Name()) == ".yml" {
			fileLoc := filepath.Clean(dir + "/" + file.Name())
			defs, err := os.ReadFile(fileLoc)
			if err != nil {
				log.Println("error reading file: "+fileLoc, err)
				continue // Skip to the next file
			}

			var groupData map[string][]data.ActionData
			err = yaml.Unmarshal(defs, &groupData)
			if err != nil {
				log.Println("error unmarshaling YAML: "+fileLoc, err)
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

	var uploadDir string
	uploadDir, err = filepath.Abs(filepath.Clean(config.HTTP.UploadDir))
	if err != nil {
		uploadDir = ""
	}

	workingDir, err := filepath.Abs(filepath.Clean(config.Global.WorkingDir))
	if err != nil {
		workingDir, err = os.Getwd()
		if err != nil {
			workingDir = "./"
		}
	}

	var containerCmd string
	if config.Global.ContainerCmd != "" {
		containerCmd = config.Global.ContainerCmd
	} else {
		// Check if Podman is available
		if _, err := exec.LookPath("podman"); err == nil {
			containerCmd = "podman"
		} else {
			// If Podman is not found, check for Docker
			if _, err := exec.LookPath("docker"); err == nil {
				containerCmd = "docker"
			}
		}
	}

	configMap.Set("global_working_dir", workingDir)
	configMap.Set("global_debug", config.Global.Debug)
	configMap.Set("global_container_cmd", containerCmd)
	configMap.Set("http_prometheus", config.HTTP.Prometheus)
	configMap.Set("http_ipv6", config.HTTP.IPV6)
	configMap.Set("http_cert", config.HTTP.Cert)
	configMap.Set("http_key", config.HTTP.Key)
	configMap.Set("http_listen", config.HTTP.Listen)
	configMap.Set("http_timeout_min", config.HTTP.TimeoutMin)
	configMap.Set("http_body_limit", config.HTTP.BodyLimit)
	configMap.Set("http_max_age", config.HTTP.MaxAge)
	configMap.Set("http_cors_allow_origins", config.HTTP.CorsAllowOrigins)
	configMap.Set("http_session_secret", config.HTTP.SessionSecret)
	configMap.Set("http_ui", config.HTTP.UI)
	configMap.Set("http_upload_dir", uploadDir)
	configMap.Set("db_path", config.DB.Path)
	configMap.Set("db_encrypt_key", config.DB.EncryptKey)
	configMap.Set("db_headers", config.DB.ResponseHeaders)
	// Set default value for notifications.max to defaultNotifications const
	if config.Notifications.Max == 0 {
		configMap.Set("notifications_max", defaultNotifications)
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
	v, ok := val.(string)
	if !ok {
		return ""
	}
	return v
}

func GetConfigBool(key string) bool {
	val, _ := configMap.Get(key)
	v, ok := val.(bool)
	if !ok {
		return false
	}
	return v
}

func GetConfigArray(key string) []string {
	val, _ := configMap.Get(key)
	v, ok := val.([]string)
	if !ok {
		return []string{}
	}
	return v
}

func GetConfigInt(key string) int {
	val, _ := configMap.Get(key)
	v, ok := val.(int)
	if !ok {
		return 0
	}
	return v
}

func GetConfigResponseHeaders() []data.ResponseHeaders {
	val, _ := configMap.Get("db_headers")
	v, ok := val.([]data.ResponseHeaders)
	if !ok {
		return []data.ResponseHeaders{}
	}
	return v
}

func GetConfigUI() data.UI {
	val, _ := configMap.Get("http_ui")
	v, ok := val.(data.UI)
	if !ok {
		return data.UI{}
	}
	return v
}

func SetActionsDir(dir string) {
	configMap.Set("global_actions_dir", dir)
}

func SetActionsReload() {
	configMap.Set("global_actions_reload", utils.TimeNow(GetConfigStr("global_timezone")))
}

func SetConfigFile(file string) {
	configMap.Set("global_config_file", file)
}

func SetVersion(ver string) {
	configMap.Set("global_version", ver)
}
