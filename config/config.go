package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/user"
	"time"

	"third_party/code.google.com/p/goauth2/oauth"
)

const (
	GoogleOAuth2AuthURL  = "https://accounts.google.com/o/oauth2/auth"
	GoogleOAuth2TokenURL = "https://accounts.google.com/o/oauth2/token"
)

type credentialsType struct {
	ClientId     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RefreshToken string `json:"refresh_token"`
}

type configType struct {
	Credentials credentialsType
	BlobPath    string `json:"blob_path,omitempty"`
	MountPath   string `json:"mount_path,omitempty"`
}

type Config struct {
	path string
	cfg  *configType
}

func New(path string) (*Config, error) {
	if path == "" {
		usr, err := user.Current()
		if err != nil {
			return nil, err
		}
		path = fmt.Sprintf("%s%c%s", usr.HomeDir, os.PathSeparator, ".googledrive")
	}
	c := &Config{path: path}
	c.setup()
	// read and unmarshall configuration file
	if err := c.readFromFile(); err != nil {
		return nil, err
	}
	return c, nil
}

// TODO(burcu): Doesn't belong to this package, move somewhere else
func (c *Config) GetDefaultTransport() *oauth.Transport {
	oauthConf := &oauth.Config{
		ClientId:     c.cfg.Credentials.ClientId,
		ClientSecret: c.cfg.Credentials.ClientSecret,
		AuthURL:      GoogleOAuth2AuthURL,
		TokenURL:     GoogleOAuth2TokenURL,
	}
	// force refreshes the access token on start, make sure
	// refresh request in parallel are being started
	return &oauth.Transport{
		Token:     &oauth.Token{RefreshToken: c.cfg.Credentials.RefreshToken, Expiry: time.Now()},
		Config:    oauthConf,
		Transport: http.DefaultTransport,
	}
}

func (c *Config) GetConfigPath() string {
	return fmt.Sprintf("%s%c%s", c.path, os.PathSeparator, "config.json")
}

func (c *Config) GetMetadataPath() string {
	return fmt.Sprintf("%s%c%s", c.path, os.PathSeparator, "meta.sql")
}

// TODO: blob path should be able to set absolutely
func (c *Config) GetBlobPath() string {
	return fmt.Sprintf("%s%c%s", c.path, os.PathSeparator, "blob")
}

func (c *Config) GetMountPoint() string {
	return c.cfg.MountPath
}

func (c *Config) setup() error {
	// TODO(burcud): Initialize with a sample config.
	return os.MkdirAll(c.GetBlobPath(), 0750)
}

func (c *Config) readFromFile() (err error) {
	log.Println("Reading configuration file...")
	var content []byte
	if content, err = ioutil.ReadFile(c.GetConfigPath()); err != nil {
		return
	}
	var cfg configType
	if err = json.Unmarshal(content, &cfg); err != nil {
		return
	}
	c.cfg = &cfg
	return nil
}
