// Copyright 2020 the Drone Authors. All rights reserved.
// Use of this source code is governed by the Blue Oak Model License
// that can be found in the LICENSE file.

package plugin_defs

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/sirupsen/logrus"
)

func GetNewError(s string) error {
	return errors.New(s)
}

func LogPrintln(p Plugin, args ...interface{}) {

	if !IsDevTestingMode() {
		return
	}

	if p != nil {
		if p.IsQuiet() {
			return
		}
	}

	logrus.Println(append([]interface{}{"Plugin Info:"}, args...)...)
}

func LogPrintf(p Plugin, format string, v ...interface{}) {

	if !IsDevTestingMode() {
		return
	}

	if p != nil {
		if p.IsQuiet() {
			return
		}
	}
	logrus.Printf(format, v...)
}

func IsDirExists(dir string) (bool, error) {
	info, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return false, err
	}
	return info.IsDir(), nil
}

func CreateDir(absolutePath string) error {
	if absolutePath == "" || absolutePath == "." || absolutePath == ".." {
		return nil
	}

	err := os.MkdirAll(absolutePath, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create directory %s: %w", absolutePath, err)
	}
	return nil
}

func GetOutputVariablesStorageFilePath() string {
	if IsDevTestingMode() {
		return filepath.Join("/tmp", "drone-output")
	}
	return os.Getenv("DRONE_OUTPUT")
}

func ReadFileAsString(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func WriteEnvVariableAsString(key string, value interface{}) error {

	if GetOutputVariablesStorageFilePath() == "" {
		return GetNewError("Output file path is empty, check env var DRONE_OUTPUT")
	}

	outputFile, err := os.OpenFile(GetOutputVariablesStorageFilePath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open output file: %w", err)
	}
	defer outputFile.Close()

	valueStr := fmt.Sprintf("%v", value)

	_, err = fmt.Fprintf(outputFile, "%s=%s\n", key, valueStr)
	if err != nil {
		return fmt.Errorf("failed to write to env: %w", err)
	}

	return nil
}

func IsDevTestingMode() bool {
	return os.Getenv("DEV_TEST_d6c9b463090c") == "true"
}

func StructToJSONWithEnvKeys(v interface{}) (string, error) {
	val := reflect.ValueOf(v)
	typ := reflect.TypeOf(v)

	data := make(map[string]interface{})

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		key := field.Tag.Get("envconfig")
		if key != "" {
			data[key] = val.Field(i).Interface()
		}
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}

	return string(jsonData), nil
}

func GetTestWorkSpaceDir() string {

	nexusWorkSpaceDir := os.Getenv(DefaultWorkSpaceDirEnvVarKey)
	if nexusWorkSpaceDir == "" {
		nexusWorkSpaceDir = TestWorkSpaceDir
	}

	return nexusWorkSpaceDir
}

func GetTestBuildRootDir() string {
	return GetTestWorkSpaceDir()
}

const (
	DefaultWorkSpaceDirEnvVarKey = "DRONE_WORKSPACE"
	TestWorkSpaceDir             = "../test/tmp_workspace"
)

//
//
