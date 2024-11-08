// Copyright 2020 the Drone Authors. All rights reserved.
// Use of this source code is governed by the Blue Oak Model License
// that can be found in the LICENSE file.

package plugin

import (
	"context"
	//nx "github.com/harness-community/drone-nexus-publish/plugin/nexus"
	//pd "github.com/harness-community/drone-nexus-publish/plugin/plugin_defs"
)

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
	if err != nil {
		return plugin, err
	}

	err = plugin.PersistResults()
	if err != nil {
		return plugin, err
	}

	err = plugin.WriteOutputVariables()
	if err != nil {
		return plugin, err
	}

	return plugin, nil
}
