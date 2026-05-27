module require-tags

go 1.25.0

require (
	github.com/hashicorp/hcl/v2 v2.24.0
	github.com/santosr2/TerraTidy v0.2.0-alpha.4
)

require (
	github.com/agext/levenshtein v1.2.1 // indirect
	github.com/apparentlymart/go-textseg/v15 v15.0.0 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/zclconf/go-cty v1.18.1 // indirect
	golang.org/x/mod v0.35.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/text v0.37.0 // indirect
	golang.org/x/tools v0.44.0 // indirect
)

// Use local TerraTidy for development
replace github.com/santosr2/TerraTidy => ../..
