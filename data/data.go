package data

import "time"

// ResponseHeaders
type ResponseHeaders struct {
	Header string `yaml:"header"`
	Value  string `yaml:"value"`
}

// ResourceData struct for target data of a resource
type ResourceData struct {
	Background      bool              `yaml:"background"`
	Target          string            `yaml:"target" validate:"required"`
	Concurrent      bool              `yaml:"concurrent"`
	AuthHeader      string            `yaml:"auth_header"`
	Output          bool              `yaml:"output"`
	Cmd             string            `yaml:"cmd" validate:"required"`
	ResponseHeaders []ResponseHeaders `yaml:"response_headers"`
	ContentType     string            `yaml:"content_type"`
	Schedule        string            `yaml:"schedule"`
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
	LastRun time.Time `json:"last_run"`
	NextRun time.Time `json:"next_run"`
}

// GenericResponse
type GenericResponse struct {
	Message string `json:"message"`
	Err     string `json:"err"`
}

type Notification struct {
	Notification    string `json:"notification" validate:"required"`
	NotificationRcv string `json:"notification_received"`
}
