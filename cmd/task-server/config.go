package server

// Config is the configuration used by the server.
type Config struct {
	ListenAddr string `json:"listenAddr"`
	ListenPort int    `json:"listenPort"`
	TaskFile   string `json:"taskFile"`
	AllowVars  bool   `json:"allowVars"`

	UseTLS      bool   `json:"useTLS"`
	TLSKeyPath  string `json:"tlsKeyPath"`
	TLSCertPath string `json:"tlsCertPath"`

	LogRequests bool `json:"logRequests"`
}

// RuntimeConfig keeps only the info we need at
// runtime.
type RuntimeConfig struct {
	taskFilePath string
	logRequests  bool
}
