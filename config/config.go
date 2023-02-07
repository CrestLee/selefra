package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/selefra/selefra/pkg/utils"
	"github.com/selefra/selefra/ui"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/selefra/selefra/global"
)

const SELEFRA = "selefra"

const MODULES = "modules"

const PROVIDERS = "providers"

const VARIABLES = "variables"

const RULES = "rules"

var typeMap = map[string]bool{
	SELEFRA:   true,
	MODULES:   true,
	PROVIDERS: true,
	RULES:     true,
	VARIABLES: true,
}

type ProviderConfig struct {
	Name          string   `yaml:"name" json:"name"`
	Cache         string   `yaml:"cache" json:"cache"`
	Provider      string   `yaml:"provider" json:"provider"`
	MaxGoroutines uint64   `yaml:"max_goroutines" json:"max_goroutines"`
	Resources     []string `yaml:"resources" json:"resources"`
	LogLevel      string   `yaml:"log_level" json:"log_level"`
}

type Variable struct {
	Key         string `yaml:"key" json:"key"`
	Default     string `yaml:"default" json:"default"`
	Description string `yaml:"description" json:"description"`
	Author      string `yaml:"author" json:"author"`
}

// RootConfig is root config for selefra project
type RootConfig struct {
	Selefra   SelefraConfig `yaml:"selefra"`
	Providers yaml.Node     `yaml:"providers"`
	Variables []Variable    `yaml:"variables"`
}

type RootConfigInit struct {
	Selefra   SelefraConfigInit `yaml:"selefra"`
	Providers yaml.Node         `yaml:"providers"`
}

type RootConfigInitWithLogin struct {
	Selefra   SelefraConfigInitWithLogin `yaml:"selefra"`
	Providers yaml.Node                  `yaml:"providers"`
}

type RulesConfig struct {
	Rules []Rule `yaml:"rules"`
}

type Rule struct {
	Path     string                 `yaml:"path" json:"path"`
	Name     string                 `yaml:"name" json:"name"`
	Query    string                 `yaml:"query" json:"query"`
	Labels   map[string]interface{} `yaml:"labels" json:"labels"`
	Metadata struct {
		Id          string   `yaml:"id" json:"id"`
		Severity    string   `yaml:"severity" json:"severity"`
		Provider    string   `yaml:"provider" json:"provider"`
		Tags        []string `yaml:"tags" json:"tags"`
		Author      string   `yaml:"author" json:"author"`
		Remediation string   `yaml:"remediation" json:"remediation"`
		Title       string   `yaml:"title" json:"title"`
		Description string   `yaml:"description" json:"description"`
	}
	Output string `yaml:"output" json:"-"`
}

type ModuleConfig struct {
	Modules []Module `yaml:"modules" json:"modules"`
}

type Module struct {
	Name     string          `yaml:"name" json:"name"`
	Uses     []string        `yaml:"uses" json:"uses"`
	Children []*ModuleConfig `yaml:"-" json:"children"`
}

// Cloud is config for selefra cloud
// when user is login, cloud config exist, else not
type Cloud struct {
	Project      string `yaml:"project" mapstructure:"project"`
	Organization string `yaml:"organization" mapstructure:"organization"`
	HostName     string `yaml:"hostname" mapstructure:"hostname"`
}

// SelefraConfig is the project config
type SelefraConfig struct {
	Cloud      *Cloud              `yaml:"cloud" mapstructure:"cloud"`
	Name       string              `yaml:"name" mapstructure:"name"`
	CliVersion string              `yaml:"cli_version" mapstructure:"cli_version"`
	LogLevel   string              `yaml:"log_level" mapstructure:"log_level"`
	Providers  []*ProviderRequired `yaml:"providers" mapstructure:"providers"`
	//Connection *DB                 `yaml:"connection" mapstructure:"connection"`
}

// SelefraConfigInit is a subset for SelefraConfig without cloud config
type SelefraConfigInit struct {
	Name       string                  `yaml:"name" mapstructure:"name"`
	CliVersion string                  `yaml:"cli_version" mapstructure:"cli_version"`
	Providers  []*ProviderRequiredInit `yaml:"providers" mapstructure:"providers"`
}

// SelefraConfigInitWithLogin is a subset for SelefraConfig with a cloud config
type SelefraConfigInitWithLogin struct {
	Cloud      *Cloud                  `yaml:"cloud" mapstructure:"cloud"`
	Name       string                  `yaml:"name" mapstructure:"name"`
	CliVersion string                  `yaml:"cli_version" mapstructure:"cli_version"`
	Providers  []*ProviderRequiredInit `yaml:"providers" mapstructure:"providers"`
}

type ProviderRequired struct {
	Name    string  `yaml:"name,omitempty" json:"name,omitempty"`
	Source  *string `yaml:"source,omitempty" json:"source,omitempty"`
	Version string  `yaml:"version,omitempty" json:"version,omitempty"`
	Path    string  `yaml:"path" json:"path"`
}

type ProviderRequiredInit struct {
	Name    string  `yaml:"name,omitempty" json:"name,omitempty"`
	Source  *string `yaml:"source,omitempty" json:"source,omitempty"`
	Version string  `yaml:"version,omitempty" json:"version,omitempty"`
}

type DB struct {
	Driver string `yaml:"driver,omitempty" json:"driver,omitempty"`
	// These params are mutually exclusive with DSN
	Type     string   `yaml:"type,omitempty" json:"type,omitempty"`
	Username string   `yaml:"username,omitempty" json:"username,omitempty"`
	Password string   `yaml:"password,omitempty" json:"password,omitempty"`
	Host     string   `yaml:"host,omitempty" json:"host,omitempty"`
	Port     string   `yaml:"port,omitempty" json:"port,omitempty"`
	Database string   `yaml:"database,omitempty" json:"database,omitempty"`
	SSLMode  string   `yaml:"sslmode,omitempty" json:"sslmode,omitempty"`
	Extras   []string `yaml:"extras,omitempty" json:"extras,omitempty"`
}

type YamlKey int

type ConfigMap map[string]map[string]string
type FileMap map[string]string

func (c *SelefraConfig) GetHostName() string {
	if c.Cloud != nil && c.Cloud.HostName != "" {
		return c.Cloud.HostName
	}
	return "main-api.selefra.io"
}

func GetConfig() (*RootConfig, error) {
	if err := IsSelefra(); err != nil {
		return nil, err
	}

	return getConfig()
}

func getConfig() (c *RootConfig, err error) {
	config := viper.New()
	config.SetConfigType("yaml")
	clientByte, err := GetClientStr()
	if err != nil {
		return nil, err
	}
	err = config.ReadConfig(bytes.NewBuffer(clientByte))
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(clientByte, &c)
	if err != nil {
		return nil, err
	}
	global.SetLogLevel(c.Selefra.LogLevel)
	global.SetProjectName(c.Selefra.Name)

	if c.Selefra.Cloud != nil {
		global.SetRelvPrjName(c.Selefra.Cloud.Project)
	}

	global.SERVER = c.Selefra.GetHostName() // TODO: replace

	return c, nil
}

// GetAllConfig load all yaml config file in [dirname] and return a map filename => file_content
func GetAllConfig(dirname string, fileMap FileMap) (FileMap, error) {
	if fileMap == nil || len(fileMap) == 0 {
		fileMap = make(FileMap)
	}
	files, err := os.ReadDir(dirname)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		if file.IsDir() {
			_, err := GetAllConfig(filepath.Join(dirname, file.Name()), fileMap)
			if err != nil {
				return nil, err
			}
		} else {
			if path.Ext(file.Name()) == ".yaml" {
				b, err := os.ReadFile(filepath.Join(dirname, file.Name()))
				if err != nil {
					fmt.Println(err)
					return nil, err
				}
				fileMap[filepath.Join(dirname, file.Name())] = string(b)
			}
		}
	}
	return fileMap, nil
}

func GetCacheKey() string {
	return "update_time"
}

// GetSchemaKey return provider schema named <required.name>_<required_version>_<provider_name>
func GetSchemaKey(required *ProviderRequired, cp ProviderConfig) string {
	var pre string
	if required == nil {
		return pre + "public"
	}
	sourceArr := strings.Split(*required.Source, "/")
	source := strings.Replace(sourceArr[1]+"@"+required.Version, "/", "_", -1)
	source = strings.Replace(source, "@", "_", -1)
	source = strings.Replace(source, ".", "", -1)
	s := source + "_" + cp.Name
	return pre + s
}

var ErrNotSelefra = errors.New("this workspace is not selefra workspace")

// IsSelefra return an error when workspace is not a selefra workspace
func IsSelefra() error {
	configMap, err := readAllConfig(global.WorkSpace())
	if err != nil {
		return err
	}
	if configMap[SELEFRA] == nil {
		return ErrNotSelefra
	}
	return nil
}

// realAllConfig read all yaml file and store it in a map
func readAllConfig(dirname string) (ConfigMap, error) {
	var err error

	cm := make(ConfigMap)

	var fn func(dirname string)
	fn = func(dirname string) {
		files, err := os.ReadDir(dirname)
		if err != nil {
			err = err
			return
		}
		for _, file := range files {
			if file.IsDir() {
				fn(filepath.Join(dirname, file.Name()))
			} else {
				if path.Ext(file.Name()) == ".yaml" {
					f, _ := file.Info()
					_, err = readConfigFile(dirname, cm, f)
					if err != nil {
						err = err
						continue
					}
				}
			}
		}
	}

	fn(dirname)

	return cm, err
}

func readConfigFile(dirname string, configMap ConfigMap, file os.FileInfo) (ConfigMap, error) {
	b, err := os.ReadFile(filepath.Join(dirname, file.Name()))
	if err != nil {
		ui.Errorln(err)
		return nil, err
	}
	var node yaml.Node
	err = yaml.Unmarshal(b, &node)
	if len(node.Content) > 0 && node.Content[0] != nil && len(node.Content[0].Content) > 0 {
		for i := range node.Content[0].Content {
			if i%2 != 0 {
				continue
			}

			if typeMap[node.Content[0].Content[i].Value] {
				var strNode = yaml.Node{
					Kind: yaml.MappingNode,
					Content: []*yaml.Node{
						node.Content[0].Content[i],
						node.Content[0].Content[i+1],
					},
				}

				b, e := yaml.Marshal(strNode)
				if e != nil {
					ui.Errorln(e)
					return nil, err
				}
				if configMap[node.Content[0].Content[i].Value] == nil {
					configMap[node.Content[0].Content[i].Value] = make(map[string]string)
				}
				configMap[node.Content[0].Content[i].Value][filepath.Join(dirname, file.Name())] = string(b)
			}
		}
	}
	return configMap, nil
}

func assembleNode(configMap map[string]string) (*yaml.Node, map[string]*yaml.Node, error) {
	var baseNode *yaml.Node
	var nodeMap = make(map[string]*yaml.Node)
	for strPath, value := range configMap {
		if baseNode == nil {
			baseNode = new(yaml.Node)
			tempNode := new(yaml.Node)
			err := yaml.Unmarshal([]byte(value), baseNode)
			fmtNodePath(baseNode.Content[0].Content[1].Content, strPath, "uses")
			s, _ := yaml.Marshal(baseNode)
			_ = yaml.Unmarshal(s, tempNode)
			nodeMap[strPath] = tempNode
			if err != nil {
				return nil, nil, err
			}
		} else {
			var tempNode = new(yaml.Node)
			err := yaml.Unmarshal([]byte(value), tempNode)
			fmtNodePath(tempNode.Content[0].Content[1].Content, strPath, "uses")
			baseNode.Content[0].Content[1].Content = append(baseNode.Content[0].Content[1].Content, tempNode.Content[0].Content[1].Content...)
			nodeMap[strPath] = tempNode
			if err != nil {
				return nil, nil, err
			}
		}

	}

	return baseNode, nodeMap, nil
}

func fmtNodePath(nodes []*yaml.Node, path string, key string) {
	if key == "" {
		return
	}
	for i := range nodes {
		for ii := range nodes[i].Content {
			if nodes[i].Content[ii].Value == key {
				for iii := range nodes[i].Content[ii+1].Content {
					if strings.HasPrefix(nodes[i].Content[ii+1].Content[iii].Value, ".") {
						nodes[i].Content[ii+1].Value = filepath.Join(filepath.Dir(path), nodes[i].Content[ii+1].Value)
					}
				}
			}
		}
	}
}

var NoClient = errors.New("There is no selefra configuration！")

func GetClientStr() ([]byte, error) {
	configMap, err := readAllConfig(global.WorkSpace())
	if err != nil {
		return nil, err
	}

	if len(configMap[SELEFRA]) == 0 {
		return nil, NoClient
	}

	selefraNode, _, err := assembleNode(configMap[SELEFRA])
	if err != nil {
		return nil, err
	}

	providerNodes, _, err := assembleNode(configMap[PROVIDERS])
	if err != nil {
		return nil, err
	}

	variableNodes, _, err := assembleNode(configMap[VARIABLES])
	if err != nil {
		return nil, err
	}

	SelefraStr, err := yaml.Marshal(selefraNode)
	if err != nil {
		return nil, err
	}
	providerStr, err := yaml.Marshal(providerNodes)
	if err != nil {
		return nil, err
	}

	configStr := append(SelefraStr, providerStr...)
	if variableNodes != nil {
		variableStr, err := yaml.Marshal(variableNodes)
		if err != nil {
			return nil, err
		}
		configStr = append(configStr, variableStr...)
	}
	return configStr, nil
}

func GetModulesStr() ([]byte, error) {
	configMap, err := readAllConfig(global.WorkSpace())
	if err != nil {
		return nil, err
	}
	var paths []string
	for k := range configMap[MODULES] {
		paths = append(paths, k)
	}
	for i := range paths {
		getAllModules(configMap[MODULES], "", paths[i])
	}
	_, moduleMap, err := assembleNode(configMap[MODULES])
	err = deepPathModules(moduleMap)
	cyclePathMap, err := makeCyclePathMap(moduleMap)
	if err != nil {
		return nil, err
	}
	for cyclePath, paths := range cyclePathMap {
		var cyclePathList = []string{cyclePath}
		if checkCycle(cyclePathMap, cyclePath, paths, &cyclePathList) {
			cyclePathStr := strings.Join(cyclePathList, " -> ")
			return nil, errors.New("Modules have circular references:" + cyclePathStr)
		}
	}
	return makeUsesModule(moduleMap)
}

func checkModuleFile(configMap map[string]string, workspace string, waitUsePath string, file os.FileInfo) error {
	var b []byte
	var err error
	if strings.HasSuffix(waitUsePath, ".yaml") {
		b, err = os.ReadFile(waitUsePath)
	} else if strings.HasSuffix(file.Name(), ".yaml") {
		waitUsePath = filepath.Join(waitUsePath, file.Name())
		b, err = os.ReadFile(waitUsePath)
	} else {
		err = fmt.Errorf("the file name is not yaml:%s", waitUsePath)
	}
	if err != nil {
		ui.Errorln(err.Error())
		return err
	}
	if strings.Index(string(b), "modules:") > -1 {
		configMap[waitUsePath] = string(b)
		var module ModuleConfig
		err = yaml.Unmarshal(b, &module)
		if err != nil {
			ui.Errorln(err.Error())
			return err
		}
		for _, module := range module.Modules {
			for i := range module.Uses {
				getAllModules(configMap, workspace, module.Uses[i])
			}
		}
	}
	return nil
}

func getAllModules(configMap map[string]string, workspace, path string) {
	var waitUsePath string
	if strings.HasPrefix(path, "selefra/") {
		modulesName := strings.Split(path, "/")[1]
		modulePath, err := utils.GetHomeModulesPath(modulesName, "")
		if err != nil {
			ui.Errorln(err.Error())
		}
		waitUsePath = strings.Replace(path, "selefra", modulePath, 1)
		workspace = modulePath + "/" + modulesName
	} else if strings.HasPrefix(path, "app.selefra.io") {
		modulesArr := strings.Split(path, "/")
		modulesOrg := modulesArr[1]
		modulesName := modulesArr[2]
		modulePath, err := utils.GetHomeModulesPath(modulesName, modulesOrg)
		if err != nil {
			ui.Errorln(err.Error())
		}
		waitUsePath = strings.Replace(path, strings.Join(modulesArr[:2], "/"), modulePath, 1)
		workspace = modulePath + "/" + modulesName
	} else {
		waitUsePath = filepath.Join(workspace, path)
		if workspace == "" {
			workspace = global.WorkSpace()
		}
	}
	file, err := os.Stat(waitUsePath)
	if err != nil {
		ui.Errorln(err.Error())
		return
	}
	if file.IsDir() {
		files, err := os.ReadDir(waitUsePath)
		if err != nil {
			ui.Errorln(err.Error())
			return
		}
		for _, file := range files {
			f, err := file.Info()
			if err != nil {
				ui.Errorln(err.Error())
				continue
			}
			err = checkModuleFile(configMap, workspace, waitUsePath, f)
			if err != nil {
				ui.Errorln(err.Error())
				continue
			}
		}
	} else {
		err = checkModuleFile(configMap, workspace, waitUsePath, file)
		if err != nil {
			ui.Errorln(err.Error())
			return
		}
	}
}

func deepCopyYamlContent(node *yaml.Node) *yaml.Node {
	var tempNode = new(yaml.Node)
	s, _ := yaml.Marshal(node)
	_ = yaml.Unmarshal(s, tempNode)
	return tempNode.Content[0]
}

func deepPathModules(moduleMap map[string]*yaml.Node) error {
	for excludePath, node := range moduleMap {
		for i := range node.Content[0].Content[1].Content {
			var uses string
			for i2 := range node.Content[0].Content[1].Content[i].Content {
				if node.Content[0].Content[1].Content[i].Content[i2].Value == "uses" {
					uses = node.Content[0].Content[1].Content[i].Content[i2+1].Value
				}
			}
			if uses == "" {
				return errors.New("Module configuration error, missing uses")
			}
			file, err := os.Stat(uses)
			if os.IsNotExist(err) {
				return errors.New("Module file does not exist:" + uses)
			}
			if file.IsDir() {
				var paths []string
				files, err := os.ReadDir(uses)
				if err != nil {
					return errors.New("open dir failed:" + err.Error())
				}
				for _, file := range files {
					fileName := filepath.Join(uses, file.Name())
					if strings.HasSuffix(fileName, ".yaml") && fileName != excludePath {
						paths = append(paths, fileName)
					}
				}
				if len(paths) > 0 {
					tempNode := deepCopyYamlContent(node.Content[0].Content[1].Content[i])
					node.Content[0].Content[1].Content = append(node.Content[0].Content[1].Content[:i], node.Content[0].Content[1].Content[i+1:]...)
					for _, mPath := range paths {
						waitAppendNode := deepCopyYamlContent(tempNode)
						for i3 := range waitAppendNode.Content {
							if waitAppendNode.Content[i3].Value == "uses" {
								waitAppendNode.Content[i3+1].Value = mPath
							}
						}
						node.Content[0].Content[1].Content = append(node.Content[0].Content[1].Content, waitAppendNode)
					}
				}
			} else {
				fileName := file.Name()
				if !strings.HasSuffix(fileName, ".yaml") {
					return errors.New("Module file does not yaml:" + uses)
				}
			}
		}
	}
	return nil
}

func makeUsesModule(nodesMap map[string]*yaml.Node) ([]byte, error) {
	var usedModuleMap = make(map[string]bool)
	var ModulesMap = make(map[string]*ModuleConfig)
	var resultModules []Module
	for pathStr, node := range nodesMap {
		ModulesMap[pathStr] = new(ModuleConfig)
		nodeStr, err := yaml.Marshal(node)
		if err != nil {
			return nil, err
		}
		err = yaml.Unmarshal(nodeStr, ModulesMap[pathStr])
		if err != nil {
			return nil, err
		}
	}

	for _, moduleConfig := range ModulesMap {
		for i := range moduleConfig.Modules {
			for ii, use := range moduleConfig.Modules[i].Uses {
				if strings.HasPrefix(use, "selefra") {
					modulesName := strings.Split(use, "/")[1]
					modules, err := utils.GetHomeModulesPath(modulesName, "")
					if err != nil {
						return nil, err
					}
					moduleConfig.Modules[i].Uses[ii] = strings.Replace(use, "selefra", modules, 1)
				}
				if strings.HasPrefix(use, "app.selefra.io") {
					modulesArr := strings.Split(use, "/")
					modulesOrg := modulesArr[1]
					modulesName := modulesArr[2]
					modulePath, err := utils.GetHomeModulesPath(modulesName, modulesOrg)
					if err != nil {
						ui.Errorln(err.Error())
					}
					moduleConfig.Modules[i].Uses[ii] = strings.Replace(use, strings.Join(modulesArr[:2], "/"), modulePath, 1)
				}
			}
			for _, use := range moduleConfig.Modules[i].Uses {
				if ModulesMap[use] != nil {
					usedModuleMap[use] = true
					if path.IsAbs(use) {
						for i2 := range ModulesMap[use].Modules {
							mUses := ModulesMap[use].Modules[i2].Uses
							for i3 := range mUses {
								mUses[i3] = filepath.Join(filepath.Dir(use), mUses[i3])
							}
						}
					}
					moduleConfig.Modules[i].Children = append(moduleConfig.Modules[i].Children, ModulesMap[use])
				}
			}
		}
	}
	for s := range ModulesMap {
		if usedModuleMap[s] {
			continue
		}
		var tempModules = new(ModuleConfig)
		b, err := json.Marshal(ModulesMap[s])
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(b, tempModules)
		if err != nil {
			return nil, err
		}
		for i := range tempModules.Modules {
			resultModules = append(resultModules, deepFmtModules(&tempModules.Modules[i], usedModuleMap)...)
		}
	}

	var resultM = new(ModuleConfig)
	resultM.Modules = resultModules
	return yaml.Marshal(resultM)
}

func deepFmtModules(module *Module, usedModuleMap map[string]bool) []Module {
	var output []Module
	for i := 0; i < len(module.Uses); i++ {
		if usedModuleMap[module.Uses[i]] {
			module.Uses = append(module.Uses[:i], module.Uses[i+1:]...)
			i--
		}
	}
	if len(module.Children) != 0 {
		for i := range module.Children {
			for i2 := range module.Children[i].Modules {
				module.Children[i].Modules[i2].Name = module.Name + "." + module.Children[i].Modules[i2].Name
			}
			for i3 := range module.Children[i].Modules {
				output = append(output, deepFmtModules(&module.Children[i].Modules[i3], usedModuleMap)...)
			}
		}
	}
	output = append(output, *module)
	return output
}

func makeCyclePathMap(nodesMap map[string]*yaml.Node) (map[string][]string, error) {
	var userMap = make(map[string][]string)
	for modulePath, node := range nodesMap {
		userMap[modulePath] = make([]string, 0)
		var modules ModuleConfig
		nodeByte, err := yaml.Marshal(node)
		if err != nil {
			return nil, err
		}
		err = yaml.Unmarshal(nodeByte, &modules)
		if err != nil {
			return nil, err
		}
		for _, module := range modules.Modules {
			for _, use := range module.Uses {
				waitPath := use
				if nodesMap[waitPath] != nil {
					userMap[modulePath] = append(userMap[modulePath], waitPath)
				}
			}
		}
	}
	return userMap, nil
}

func checkCycle(cyclePathMap map[string][]string, path string, pathList []string, cyclePathList *[]string) bool {
	for _, p := range pathList {
		*cyclePathList = append(*cyclePathList, p)
		if p == path {
			return true
		}
		if checkCycle(cyclePathMap, path, cyclePathMap[p], cyclePathList) {
			return true
		}
		*cyclePathList = (*cyclePathList)[:len(*cyclePathList)-1]
	}
	return false
}

func GetConfigPath() (string, error) {

	configMap, err := readAllConfig(global.WorkSpace())
	if err != nil {
		return "", err
	}
	if err != nil {
		return "", err
	}

	clientMap := configMap[SELEFRA]
	for cofPath := range clientMap {
		return cofPath, nil
	}
	return "", errors.New("No config file found")
}

func GetRules() (RulesConfig, error) {
	var rules RulesConfig
	configMap, err := readAllConfig(global.WorkSpace())
	if err != nil {
		return rules, err
	}
	for rulePath, rule := range configMap[RULES] {
		var baseRule RulesConfig
		err := yaml.Unmarshal([]byte(rule), &baseRule)
		if err != nil {
			return RulesConfig{}, err
		}
		for i := range baseRule.Rules {
			baseRule.Rules[i].Path = rulePath
			ui.Successf("	%s - Rule %s: loading ... \n", rulePath, baseRule.Rules[i].Name)
		}
		rules.Rules = append(rules.Rules, baseRule.Rules...)
	}
	return rules, err
}

func (c *RootConfig) TestConfigByNode() error {
	configMap, err := readAllConfig(global.WorkSpace())
	if err != nil {
		return err
	}
	clientMap := configMap[SELEFRA]

	for pathStr, configStr := range clientMap {
		var selefraMap = make(map[string]*yaml.Node)
		selefraMap["cloud"] = new(yaml.Node)
		selefraMap["cli_version"] = nil
		selefraMap["name"] = nil
		selefraMap["connection"] = new(yaml.Node)
		selefraMap["log_level"] = new(yaml.Node)
		selefraMap["providers"] = nil
		bodyNode := new(yaml.Node)
		err := yaml.Unmarshal([]byte(configStr), bodyNode)
		if err != nil {
			return err
		}
		err = checkNode(selefraMap, bodyNode.Content[0].Content[1].Content, pathStr, "selefra:")
		if err != nil {
			return err
		}

		for index, node := range selefraMap["providers"].Content {
			var providersMap = make(map[string]*yaml.Node)
			providersMap["name"] = nil
			providersMap["source"] = nil
			providersMap["version"] = nil
			providersMap["path"] = new(yaml.Node)
			yamlPath := fmt.Sprintf("selefra.providers[%d]:", index)
			err = checkNode(providersMap, node.Content, pathStr, yamlPath)
			if err != nil {
				return err
			}
		}

	}

	modulesMap := configMap[MODULES]

	for pathStr, modulesStr := range modulesMap {
		var modulesNode = new(yaml.Node)
		err := yaml.Unmarshal([]byte(modulesStr), modulesNode)
		if err != nil {
			return err
		}
		for _, node := range modulesNode.Content[0].Content[1].Content {
			var ModuleMap = make(map[string]*yaml.Node)
			ModuleMap["name"] = nil
			ModuleMap["uses"] = nil
			ModuleMap["input"] = new(yaml.Node)

			err = checkNode(ModuleMap, node.Content, pathStr, "modules:")
			if err != nil {
				return err
			}
		}
	}

	rulesMap := configMap[RULES]
	for pathStr, rulesStr := range rulesMap {
		var rulesNode = new(yaml.Node)
		err := yaml.Unmarshal([]byte(rulesStr), rulesNode)
		if err != nil {
			return err
		}
		for index, node := range rulesNode.Content[0].Content[1].Content {
			var ruleMap = make(map[string]*yaml.Node)
			ruleMap["name"] = nil
			ruleMap["input"] = new(yaml.Node)
			ruleMap["query"] = nil
			ruleMap["labels"] = nil
			ruleMap["interval"] = new(yaml.Node)
			ruleMap["metadata"] = nil
			ruleMap["output"] = nil
			yamlPath := fmt.Sprintf("rules[%d]", index)
			err = checkNode(ruleMap, node.Content, pathStr, yamlPath+":")

			if err != nil {
				return err
			}

			for i := range ruleMap["input"].Content {
				if i%2 != 0 {
					var ruleInputMap = make(map[string]*yaml.Node)
					ruleInputMap["type"] = nil
					ruleInputMap["description"] = nil
					ruleInputMap["default"] = nil
					err = checkNode(ruleInputMap, ruleMap["input"].Content[i].Content, pathStr, yamlPath+"input:")
					if err != nil {
						return err
					}
				}
			}

			for i := range ruleMap["metadata"].Content {
				if i%2 != 0 {
					var ruleMetadataMap = make(map[string]*yaml.Node)
					ruleMetadataMap["id"] = nil
					ruleMetadataMap["severity"] = nil
					ruleMetadataMap["provider"] = nil
					ruleMetadataMap["tags"] = new(yaml.Node)
					ruleMetadataMap["remediation"] = nil
					ruleMetadataMap["title"] = nil
					ruleMetadataMap["author"] = nil
					ruleMetadataMap["description"] = nil
					err = checkNode(ruleMetadataMap, ruleMap["metadata"].Content, pathStr, yamlPath+"metadata:")
					if err != nil {
						return err
					}
				}
			}

		}
	}

	return nil
}

func hasKeys(key string, keys []string) bool {
	for _, v := range keys {
		if v == key {
			return true
		}
	}
	return false
}

func checkNode(configMap map[string]*yaml.Node, bodyNode []*yaml.Node, pathStr string, yamlPath string) error {
	var keys []string
	for s := range configMap {
		keys = append(keys, s)
	}
	for i := range bodyNode {
		if i == len(bodyNode)-1 || i%2 != 0 {
			continue
		}

		if !hasKeys(bodyNode[i].Value, keys) {
			errStr := fmt.Sprintf("Illegal configuration exists %s,Occurrence location %s %d:%d", bodyNode[i].Value, pathStr, bodyNode[i].Line, bodyNode[i].Column)
			return errors.New(errStr)
		}
		configMap[bodyNode[i].Value] = bodyNode[i+1]
	}
	for key, node := range configMap {
		if node == nil {
			errStr := fmt.Sprintf("%s %s Missing configuration %s", pathStr, yamlPath, key)
			return errors.New(errStr)
		}
	}
	return nil
}

func (c *RootConfig) GetConfigWithViper() (*viper.Viper, error) {
	config := viper.New()
	config.SetConfigType("yaml")
	clientByte, err := GetClientStr()
	if err != nil {
		return nil, err
	}
	err = config.ReadConfig(bytes.NewBuffer(clientByte))
	if err != nil {
		return config, err
	}
	err = yaml.Unmarshal(clientByte, &c)
	if err != nil {
		return nil, err
	}
	global.SetLogLevel(c.Selefra.LogLevel)
	global.SERVER = c.Selefra.GetHostName()
	return config, nil
}

func GetModules() ([]Module, error) {
	var modules ModuleConfig
	modulesStr, err := GetModulesStr()
	if err != nil {
		return modules.Modules, err
	}
	err = yaml.Unmarshal(modulesStr, &modules)
	if err != nil {
		return modules.Modules, err
	}

	return modules.Modules, nil
}
