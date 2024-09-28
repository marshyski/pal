package data

import "time"

// ResponseHeaders
type ResponseHeaders struct {
	Header string `yaml:"header" json:"header"`
	Value  string `yaml:"value" json:"value"`
}

type OnError struct {
	Notification string `yaml:"notification" json:"notification"`
}

// GroupData struct for action data of a group
type GroupData struct {
	Desc            string            `yaml:"desc" json:"desc"`
	Background      bool              `yaml:"background" json:"background"`
	Action          string            `yaml:"action" json:"action" validate:"required"`
	Concurrent      bool              `yaml:"concurrent" json:"concurrent"`
	AuthHeader      string            `yaml:"auth_header" json:"-"`
	Output          bool              `yaml:"output" json:"-"`
	Timeout         int               `yaml:"timeout" json:"timeout"`
	Cmd             string            `yaml:"cmd" json:"cmd" validate:"required"`
	ResponseHeaders []ResponseHeaders `yaml:"response_headers" json:"response_headers"`
	Cron            string            `yaml:"cron" json:"cron"`
	OnError         OnError           `yaml:"on_error" json:"on_error"`
	InputValidate   string            `yaml:"input_validate" json:"input_validate"`
	LastRan         string            `json:"last_ran"`
	LastDuration    int               `json:"last_duration"`
	LastOutput      string            `json:"-"`
	Status          string            `json:"status"`
	Disabled        bool              `json:"disabled"`
	Lock            bool              `json:"-"`
	Tags            []string          `json:"tags"`
}

// UI is optional no validation needed here
type UI struct {
	UploadDir string `yaml:"upload_dir"`
	BasicAuth string `yaml:"basic_auth"`
}

// Config
type Config struct {
	Global struct {
		Timezone string `yaml:"timezone"`
		Debug    bool   `yaml:"debug"`
	} `yaml:"global"`
	HTTP struct {
		Listen           string   `yaml:"listen" validate:"required"`
		TimeoutMin       int      `yaml:"timeout_min" validate:"gte=0"`
		BodyLimit        string   `yaml:"body_limit"`
		CorsAllowOrigins []string `yaml:"cors_allow_origins"`
		SessionSecret    string   `yaml:"session_secret"`
		AuthHeader       string   `yaml:"auth_header" validate:"gte=16"`
		Key              string   `yaml:"key" validate:"file"`
		Cert             string   `yaml:"cert" validate:"file"`
		UI
	} `yaml:"http"`
	DB struct {
		EncryptKey      string            `yaml:"encrypt_key" validate:"gte=16"`
		ResponseHeaders []ResponseHeaders `yaml:"response_headers"`
		Path            string            `yaml:"path" validate:"dir"`
	} `yaml:"db"`
}

// Schedules
type Schedules struct {
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
	Group           string `json:"group" validate:"required"`
	Notification    string `json:"notification" validate:"required"`
	NotificationRcv string `json:"notification_received"`
}
