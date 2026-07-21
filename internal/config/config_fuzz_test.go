package config

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func FuzzConfigParse(f *testing.F) {
	f.Add([]byte(`version: 1
engines:
  fmt:
    enabled: true
  style:
    enabled: true
`))
	f.Add([]byte(`version: 1
imports:
  - .terratidy/*.yaml
profiles:
  production:
    engines:
      policy:
        enabled: true
`))
	f.Add([]byte(`{}`))
	f.Add([]byte(``))

	f.Fuzz(func(t *testing.T, data []byte) {
		expanded, expandErr := expandEnvVars(string(data))
		if expandErr != nil {
			// A config declaring an unset required var is a legitimate error, not a crash.
			return
		}

		var cfg Config
		if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
			return
		}

		_ = cfg.Validate()
	})
}
