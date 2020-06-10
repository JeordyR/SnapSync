package main

type snapsyncConfig struct {
	// Base settings
	Executable string `yaml:"Executable,omitempty"`

	// Log Settings
	LogFile string `yaml:"LogFile,omitempty"`

	// Touch Settings
	TouchEnabled bool `yaml:"TouchEnabled,omitempty"`

	// Threashold Settings
	DeleteThreashold int `yaml:"DeleteThreashold,omitempty"`

	// Scrub Settings
	ScrubEnabled    bool     `yaml:"ScrubEnabled,omitempty"`
	ScrubPercentage string   `yaml:"ScrubPercentage,omitempty"`
	ScrubOlderThan  string   `yaml:"ScrubOlderThan,omitempty"`
	ScrubDaysOfWeek []string `yaml:"ScrubDaysOfWeek,omitempty"`

	// Status Settings
	OutputStatus bool `yaml:"OutputStatus,omitempty"`

	// Pushover Settings
	PushoverEnabled bool   `yaml:"PushoverEnabled,omitempty"`
	PushoverAppKey  string `yaml:"PushoverAppKey,omitempty"`
	PushoverUserKey string `yaml:"PushoverUserKey,omitempty"`
}
