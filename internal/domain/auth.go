package domain

type OnboardingPreferences struct {
	LogDir    string   `json:"log_dir"`
	LogErrors []string `json:"log_errors,omitempty"`
}
