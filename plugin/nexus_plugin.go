package plugin

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
)

type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type NexusPlugin struct {
	InputArgs         *Args
	IsMultiFileUpload bool
	PluginProcessingInfo
	NexusPluginResponse
	HttpClient HttpClient
}

type PluginProcessingInfo struct {
	UserName   string
	Password   string
	ServerUrl  string
	Version    string
	Repository string
	GroupId    string
	Artifacts  []Artifact
}

type NexusPluginResponse struct {
	Failed []FailedArtifact `json:"failed"`
}

type FailedArtifact struct {
	File       string `json:"file"`
	ArtifactId string `json:"artifactId"`
	Err        string `json:"err"`
}

func (n *NexusPlugin) Run() error {
	LogPrintln(n, "Starting Nexus Plugin Run")

	if n.HttpClient == nil {
		n.HttpClient = &http.Client{}
	}

	for _, artifact := range n.Artifacts {
		filePath := artifact.File
		file, err := os.Open(filePath)
		if err != nil {
			n.addFailedArtifact(artifact, fmt.Sprintf("could not open file: %v", err))
			continue
		}
		defer file.Close()

		// Prepare URLs for artifact and checksum uploads
		artifactURL := n.prepareArtifactURLs(artifact)

		// Upload the main artifact
		if err := n.uploadFile(n.HttpClient, artifactURL, file); err != nil {
			n.addFailedArtifact(artifact, fmt.Sprintf("upload failed for artifact: %v", err))
			continue
		}

		LogPrintln(n, "Successfully uploaded artifact:", artifact.File)
	}

	if len(n.Failed) > 0 {
		return GetNewError("NexusPlugin Error in Run: some artifacts failed to upload")
	}

	return nil
}

func (n *NexusPlugin) WriteOutputVariables() error {

	type EnvKvPair struct {
		Key   string
		Value interface{}
	}

	var kvPairs = []EnvKvPair{
		{Key: "UPLOAD_STATUS", Value: n.Failed},
	}

	var retErr error = nil

	for _, kvPair := range kvPairs {
		err := WriteEnvVariableAsString(kvPair.Key, kvPair.Value)
		if err != nil {
			retErr = err
		}
	}

	return retErr
}

func (n *NexusPlugin) Init(args *Args) error {
	n.InputArgs = args
	return nil
}

func (n *NexusPlugin) SetBuildRoot(buildRootPath string) error {
	return nil
}

func (n *NexusPlugin) DeInit() error {
	return nil
}

func (n *NexusPlugin) ValidateAndProcessArgs(args Args) error {
	LogPrintln(n, "NexusPlugin BuildAndValidateArgs")

	err := n.DetermineIsMultiFileUpload(args)
	if err != nil {
		LogPrintln(n, "NexusPlugin Error in ValidateAndProcessArgs: "+err.Error())
		return err
	}

	if n.IsMultiFileUpload {
		err = n.IsMultiFileUploadArgsOk(args)
		if err != nil {
			LogPrintln(n, "NexusPlugin Error in ValidateAndProcessArgs: "+err.Error())
			return err
		}
	} else {
		err = n.IsSingleFileUploadArgsOk(args)
		if err != nil {
			LogPrintln(n, "NexusPlugin Error in ValidateAndProcessArgs: "+err.Error())
			return err
		}
	}

	return nil
}

func (n *NexusPlugin) DetermineIsMultiFileUpload(args Args) error {
	LogPrintln(n, "NexusPlugin DetermineIsMultiFileUpload")

	switch {
	case args.Attributes != "" && args.Artifact == "":
		n.IsMultiFileUpload = false
	case args.Artifact != "" && args.Attributes == "":
		n.IsMultiFileUpload = true
	case args.Attributes == "" && args.Artifact == "":
		return GetNewError("Error in DetermineCompatibilityMode: both 'Attributes' and 'Artifact' cannot be empty")
	default:
		return GetNewError("Error in DetermineCompatibilityMode: both 'Attributes' and 'Artifact' provided, which is ambiguous")
	}

	return nil
}

func (n *NexusPlugin) IsMultiFileUploadArgsOk(args Args) error {
	LogPrintln(n, "NexusPlugin IsMultiFileUploadArgsOk")

	requiredArgs := map[string]string{
		"username":      args.Username,
		"credentialsId": args.CredentialsId,
		"protocol":      args.Protocol,
		"nexusUrl":      args.NexusUrl,
		"nexusVersion":  args.NexusVersion,
		"repository":    args.Repository,
		"groupId":       args.GroupId,
	}

	for field, value := range requiredArgs {
		if value == "" {
			return GetNewError("Error in IsMultiFileUploadArgsOk: " + field + " cannot be empty")
		}
	}

	n.UserName = args.Username
	n.Password = args.CredentialsId
	n.Repository = args.Repository
	n.ServerUrl = args.Protocol + "://" + args.NexusUrl
	n.GroupId = args.GroupId
	n.Version = args.NexusVersion

	// Unmarshalling YAML artifact data
	var artifacts []Artifact
	if err := yaml.Unmarshal([]byte(args.Artifact), &artifacts); err != nil {
		return GetNewError("Error in IsMultiFileUploadArgsOk: Error decoding YAML: " + err.Error())
	}

	var filteredArtifacts []Artifact
	for _, artifact := range artifacts {
		missingFields := []string{}
		if artifact.ArtifactId == "" {
			missingFields = append(missingFields, "ArtifactId")
		}
		if artifact.File == "" {
			missingFields = append(missingFields, "File")
		}
		if artifact.Type == "" {
			missingFields = append(missingFields, "Type")
		}

		if len(missingFields) > 0 {
			n.addFailedArtifact(artifact, fmt.Sprintf("Missing fields: %s", strings.Join(missingFields, ", ")))
		} else {
			// Add to filtered list if all fields are valid
			filteredArtifacts = append(filteredArtifacts, artifact)
		}
	}

	n.Artifacts = filteredArtifacts
	return nil
}

func (n *NexusPlugin) IsSingleFileUploadArgsOk(args Args) error {
	LogPrintln(n, "NexusPlugin IsSingleFileUploadArgsOk")

	requiredArgs := map[string]string{
		"Username":   args.Username,
		"Password":   args.Password,
		"ServerUrl":  args.ServerUrl,
		"Filename":   args.Filename,
		"Format":     args.Format,
		"Repository": args.Repository,
	}

	for field, value := range requiredArgs {
		if value == "" {
			return GetNewError("Error in IsSingleFileUploadArgsOk: " + field + " cannot be empty")
		}
	}

	requiredFields := []string{"CgroupId", "Cversion", "Aextension", "Aclassifier"}
	values := make(map[string]string)

	pattern := regexp.MustCompile(`-(CgroupId|CartifactId|Cversion|Aextension|Aclassifier)=(\S+)`)
	matches := pattern.FindAllStringSubmatch(args.Attributes, -1)

	for _, match := range matches {
		if len(match) == 3 {
			values[match[1]] = match[2]
		}
	}

	// Check if all required fields are present
	for _, field := range requiredFields {
		if values[field] == "" {
			return GetNewError("Error in IsSingleFileUploadArgsOk: " + field + " cannot be empty")
		}
	}
	n.UserName = args.Username
	n.Password = args.Password
	n.Repository = args.Repository
	n.ServerUrl = args.ServerUrl
	n.GroupId = values["CgroupId"]
	n.Version = values["Cversion"]
	n.Artifacts = []Artifact{
		{
			File:       args.Filename,
			Classifier: values["Aclassifier"],
			ArtifactId: values["CartifactId"],
			Type:       values["Aextension"],
		},
	}

	return nil
}

func (n *NexusPlugin) DoPostArgsValidationSetup(args Args) error {
	return nil
}

func (n *NexusPlugin) PersistResults() error {
	return nil
}

func (n *NexusPlugin) IsQuiet() bool {
	return false
}

func (n *NexusPlugin) InspectProcessArgs(argNamesList []string) (map[string]interface{}, error) {
	return nil, nil
}

func GetNewNexusPlugin() NexusPlugin {
	return NexusPlugin{}
}

func (n *NexusPlugin) prepareArtifactURLs(artifact Artifact) string {
	baseURL := fmt.Sprintf("%s/repository/%s/%s/%s/%s/%s-%s",
		n.ServerUrl, n.Repository, n.GroupId, artifact.ArtifactId, n.Version,
		artifact.ArtifactId, n.Version)

	return fmt.Sprintf("%s.%s", baseURL, artifact.Type)
}

func (n *NexusPlugin) uploadFile(httpClient HttpClient, url string, content io.Reader) error {
	req, err := http.NewRequest("PUT", url, content)
	if err != nil {
		return err
	}

	req.SetBasicAuth(n.UserName, n.Password)
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := httpClient.Do(req) // Using the HttpClient interface here
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("upload failed with status %d", resp.StatusCode)
	}
	return nil
}

func (n *NexusPlugin) addFailedArtifact(artifact Artifact, errMsg string) {
	n.Failed = append(n.Failed, FailedArtifact{
		File:       artifact.File,
		ArtifactId: artifact.ArtifactId,
		Err:        errMsg,
	})
}
