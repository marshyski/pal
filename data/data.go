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

// ActionData struct for action data of a group
type ActionData struct {
	Group           string            `json:"group"`
	Desc            string            `yaml:"desc" json:"desc"`
	Background      bool              `yaml:"background" json:"background" validate:"boolean"`
	Action          string            `yaml:"action" json:"action" validate:"required"`
	Concurrent      bool              `yaml:"concurrent" json:"concurrent" validate:"boolean"`
	AuthHeader      string            `yaml:"auth_header" json:"-"`
	Output          bool              `yaml:"output" json:"output" validate:"boolean"`
	Timeout         int               `yaml:"timeout" json:"timeout" validate:"number"`
	Cmd             string            `yaml:"cmd" json:"cmd" validate:"required"`
	ResponseHeaders []ResponseHeaders `yaml:"resp_headers" json:"resp_headers"`
	Cron            string            `yaml:"cron" json:"cron"`
	OnError         OnError           `yaml:"on_error" json:"on_error"`
	InputValidate   string            `yaml:"input_validate" json:"input_validate"`
	LastRan         string            `json:"last_ran"`
	LastDuration    int               `json:"last_duration" validate:"number"`
	LastOutput      string            `json:"last_output"`
	Status          string            `json:"status"`
	Disabled        bool              `json:"disabled" validate:"boolean"`
	Lock            bool              `json:"-" validate:"boolean"`
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
		Timezone  string `yaml:"timezone"`
		CmdPrefix string `yaml:"cmd_prefix"`
		Debug     bool   `yaml:"debug" validate:"boolean"`
	} `yaml:"global"`
	HTTP struct {
		Listen           string   `yaml:"listen" validate:"required"`
		TimeoutMin       int      `yaml:"timeout_min" validate:"number"`
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
		ResponseHeaders []ResponseHeaders `yaml:"resp_headers"`
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
