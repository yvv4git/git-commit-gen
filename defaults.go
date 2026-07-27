package defaults

import _ "embed"

//go:embed configs/generator.example.toml
var ConfigContent string

//go:embed rules.example.md
var RulesContent string
