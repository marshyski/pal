package config

import (
	"errors"
	"os"
	"path/filepath"

	cmap "github.com/orcaman/concurrent-map"
	"github.com/perlogix/pal/utils"
	"gopkg.in/yaml.v3"
)

var (
	resourceMap = cmap.New()
)

type Config struct {
	HTTP struct {
		Listen     string `yaml:"listen"`
		TimeoutMin int    `yaml:"timeout_min"`
		Key        string `yaml:"key"`
		Cert       string `yaml:"cert"`
	} `yaml:"http"`
	Store struct {
		EncryptKey string `yaml:"encrypt_key"`
		AuthHeader string `yaml:"auth_header"`
		DBPath     string `yaml:"db_path"`
	} `yaml:"store"`
}

func InitConfig(location string) error {
	if !utils.FileExists(location) {
		return errors.New("error file does not exist: " + location)
	}

	configFile, err := os.ReadFile(filepath.Clean(location))
	if err != nil {
		return err
	}

	config := &Config{}

	err = yaml.Unmarshal(configFile, config)
	if err != nil {
		return err
	}

	resourceMap.Set("http_cert", config.HTTP.Cert)
	resourceMap.Set("http_key", config.HTTP.Key)
	resourceMap.Set("http_listen", config.HTTP.Listen)
	resourceMap.Set("http_timeout_min", config.HTTP.TimeoutMin)
	resourceMap.Set("store_db_path", config.Store.DBPath)
	resourceMap.Set("store_encrypt_key", config.Store.EncryptKey)
	resourceMap.Set("store_auth_header", config.Store.AuthHeader)

	return nil
}

func GetConfigStr(key string) string {
	val, _ := resourceMap.Get(key)
	return val.(string)
}

func GetConfigInt(key string) int {
	val, _ := resourceMap.Get(key)
	return val.(int)
}
