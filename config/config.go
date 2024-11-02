package config

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"os"
)

type Config struct {
	MetadataDB  MySQL      `json:"metadata_db"`
	MappingIdDB MySQL      `json:"mapping_id_db"`
	QueryDB     ClickHouse `json:"query_db"`
	FileStore   S3         `json:"file_store"`
}

type ClickHouse struct {
	Database               string   `json:"database"`
	Username               string   `json:"username"`
	Password               string   `json:"password"`
	Addr                   []string `json:"addr"`
	Debug                  bool     `json:"debug"`
	MaxOpenConns           int      `json:"max_open_conns"`
	MaxIdleConns           int      `json:"max_idle_conns"`
	DialTimeoutSeconds     int      `json:"dial_timeout_seconds"`
	ConnMaxLifetimeSeconds int      `json:"conn_max_lifetime_seconds"`
}

type MySQL struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database string `json:"database"`
}

type S3 struct {
	Bucket            string `json:"bucket"`
	Region            string `json:"region"`
	AccessKeyID       string `json:"access_key_id"`
	SecretAccessKey   string `json:"secret_access_key"`
	ExpirationSeconds int64  `json:"expiration_seconds"`
}

func (mysql *MySQL) ToDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", mysql.Username, mysql.Password, mysql.Host, mysql.Port, mysql.Database)
}

func NewConfig() *Config {
	return &Config{
		MetadataDB: MySQL{
			Username: "admin",
			Password: "",
			Host:     "127.0.0.1",
			Port:     3306,
			Database: "metadata_db",
		},
		MappingIdDB: MySQL{
			Username: "admin",
			Password: "",
			Host:     "127.0.0.1",
			Port:     3306,
			Database: "mapping_id_db",
		},
		QueryDB: ClickHouse{
			Database:               "cdp_db",
			Addr:                   []string{"127.0.0.1:9000"},
			Debug:                  true,
			MaxOpenConns:           10,
			MaxIdleConns:           10,
			DialTimeoutSeconds:     10,
			ConnMaxLifetimeSeconds: 3600,
		},
		FileStore: S3{
			Bucket:            "cdp-file-store-test",
			Region:            "ap-southeast-1",
			AccessKeyID:       "",
			SecretAccessKey:   "",
			ExpirationSeconds: 7_776_000, // 3 months
		},
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
