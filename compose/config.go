package compose

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type Config struct {
	Services map[string]struct {
		Build BuildConfigOrString `yaml:"build"`
	} `yaml:"services"`
}

type BuildConfig struct {
	Context    string        `yaml:"context"`
	Dockerfile string        `yaml:"dockerfile"`
	Args       mapOrSlice    `yaml:"args"`
	CacheFrom  stringOrSlice `yaml:"cache_from"`
	Labels     mapOrSlice    `yaml:"labels"`
	Target     string        `yaml:"target"`
}

type BuildConfigOrString BuildConfig

func (c *BuildConfigOrString) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// try struct version first
	var conf BuildConfig
	if err := unmarshal(&conf); err == nil {
		fmt.Printf("Marshaled Build as struct\n")
		*c = BuildConfigOrString(conf)
		return nil
	}

	// next try the string version
	var buildStr string
	if err := unmarshal(&buildStr); err == nil {
		fmt.Printf("Marshaled Build as string\n")
		c.Context = buildStr
		// c.Dockerfile = filepath.Join(str, "Dockerfile")
		return nil
	}

	return fmt.Errorf("Failed to parse build config")
}

func ParseString(y string) (Config, error) {
	conf := Config{}
	err := yaml.Unmarshal([]byte(y), &conf)
	if err != nil {
		return conf, err
	}
	return conf, nil
}

func ParseFile(f string) (Config, error) {
	yamlFile, err := ioutil.ReadFile(f)
	if err != nil {
		return Config{}, err
	}
	return ParseString(string(yamlFile))
}

type mapOrSlice []string

func (c *mapOrSlice) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var sl []string
	if err := unmarshal(&sl); err == nil {
		*c = sl
		return nil
	}
	var m map[string]interface{}
	if err := unmarshal(&m); err != nil {
		return err
	}
	for k, v := range m {
		*c = append(*c, fmt.Sprintf("%s=%v", k, v))
	}
	return nil
}

type stringOrSlice []string

func (c *stringOrSlice) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var sl []string
	if err := unmarshal(&sl); err == nil {
		*c = sl
		return nil
	}
	var str string
	if err := unmarshal(&str); err != nil {
		return err
	}
	*c = []string{str}
	return nil
}
