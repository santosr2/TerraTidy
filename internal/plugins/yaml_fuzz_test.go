package plugins

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func FuzzYAMLRuleParse(f *testing.F) {
	f.Add([]byte(`name: require-description
description: Resources must have a description
severity: warning
enabled: true
message: "Resource is missing a description attribute"
patterns:
  resource_types:
    - aws_s3_bucket
  required_attributes:
    - description
`))
	f.Add([]byte(`name: test-rule
severity: error
enabled: false
tags:
  - security
  - compliance
`))
	f.Add([]byte(`{}`))
	f.Add([]byte(``))

	f.Fuzz(func(t *testing.T, data []byte) {
		var config YAMLRuleConfig
		if err := yaml.Unmarshal(data, &config); err != nil {
			return
		}

		if config.Name == "" {
			return
		}

		rule := &YAMLRule{config: config}
		_ = rule.Name()
		_ = rule.Description()
	})
}
