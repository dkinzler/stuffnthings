package config

import (
	"backup/internal/fs"
	"backup/internal/github"
	"backup/internal/zip"
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	BackupDir string        `json:"backupDir"`
	Github    github.Config `json:"github"`
	Zip       zip.Config    `json:"zip"`
	Files     []string      `json:"files"`
}

func LoadConfig(file string) (Config, error) {
	var config Config

	if file != "" {
		s, err := os.ReadFile(file)
		if err != nil {
			return config, fmt.Errorf("could not read file: %w", err)
		}
		err = json.Unmarshal(s, &config)
		if err != nil {
			return config, fmt.Errorf("could not decode json: %w", err)
		}
	}

	// validate and set defaults
	if config.BackupDir == "" {
		backupDir, err := fs.DefaultBackupDir()
		if err != nil {
			return config, fmt.Errorf("could not get default backup directory: %w", err)
		}
		config.BackupDir = backupDir
	} else {
		absPath, err := fs.AbsPath(config.BackupDir)
		if err != nil {
			return config, fmt.Errorf("invalid directory: %w", err)
		}
		config.BackupDir = absPath
	}

	return config, nil
}
