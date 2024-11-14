// Copyright 2020 the Drone Authors. All rights reserved.
// Use of this source code is governed by the Blue Oak Model License
// that can be found in the LICENSE file.

package plugin

import (
	"context"
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
