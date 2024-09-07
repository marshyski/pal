package data

// ResponseHeaders
type ResponseHeaders struct {
	Header string `yaml:"header"`
	Value  string `yaml:"value"`
}

// ResourceData struct for target data of a resource
type ResourceData struct {
	Background      bool              `yaml:"background"`
	Target          string            `yaml:"target"`
	Concurrent      bool              `yaml:"concurrent"`
	AuthHeader      string            `yaml:"auth_header"`
	Output          bool              `yaml:"output"`
	Cmd             string            `yaml:"cmd"`
	ResponseHeaders []ResponseHeaders `yaml:"response_headers"`
	ContentType     string            `yaml:"content_type"`
	Schedule        string            `yaml:"schedule"`
	ScheduleTZ      string            `yaml:"schedule_tz"`
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
		Listen           string   `yaml:"listen"`
		TimeoutMin       int      `yaml:"timeout_min"`
		BodyLimit        string   `yaml:"body_limit"`
		CorsAllowOrigins []string `yaml:"cors_allow_origins"`
		ScheduleTZ       string   `yaml:"schedule_tz"`
		Key              string   `yaml:"key"`
		Cert             string   `yaml:"cert"`
		UI
	} `yaml:"http"`
	DB struct {
		EncryptKey      string            `yaml:"encrypt_key"`
		AuthHeader      string            `yaml:"auth_header"`
		ResponseHeaders []ResponseHeaders `yaml:"response_headers"`
		Path            string            `yaml:"path"`
	} `yaml:"db"`
}
