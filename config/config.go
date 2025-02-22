package config

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"os"
)

type Config struct {
	MetadataDB        MySQL         `json:"metadata_db"`
	QueryDB           ElasticSearch `json:"query_db"`
	FileStore         GoogleDrive   `json:"file_store"`
	SMTP              Brevo         `json:"smtp"`
	WebPage           WebPage       `json:"web_page"`
	InternalSender    string        `json:"internal_sender"`
	TrialAccountToken string        `json:"trial_account_token"`
}

type ElasticSearch struct {
	Addr                 []string `json:"addr"`
	Username             string   `json:"username"`
	Password             string   `json:"password"`
	NumWorkers           int      `json:"num_workers"`
	FlushBytes           int      `json:"flush_bytes"`
	FlushInternalSeconds int      `json:"flush_internal_seconds"`
	ScrollTimeoutSeconds int      `json:"scroll_timeout_seconds"`
}

type GoogleDrive struct {
	BaseFolderID string `json:"base_folder_id"`
	AdminEmail   string `json:"admin_email"`

	GoogleServiceAccount struct {
		Type                string `json:"type"`
		ProjectID           string `json:"project_id"`
		PrivateKeyID        string `json:"private_key_id"`
		PrivateKey          string `json:"private_key"`
		ClientEmail         string `json:"client_email"`
		ClientID            string `json:"client_id"`
		AuthURI             string `json:"auth_uri"`
		TokenURI            string `json:"token_uri"`
		AuthProviderCertURL string `json:"auth_provider_x509_cert_url"`
		ClientCertURL       string `json:"client_x509_cert_url"`
		UniverseDomain      string `json:"universe_domain"`
	} `json:"google_service_account"`
}

type GoogleServiceAccount struct {
	Type                string `json:"type"`
	ProjectID           string `json:"project_id"`
	PrivateKeyID        string `json:"private_key_id"`
	PrivateKey          string `json:"private_key"`
	ClientEmail         string `json:"client_email"`
	ClientID            string `json:"client_id"`
	AuthURI             string `json:"auth_uri"`
	TokenURI            string `json:"token_uri"`
	AuthProviderCertURL string `json:"auth_provider_x509_cert_url"`
	ClientCertURL       string `json:"client_x509_cert_url"`
	UniverseDomain      string `json:"universe_domain"`
}

type WebPage struct {
	Domain string `json:"domain"`
	Paths  Paths  `json:"paths"`
}

type Paths struct {
	InitUser string `json:"init_user"`
}

type Brevo struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	APIKey   string `json:"api_key"`
}

type MySQL struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database string `json:"database"`
}

func (mysql *MySQL) ToDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", mysql.Username, mysql.Password, mysql.Host, mysql.Port, mysql.Database)
}

func NewConfig() *Config {
	return &Config{
		MetadataDB: MySQL{
			Username: "",
			Password: "",
			Host:     "127.0.0.1",
			Port:     3306,
			Database: "metadata_db",
		},
		QueryDB: ElasticSearch{
			Addr:     []string{},
			Username: "",
			Password: "",
		},
		FileStore: GoogleDrive{
			BaseFolderID: "",
			AdminEmail:   "",
		},
		SMTP: Brevo{
			Host:     "127.0.0.1",
			Port:     25,
			Username: "",
			Password: "",
			APIKey:   "",
		},
		InternalSender: "",
		WebPage: WebPage{
			Domain: "http://localhost:3000",
			Paths: Paths{
				InitUser: "/user/init",
			},
		},
		TrialAccountToken: "",
	}
}

func (c *Config) Load(ctx context.Context, path string) error {
	if path == "" {
		log.Ctx(ctx).Warn().Msgf("empty config file")
		return nil
	}

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Ctx(ctx).Warn().Msgf("config file does not exist, file path: %s", path)
			return nil
		}
		return err
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			log.Ctx(ctx).Error().Msgf("config file close failed, file path: %s", path)
		}
	}(f)

	p := json.NewDecoder(f)
	if err := p.Decode(&c); err != nil {
		return err
	}

	return nil
}
