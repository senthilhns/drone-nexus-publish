package plugin

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v2"
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
	Format     string
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

		if n.Version == "nexus2" {
			artifactURL := n.prepareNexus2ArtifactURL(artifact)
			if err := n.uploadFileNexus2(artifactURL, file); err != nil {
				n.addFailedArtifact(artifact, fmt.Sprintf("upload failed: %v", err))
				continue
			}
		} else if n.Version == "nexus3" {
			if err := n.uploadFileNexus3(artifact); err != nil {
				n.addFailedArtifact(artifact, fmt.Sprintf("upload failed: %v", err))
				continue
			}
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
		"username":     args.Username,
		"password":     args.Password,
		"protocol":     args.Protocol,
		"nexusUrl":     args.NexusUrl,
		"nexusVersion": args.NexusVersion,
		"repository":   args.Repository,
		"groupId":      args.GroupId,
		"format":       args.Format,
	}

	for field, value := range requiredArgs {
		if value == "" {
			return GetNewError("Error in IsMultiFileUploadArgsOk: " + field + " cannot be empty")
		}
	}

	n.UserName = args.Username
	n.Password = args.Password
	n.Repository = args.Repository
	n.ServerUrl = args.Protocol + "://" + args.NexusUrl
	n.GroupId = args.GroupId
	n.Version = args.NexusVersion
	n.Format = args.Format

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
		if artifact.Version == "" {
			missingFields = append(missingFields, "Version")
		}
		if artifact.GroupId == "" {
			artifact.GroupId = args.GroupId
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
	n.Format = args.Format
	n.GroupId = values["CgroupId"]
	n.Version = "nexus3"
	n.Artifacts = []Artifact{
		{
			File:       args.Filename,
			Classifier: values["Aclassifier"],
			ArtifactId: values["CartifactId"],
			Type:       values["Aextension"],
			Version:    values["Cversion"],
			GroupId:    values["CgroupId"],
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

func (n *NexusPlugin) prepareNexus2ArtifactURL(artifact Artifact) string {
	switch n.Format {
	case "maven2":
		return fmt.Sprintf("%s/repository/%s/%s/%s/%s/%s-%s.%s",
			n.ServerUrl, n.Repository, artifact.GroupId, artifact.ArtifactId, artifact.Version,
			artifact.ArtifactId, artifact.Version, artifact.Type)

	case "yum":
		return fmt.Sprintf("%s/repository/%s/%s/%s",
			n.ServerUrl, n.Repository, artifact.ArtifactId, artifact.Version)

	case "raw":
		return fmt.Sprintf("%s/repository/%s/%s/%s.%s",
			n.ServerUrl, n.Repository, artifact.GroupId, artifact.ArtifactId, artifact.Type)

	default:
		LogPrintln(n, "Unsupported format for direct upload:", n.Format)
		return ""
	}
}

func (n *NexusPlugin) uploadFileNexus2(url string, content io.Reader) error {
	req, err := http.NewRequest("PUT", url, content)
	if err != nil {
		return err
	}

	req.SetBasicAuth(n.UserName, n.Password)
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := n.HttpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("upload failed with status %d", resp.StatusCode)
	}
	return nil
}

func (n *NexusPlugin) uploadFileNexus3(artifact Artifact) error {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	var url string
	var assetFieldName string

	switch n.Format {
	case "maven2":
		_ = writer.WriteField("maven2.groupId", artifact.GroupId)
		_ = writer.WriteField("maven2.artifactId", artifact.ArtifactId)
		_ = writer.WriteField("maven2.version", artifact.Version)
		assetFieldName = "maven2.asset1"
		_ = writer.WriteField("maven2.asset1.extension", artifact.Type)

	case "raw":
		_ = writer.WriteField("raw.directory", artifact.GroupId)
		assetFieldName = "raw.asset1"
		_ = writer.WriteField("raw.asset1.filename", fmt.Sprintf("%s.%s", artifact.ArtifactId, artifact.Type))

	default:
		assetFieldName = fmt.Sprintf("%s.asset", n.Format)
	}

	fileWriter, err := writer.CreateFormFile(assetFieldName, artifact.File)
	if err != nil {
		return err
	}
	file, err := os.Open(artifact.File)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(fileWriter, file)
	if err != nil {
		return err
	}

	err = writer.Close()
	if err != nil {
		return err
	}

	url = fmt.Sprintf("%s/service/rest/v1/components?repository=%s", n.ServerUrl, n.Repository)

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return err
	}

	req.SetBasicAuth(n.UserName, n.Password)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := n.HttpClient.Do(req)
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
