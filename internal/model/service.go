package model

type Service struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Project         string   `json:"project"`
	Directory       string   `json:"directory"`
	Command         string   `json:"command"`
	Port            int      `json:"port"`
	URL             string   `json:"url"`
	Framework       string   `json:"framework"`
	Source          string   `json:"source"`
	PID             int32    `json:"pid"`
	Birth           int64    `json:"birth"`
	Status          string   `json:"status"`
	Managed         bool     `json:"managed"`
	Cover           string   `json:"cover"`
	ScreenshotError string   `json:"screenshotError"`
	ProcessName     string   `json:"processName,omitempty"`
	Ports           []int    `json:"ports,omitempty"`
	DiscoveryReason string   `json:"discoveryReason,omitempty"`
	DiscoveryDetail string   `json:"discoveryDetail,omitempty"`
	Failure         string   `json:"failure,omitempty"`
	FailureDetail   string   `json:"failureDetail,omitempty"`
	Env             []EnvVar `json:"env,omitempty"`
	Health          Health   `json:"health"`
	Updated         string   `json:"updated"`
}

type EnvVar struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Secret bool   `json:"secret"`
}

const (
	FailureStartExit    = "start-exit"
	FailureStartTimeout = "start-timeout"
	HealthProcessNoPort = "process-no-port"
)

// Health separates process identity, TCP reachability and HTTP readiness.
// The values are deliberately small strings so clients can render them
// without knowing platform-specific probe details.
type Health struct {
	Process    string `json:"process"`
	TCP        string `json:"tcp"`
	HTTP       string `json:"http"`
	HTTPStatus int    `json:"httpStatus,omitempty"`
	Error      string `json:"error,omitempty"`
	CheckedAt  string `json:"checkedAt,omitempty"`
}

type Log struct {
	ID   int64  `json:"id"`
	Time string `json:"time"`
	Text string `json:"text"`
}
