// Copyright 2020 the Drone Authors. All rights reserved.
// Use of this source code is governed by the Blue Oak Model License
// that can be found in the LICENSE file.

package plugin

import (
	"context"
)

type Plugin interface {
	Init(args *Args) error
	SetBuildRoot(buildRootPath string) error
	DeInit() error
	ValidateAndProcessArgs(args Args) error
	DoPostArgsValidationSetup(args Args) error
	Run() error
	WriteOutputVariables() error
	PersistResults() error
	IsQuiet() bool
	InspectProcessArgs(argNamesList []string) (map[string]interface{}, error)
}

type Args struct {
	EnvPluginInputArgs
	Level string `envconfig:"PLUGIN_LOG_LEVEL"`
}

type EnvPluginInputArgs struct {
	NexusVersion string `envconfig:"PLUGIN_NEXUS_VERSION"`
	Protocol     string `envconfig:"PLUGIN_PROTOCOL"`
	GroupId      string `envconfig:"PLUGIN_GROUP_ID"`
	Repository   string `envconfig:"PLUGIN_REPOSITORY"`
	Artifact     string `envconfig:"PLUGIN_ARTIFACTS"`
	Username     string `envconfig:"PLUGIN_USERNAME"`
	Password     string `envconfig:"PLUGIN_PASSWORD"`

	// For backward compatibility
	ServerUrl  string `envconfig:"PLUGIN_SERVER_URL"`
	Filename   string `envconfig:"PLUGIN_FILENAME"`
	Format     string `envconfig:"PLUGIN_FORMAT"`
	Attributes string `envconfig:"PLUGIN_ATTRIBUTES"`
}

type Artifact struct {
	File       string `yaml:"file"`
	Classifier string `yaml:"classifier"`
	ArtifactId string `yaml:"artifactId"`
	Type       string `yaml:"type"`
	Version    string `yaml:"version"`
	GroupId    string `yaml:"groupId"`
}

func GetNewPlugin(ctx context.Context, args Args) (Plugin, error) {

	nxp := GetNewNexusPlugin()
	return &nxp, nil
}

func Exec(ctx context.Context, args Args) (Plugin, error) {

	plugin, err := GetNewPlugin(ctx, args)
	if err != nil {
		return plugin, err
	}

	err = plugin.Init(&args)
	if err != nil {
		return plugin, err
	}
	defer func(p Plugin) {
		err := p.DeInit()
		if err != nil {
			LogPrintln(p, "Error in DeInit: "+err.Error())
		}
	}(plugin)

	err = plugin.ValidateAndProcessArgs(args)
	if err != nil {
		return plugin, err
	}

	err = plugin.DoPostArgsValidationSetup(args)
	if err != nil {
		return plugin, err
	}

	err = plugin.Run()

	err2 := plugin.WriteOutputVariables()
	if err2 != nil {
		LogPrintln(plugin, "Writing output variable UPLOAD_STATUS failed "+err2.Error())
	}
	if err != nil {
		LogPrintln(plugin, "Upload failed "+err.Error())
		return plugin, err
	}

	err = plugin.PersistResults()
	if err != nil {
		return plugin, err
	}

	return plugin, nil
}
