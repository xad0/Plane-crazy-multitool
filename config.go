package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type config struct {
	Appearance appearanceConfig `json:"appearance"`
	Layout     layoutConfig     `json:"layout"`
	Reference  referenceConfig  `json:"reference"`
	Calculator calculatorConfig `json:"calculator"`
	Crank      crankConfig      `json:"crank"`
	Dyno       dynoConfig       `json:"dyno"`
	Wheel      wheelConfig      `json:"wheel"`
	Piston     pistonConfig     `json:"piston"`

	SetupCompleted  bool   `json:"setup_completed"`
	ExportDirectory string `json:"export_directory"`
}

type appearanceConfig struct {
	Theme                 string `json:"theme"`
	Accent                string `json:"accent"`
	Background            string `json:"background"`
	TopBarTransparent     bool   `json:"top_bar_transparent"`
	ReferenceTransparent  bool   `json:"reference_transparent"`
	CalculatorTransparent bool   `json:"calculator_transparent"`
	BottomBarTransparent  bool   `json:"bottom_bar_transparent"`
}

type layoutConfig struct {
	Mode           string `json:"mode"`
	ReferenceWidth string `json:"reference_width"`
	Order          string `json:"order"`
}

type referenceConfig struct {
	Enabled         bool `json:"enabled"`
	SmallestGear    int  `json:"smallest_gear"`
	MaximumGear     int  `json:"maximum_gear"`
	Teeth           bool `json:"teeth"`
	FullAngle       bool `json:"full_angle"`
	HalfAngle       bool `json:"half_angle"`
	Offset          bool `json:"offset"`
	CompressorValue bool `json:"compressor_value"`
	Compressors     bool `json:"compressors"`
}

type crankConfig struct {
	Layout string `json:"layout"`
}

type dynoConfig struct {
	GraphSide string `json:"graph_side"`
}

type wheelConfig struct {
	ResultSide string `json:"result_side"`
}

type pistonConfig struct {
	ResultSide string `json:"result_side"`
}

type calculatorConfig struct {
	Enabled         bool `json:"enabled"`
	Teeth           bool `json:"teeth"`
	Compressors     bool `json:"compressors"`
	FullAngle       bool `json:"full_angle"`
	HalfAngle       bool `json:"half_angle"`
	Offset          bool `json:"offset"`
	CompressorValue bool `json:"compressor_value"`
	Warnings        bool `json:"warnings"`
}

func defaultExportDirectory() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}

	return filepath.Join(
		home,
		"Pictures",
		"PC-Multitool",
	)
}

func defaultConfig() config {
	return config{
		Appearance: appearanceConfig{
			Theme:                 "catppuccin",
			Accent:                "green",
			Background:            "theme",
			TopBarTransparent:     false,
			ReferenceTransparent:  false,
			CalculatorTransparent: false,
			BottomBarTransparent:  false,
		},

		Layout: layoutConfig{
			Mode:           "automatic",
			ReferenceWidth: "balanced",
			Order:          "reference-first",
		},

		Reference: referenceConfig{
			Enabled:         true,
			SmallestGear:    4,
			MaximumGear:     20,
			Teeth:           true,
			FullAngle:       true,
			HalfAngle:       true,
			Offset:          true,
			CompressorValue: true,
			Compressors:     true,
		},

		Crank: crankConfig{
			Layout: "results-right",
		},

		Dyno: dynoConfig{
			GraphSide: "right",
		},

		Wheel: wheelConfig{
			ResultSide: "right",
		},

		Piston: pistonConfig{
			ResultSide: "right",
		},

		Calculator: calculatorConfig{
			Enabled:         true,
			Teeth:           true,
			Compressors:     true,
			FullAngle:       true,
			HalfAngle:       true,
			Offset:          true,
			CompressorValue: true,
			Warnings:        true,
		},

		SetupCompleted:  false,
		ExportDirectory: defaultExportDirectory(),
	}
}

func configPath() string {
	dir, err := os.UserConfigDir()

	if err == nil {
		return filepath.Join(
			dir,
			"PC-Gear-Calculator",
			"config.json",
		)
	}

	home, err := os.UserHomeDir()

	if err != nil {
		return "pc-gear-calculator.json"
	}

	return filepath.Join(
		home,
		".pc-gear-calculator.json",
	)
}

func loadConfig() config {
	data, err := os.ReadFile(
		configPath(),
	)

	if err != nil {
		return defaultConfig()
	}

	var raw map[string]json.RawMessage

	if err := json.Unmarshal(
		data,
		&raw,
	); err != nil {
		return defaultConfig()
	}

	cfg :=
		defaultConfig()

	if value, ok := raw["appearance"]; ok {
		_ = json.Unmarshal(
			value,
			&cfg.Appearance,
		)
	}

	if value, ok := raw["layout"]; ok {
		_ = json.Unmarshal(
			value,
			&cfg.Layout,
		)
	}

	if value, ok := raw["reference"]; ok {
		_ = json.Unmarshal(
			value,
			&cfg.Reference,
		)
	}

	if value, ok := raw["calculator"]; ok {
		_ = json.Unmarshal(
			value,
			&cfg.Calculator,
		)
	}

	if value, ok := raw["setup_completed"]; ok {
		_ = json.Unmarshal(
			value,
			&cfg.SetupCompleted,
		)
	}

	if value, ok := raw["export_directory"]; ok {
		_ = json.Unmarshal(
			value,
			&cfg.ExportDirectory,
		)
	}

	return normalizeConfig(cfg)
}

func normalizeConfig(
	cfg config,
) config {

	defaults :=
		defaultConfig()

	if cfg.Appearance.Theme == "" {
		cfg.Appearance.Theme =
			defaults.Appearance.Theme
	}

	if cfg.Appearance.Accent == "" {
		cfg.Appearance.Accent =
			defaults.Appearance.Accent
	}

	if cfg.Appearance.Background == "" {
		cfg.Appearance.Background =
			defaults.Appearance.Background
	}

	if cfg.Layout.Mode == "" {
		cfg.Layout.Mode =
			defaults.Layout.Mode
	}

	if cfg.Layout.ReferenceWidth == "" {
		cfg.Layout.ReferenceWidth =
			defaults.Layout.ReferenceWidth
	}

	if cfg.Layout.Order == "" {
		cfg.Layout.Order =
			defaults.Layout.Order
	}

	if cfg.ExportDirectory == "" {
		cfg.ExportDirectory =
			defaults.ExportDirectory
	}

	if cfg.Reference.SmallestGear < 3 {
		cfg.Reference.SmallestGear =
			defaults.Reference.SmallestGear
	}

	if cfg.Crank.Layout != "results-left" &&
		cfg.Crank.Layout != "results-right" {
		cfg.Crank.Layout =
			defaults.Crank.Layout
	}

	if cfg.Dyno.GraphSide != "left" &&
		cfg.Dyno.GraphSide != "right" {
		cfg.Dyno.GraphSide =
			defaults.Dyno.GraphSide
	}

	if cfg.Wheel.ResultSide != "left" &&
		cfg.Wheel.ResultSide != "right" {
		cfg.Wheel.ResultSide =
			defaults.Wheel.ResultSide
	}

	if cfg.Reference.MaximumGear < cfg.Reference.SmallestGear {
		cfg.Reference.MaximumGear =
			defaults.Reference.MaximumGear

		if cfg.Reference.MaximumGear < cfg.Reference.SmallestGear {
			cfg.Reference.MaximumGear =
				cfg.Reference.SmallestGear
		}
	}

	if cfg.Piston.ResultSide != "left" &&
		cfg.Piston.ResultSide != "right" {
		cfg.Piston.ResultSide =
			defaults.Piston.ResultSide
	}

	return cfg
}

func saveConfig(cfg config) {
	path :=
		configPath()

	if err := os.MkdirAll(
		filepath.Dir(path),
		0o755,
	); err != nil {
		return
	}

	data, err :=
		json.MarshalIndent(
			cfg,
			"",
			"    ",
		)

	if err != nil {
		return
	}

	_ = os.WriteFile(
		path,
		data,
		0o644,
	)
}
