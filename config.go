package main

import (
	"context"
	"errors"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	HttpPort             int            `yaml:"http_port" json:"http_port"`
	Endpoint             string         `yaml:"endpoint" json:"endpoint"`
	Secure               bool           `yaml:"secure" json:"secure"`
	AccessKey            string         `yaml:"access_key" json:"access_key"`
	SecretKey            string         `yaml:"secret_key" json:"secret_key"`
	Domains              []string       `yaml:"domains" json:"domains"`
	DefaultIndexDocument string         `yaml:"default_index_document" json:"default_index_document"`
	DefaultErrorDocument string         `yaml:"default_error_document" json:"default_error_document"`
	Policies             []BucketPolicy `yaml:"policies" json:"policies"`
}

type BucketPolicy struct {
	Bucket        string   `yaml:"bucket" json:"bucket"`
	DomainAlias   []string `yaml:"domain_alias" json:"domain_alias"`
	IndexDocument string   `yaml:"index_document" json:"index_document"`
	ErrorDocument string   `yaml:"error_document" json:"error_document"`
	AllowListing  bool     `yaml:"allow_listing" json:"allow_listing"`
}

var DefaultConfig = Config{
	HttpPort:             80,
	DefaultIndexDocument: "index.html",
	DefaultErrorDocument: "",
	Policies:             []BucketPolicy{},
}

func (c Config) Validate() error {
	if c.HttpPort == 0 {
		return errors.New("config http port empty")
	}
	if c.Endpoint == "" {
		return errors.New("config endpoint empty")
	}
	if len(c.Domains) == 0 {
		return errors.New("config domains empty")
	}
	for _, p := range c.Policies {
		if p.Bucket == "" {
			return errors.New("config bucket policy with empty bucket name")
		}
	}
	return nil
}

func ParseConfig(ctx context.Context, path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	conf := DefaultConfig
	if err := decoder.Decode(&conf); err != nil {
		return nil, err
	}
	if err := conf.Validate(); err != nil {
		return nil, err
	}
	return &conf, err
}
