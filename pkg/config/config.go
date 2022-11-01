package config

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
)

var (
	configs *Config
)

func InitConfig(cfgfile string) {
	configs = new(Config)
	f, err := os.Stat(cfgfile)
	if err != nil {
		log.Fatal(err)
	}
	if f.IsDir() == true {
		log.Fatal("config file is a dir")
	}
	yamlconfig, err := ioutil.ReadFile(cfgfile)
	if err != nil {
		log.Fatal(err)
	}
	err = yaml.Unmarshal(yamlconfig, configs)
	if err != nil {
		log.Fatal(err)
	}
}

func GetConfig() *Config {
	return configs
}
