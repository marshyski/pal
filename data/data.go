// pal - github.com/marshyski/pal
// Copyright (C) 2024  github.com/marshyski

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

package data

import "time"

// ResponseHeaders
type ResponseHeaders struct {
	Header string `yaml:"header" json:"header"`
	Value  string `yaml:"value" json:"value"`
}

type OnError struct {
	Notification  string `yaml:"notification" json:"notification"`
	Retries       int    `yaml:"retries" json:"retries" validate:"number"`
	RetryInterval int    `yaml:"retry_interval" json:"retry_interval" validate:"number"`
}

type Container struct {
	Sudo    bool   `yaml:"sudo" json:"sudo"`
	Image   string `yaml:"image" json:"image"`
	Options string `yaml:"options" json:"options"`
}

// ActionData struct for action data of a group
type ActionData struct {
	Group             string            `yaml:"-" json:"group"`
	Desc              string            `yaml:"desc" json:"desc"`
	Background        bool              `yaml:"background" json:"background" validate:"boolean"`
	Action            string            `yaml:"action" json:"action" validate:"required"`
	Concurrent        bool              `yaml:"concurrent" json:"concurrent" validate:"boolean"`
	AuthHeader        string            `yaml:"auth_header" json:"auth_header"`
	Output            bool              `yaml:"output" json:"output" validate:"boolean"`
	Container         Container         `yaml:"container" json:"container"`
	Timeout           int               `yaml:"timeout" json:"timeout" validate:"number"`
	Cmd               string            `yaml:"cmd" json:"cmd" validate:"required"`
	ResponseHeaders   []ResponseHeaders `yaml:"headers" json:"headers"`
	Crons             []string          `yaml:"crons" json:"crons"`
	OnError           OnError           `yaml:"on_error" json:"on_error"`
	Input             string            `yaml:"input" json:"input"`
	InputValidate     string            `yaml:"input_validate" json:"input_validate"`
	Tags              []string          `yaml:"tags" json:"tags"`
	LastRan           string            `yaml:"-" json:"last_ran"`
	LastSuccess       string            `yaml:"-" json:"last_success"`
	LastFailure       string            `yaml:"-" json:"last_failure"`
	LastDuration      int               `yaml:"-" json:"last_duration" validate:"number"`
	LastSuccessOutput string            `yaml:"-" json:"last_success_output"`
	LastFailureOutput string            `yaml:"-" json:"last_failure_output"`
	Status            string            `yaml:"-" json:"status"`
	Disabled          bool              `yaml:"-" json:"disabled" validate:"boolean"`
	Lock              bool              `yaml:"-" json:"-" validate:"boolean"`
}

// UI is optional no validation needed here
type UI struct {
	UploadDir string `yaml:"upload_dir"`
	BasicAuth string `yaml:"basic_auth"`
}

// Config
type Config struct {
	Global struct {
		Timezone     string `yaml:"timezone"`
		CmdPrefix    string `yaml:"cmd_prefix"`
		ContainerCmd string `yaml:"container_cmd"`
		WorkingDir   string `yaml:"working_dir"`
		Debug        bool   `yaml:"debug" validate:"boolean"`
	} `yaml:"global"`
	HTTP struct {
		Listen           string   `yaml:"listen" validate:"required"`
		TimeoutMin       int      `yaml:"timeout_min" validate:"number"`
		BodyLimit        string   `yaml:"body_limit"`
		CorsAllowOrigins []string `yaml:"cors_allow_origins"`
		SessionSecret    string   `yaml:"session_secret"`
		AuthHeader       string   `yaml:"auth_header" validate:"gte=16"`
		MaxAge           int      `yaml:"max_age" validate:"number"`
		Prometheus       bool     `yaml:"prometheus"`
		IPV6             bool     `yaml:"ipv6"`
		Key              string   `yaml:"key" validate:"file"`
		Cert             string   `yaml:"cert" validate:"file"`
		UI
	} `yaml:"http"`
	DB struct {
		EncryptKey      string            `yaml:"encrypt_key" validate:"gte=16"`
		ResponseHeaders []ResponseHeaders `yaml:"headers"`
		Path            string            `yaml:"path" validate:"dir"`
	} `yaml:"db"`
	Notifications struct {
		Max int `yaml:"max" validate:"number"`
	} `yaml:"notifications"`
}

// Crons
type Crons struct {
	LastRan time.Time `json:"last_ran"`
	NextRun time.Time `json:"next_run"`
	Group   string    `json:"group"`
	Action  string    `json:"action"`
}

// GenericResponse
type GenericResponse struct {
	Message string `json:"message"`
	Err     string `json:"err"`
}

// Notification
type Notification struct {
	ID              string `json:"id"`
	Group           string `json:"group" validate:"required"`
	Notification    string `json:"notification" validate:"required"`
	NotificationRcv string `json:"notification_received"`
}

// DBSet
type DBSet struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	Secret bool   `json:"secret"`
}
