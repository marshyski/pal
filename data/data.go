package data

import "time"

// ResponseHeaders
type ResponseHeaders struct {
	Header string `yaml:"header"`
	Value  string `yaml:"value"`
}

type OnError struct {
	Notification string `yaml:"notification"`
}

// GroupData struct for action data of a group
type GroupData struct {
	Background      bool              `yaml:"background"`
	Action          string            `yaml:"action" validate:"required"`
	Concurrent      bool              `yaml:"concurrent"`
	AuthHeader      string            `yaml:"auth_header"`
	Output          bool              `yaml:"output"`
	Cmd             string            `yaml:"cmd" validate:"required"`
	ResponseHeaders []ResponseHeaders `yaml:"response_headers"`
	ContentType     string            `yaml:"content_type"`
	Schedule        string            `yaml:"schedule"`
	OnError         OnError           `yaml:"on_error"`
	InputValidate   string            `yaml:"input_validate"`
	LastRan         string
	LastOutput      string
	Status          string
	Lock            bool
}

// UI
type UI struct {
	UploadDir string `yaml:"upload_dir"`
	BasicAuth string `yaml:"basic_auth"`
}

// Config
type Config struct {
	HTTP struct {
		Listen           string   `yaml:"listen" validate:"required"`
		TimeoutMin       int      `yaml:"timeout_min" validate:"gte=0"`
		BodyLimit        string   `yaml:"body_limit"`
		CorsAllowOrigins []string `yaml:"cors_allow_origins"`
		SessionSecret    string   `yaml:"session_secret"`
		ScheduleTZ       string   `yaml:"schedule_tz"`
		AuthHeader       string   `yaml:"auth_header" validate:"gte=0"`
		Key              string   `yaml:"key"`
		Cert             string   `yaml:"cert"`
		UI
	} `yaml:"http"`
	DB struct {
		EncryptKey      string            `yaml:"encrypt_key" validate:"required"`
		ResponseHeaders []ResponseHeaders `yaml:"response_headers"`
		Path            string            `yaml:"path" validate:"required"`
	} `yaml:"db"`
}

// Schedules
type Schedules struct {
	Name    string    `json:"name"`
	LastRan time.Time `json:"last_ran"`
	NextRun time.Time `json:"next_run"`
}

// GenericResponse
type GenericResponse struct {
	Message string `json:"message"`
	Err     string `json:"err"`
}

// Notification
type Notification struct {
	Group           string `json:"group" validate:"required"`
	Notification    string `json:"notification" validate:"required"`
	NotificationRcv string `json:"notification_received"`
}
