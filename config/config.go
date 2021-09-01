package config

// Config file (globa)
var Config JSONConfig

// JSONConfig structure based on config.json
type JSONConfig struct {
	Origin    string      `json:"origin"`
	Port      string      `json:"port"`
	Version   string      `json:"version"`
	EmailFrom string      `json:"emailFrom"`
	SMTP      SMTPConfig  `json:"smtp"`
	PubPort   string      `json:"pubPort"`
	SubPort   string      `json:"subPort"`
	MinIO     MinIOConfig `json:"minIO"`
}

// SMTPConfig structure based on smtp part of config.json
type SMTPConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
}

// MinIOConfig structure is the config for MinIO connectoin
type MinIOConfig struct {
	User     string `json:"user"`
	Password string `json:"password"`
}
