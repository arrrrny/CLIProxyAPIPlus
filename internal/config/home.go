package config

// HomeTLSConfig configures TLS for the Home control plane connection.
type HomeTLSConfig struct {
	Enable              bool   `yaml:"enable" json:"enable"`
	CACert              string `yaml:"ca_cert" json:"ca_cert"`
	ClientCert          string `yaml:"client_cert" json:"client_cert"`
	ClientKey           string `yaml:"client_key" json:"client_key"`
	ServerName          string `yaml:"server_name" json:"server_name"`
	InsecureSkipVerify  bool   `yaml:"insecure_skip_verify" json:"insecure_skip_verify"`
	UseTargetServerName bool   `yaml:"use_target_server_name" json:"use_target_server_name"`
}

// HomeConfig configures the optional "home" control plane integration over Redis protocol.
type HomeConfig struct {
	Enabled                 bool          `yaml:"enabled" json:"enabled"`
	Host                    string        `yaml:"host" json:"-"`
	Port                    int           `yaml:"port" json:"-"`
	Password                string        `yaml:"password" json:"-"`
	DisableClusterDiscovery bool          `yaml:"disable_cluster_discovery" json:"disable_cluster_discovery"`
	TLS                     HomeTLSConfig `yaml:"tls" json:"tls"`
}
