package plugin

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
	Pipeline
	EnvPluginInputArgs
	Level string `envconfig:"PLUGIN_LOG_LEVEL"`
}

type EnvPluginInputArgs struct {
	NexusVersion string `envconfig:"PLUGIN_NEXUS_VERSION"`
	NexusUrl     string `envconfig:"PLUGIN_NEXUS_URL"`
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
