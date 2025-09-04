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

package data

import "time"

// HTTP Headers
type Headers struct {
	Header string `yaml:"header" json:"header"`
	Value  string `yaml:"value" json:"value"`
}

type Run struct {
	Group  string `yaml:"group" json:"group"`
	Action string `yaml:"action" json:"action"`
	Input  string `yaml:"input" json:"input"`
}

type OnError struct {
	Notification  string   `yaml:"notification" json:"notification"`
	Retries       int      `yaml:"retries" json:"retries" validate:"number"`
	RetryInterval int      `yaml:"retry_interval" json:"retry_interval" validate:"number"`
	Run           []Run    `yaml:"run" json:"run"`
	Webhook       []string `yaml:"webhooks" json:"webhooks"`
}

type OnSuccess struct {
	Notification string   `yaml:"notification" json:"notification"`
	Run          []Run    `yaml:"run" json:"run"`
	Webhook      []string `yaml:"webhooks" json:"webhooks"`
}

type Container struct {
	Sudo    bool   `yaml:"sudo" json:"sudo"`
	Image   string `yaml:"image" json:"image"`
	Options string `yaml:"options" json:"options"`
}

type Triggers struct {
	OriginGroup      string `json:"origin_group"`
	OriginAction     string `json:"origin_action"`
	TriggerGroup     string `json:"trigger_group"`
	TriggerAction    string `json:"trigger_action"`
	TriggerCondition string `json:"trigger_condition"`
	TriggerInput     string `json:"trigger_input"`
}

// ActionData struct for action data of a group
type ActionData struct {
	Group             string       `yaml:"-" json:"group"`
	Desc              string       `yaml:"desc" json:"desc"`
	Background        bool         `yaml:"background" json:"background" validate:"boolean"`
	Action            string       `yaml:"action" json:"action" validate:"required"`
	Concurrent        bool         `yaml:"concurrent" json:"concurrent" validate:"boolean"`
	AuthHeader        string       `yaml:"auth_header" json:"auth_header"`
	Output            bool         `yaml:"output" json:"output" validate:"boolean"`
	Container         Container    `yaml:"container" json:"container"`
	Timeout           int          `yaml:"timeout" json:"timeout" validate:"number"`
	CmdPrefix         string       `yaml:"cmd_prefix" json:"cmd_prefix"`
	Cmd               string       `yaml:"cmd" json:"cmd" validate:"required"`
	ResponseHeaders   []Headers    `yaml:"headers" json:"headers"`
	Crons             []string     `yaml:"crons" json:"crons"`
	OnError           OnError      `yaml:"on_error" json:"on_error"`
	OnSuccess         OnSuccess    `yaml:"on_success" json:"on_success"`
	Input             string       `yaml:"input" json:"input"`
	InputValidate     string       `yaml:"input_validate" json:"input_validate"`
	Register          DBSet        `yaml:"register" json:"register"`
	Image             string       `yaml:"image" json:"image"`
	GitRepo           string       `yaml:"git_repo" json:"git_repo"`
	Triggers          []Triggers   `yaml:"-" json:"triggers"`
	LastRan           string       `yaml:"-" json:"last_ran"`
	LastSuccess       string       `yaml:"-" json:"last_success"`
	LastFailure       string       `yaml:"-" json:"last_failure"`
	LastDuration      string       `yaml:"-" json:"last_duration"`
	LastSuccessOutput string       `yaml:"-" json:"last_success_output"`
	LastFailureOutput string       `yaml:"-" json:"last_failure_output"`
	RunCount          int          `yaml:"-" json:"run_count"`
	Status            string       `yaml:"-" json:"status"`
	Disabled          bool         `yaml:"-" json:"disabled" validate:"boolean"`
	Lock              bool         `yaml:"-" json:"-" validate:"boolean"`
	RunHistory        []RunHistory `yaml:"-" json:"run_history"`
}

type RunHistory struct {
	Ran      string `yaml:"-" json:"ran"`
	Duration string `yaml:"-" json:"duration"`
	Status   string `yaml:"-" json:"status"`
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
		PodmanSocket string `yaml:"podman_socket"`
	} `yaml:"global"`
	HTTP struct {
		Listen          string    `yaml:"listen" validate:"required"`
		TimeoutMin      int       `yaml:"timeout_min" validate:"number"`
		BodyLimit       string    `yaml:"body_limit"`
		ResponseHeaders []Headers `yaml:"headers"`
		SessionSecret   string    `yaml:"session_secret"`
		MaxAge          int       `yaml:"max_age" validate:"number"`
		Prometheus      bool      `yaml:"prometheus"`
		IPV6            bool      `yaml:"ipv6"`
		Key             string    `yaml:"key" validate:"file"`
		Cert            string    `yaml:"cert" validate:"file"`
		UI
	} `yaml:"http"`
	DB struct {
		EncryptKey string `yaml:"encrypt_key" validate:"gte=16"`
		Path       string `yaml:"path" validate:"dir"`
	} `yaml:"db"`
	Notifications struct {
		StoreMax int       `yaml:"store_max" validate:"number"`
		Webhooks []Webhook `yaml:"webhooks" json:"webhooks"`
	} `yaml:"notifications"`
}

type Webhook struct {
	Name     string    `yaml:"name" json:"name"`
	URL      string    `yaml:"url" json:"url"`
	Method   string    `yaml:"method" json:"method"`
	Headers  []Headers `yaml:"headers" json:"headers"`
	Insecure bool      `yaml:"insecure" json:"insecure"`
	Body     string    `yaml:"body" json:"body"`
}

// Crons
type Crons struct {
	LastDuration string    `json:"last_duration"`
	Status       string    `json:"status"`
	LastRan      time.Time `json:"last_ran"`
	NextRun      time.Time `json:"next_run"`
	Group        string    `json:"group"`
	Action       string    `json:"action"`
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
	Action          string `json:"action"`
	Status          string `json:"status"`
	Notification    string `json:"notification" validate:"required"`
	NotificationRcv string `json:"notification_received"`
}

// DBSet
type DBSet struct {
	Key    string `yaml:"key" json:"key"`
	Value  string `yaml:"value" json:"value"`
	Secret bool   `yaml:"secret" json:"secret"`
}
