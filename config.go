package main

import "log"
import "flag"
import "io/ioutil"
import "path/filepath"
import "gopkg.in/yaml.v2"

type configuration struct {
	DatabaseName        string `yaml:"DatabaseName"`
	DatabaseUser        string `yaml:"DatabaseUser"`
	DatabasePassword    string `yaml:"DatabasePassword"`
	WebHost             string `yaml:"WebHost"`
	WebPort             string `yaml:"WebPort"`
	SecureCookie        bool   `yaml:"SecureCookie"`
	TLS                 bool   `yaml:"TLS"`
	Cert                string `yaml:"PathToCert"`
	Key                 string `yaml:"PathToKey"`
	WebRoot             string `yaml:"PathToWebRoot"`
	UploadPath          string `yaml:"UploadPath"`
	CsrfToken           string `yaml:"CsrfToken"`
	Delay               int64  `yaml:"Delay"`
	DebugLevel          int    `yaml:"DebugLevel"`
	OauthConsumerKey    string `yaml:"OauthConsumerKey"`
	OauthConsumerSecret string `yaml:"OauthConsumerSecret"`
	OauthToken          string `yaml:"OauthToken"`
	OauthTokenSecret    string `yaml:"OauthTokenSecret"`
}

var config configuration

func readConfig() {
	var configPath = flag.String("config", "/etc/kereru/config.yml", "Path to the config file")
	flag.Parse()

	ConfigFilename, err := filepath.Abs(*configPath)
	if err != nil {
		log.Fatal("Cant read config: ", err)
	}

	yamlFile, err := ioutil.ReadFile(ConfigFilename)
	if err != nil {
		log.Fatal("Cant read config: ", err)
	}

	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		log.Fatal("Cant read config: ", err)
	}
}
