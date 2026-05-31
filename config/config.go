package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/langgenius/dify-cli/types"
)

const ConfigFilename = ".dify_cli.json"

func GetConfigPath() (string, error) {
	if envPath := os.Getenv("DIFY_CLI_CONFIG"); envPath != "" {
		return envPath, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}
	return filepath.Join(cwd, ConfigFilename), nil
}

func GetSelfPath() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}
	return filepath.EvalSymlinks(execPath)
}

func Load() (*types.DifyConfig, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("config not found, run 'dify init' first: %w", err)
	}

	var cfg types.DifyConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &cfg, nil
}

func FindToolReference(symlinkName string) (*types.ToolReference, error) {
	cfg, err := Load()
	if err != nil {
		return nil, err
	}

	for _, ref := range cfg.ToolReferences {
		if GetReferenceSymlinkName(ref) == symlinkName {
			return &ref, nil
		}
	}

	parts := strings.SplitN(symlinkName, "_", 2)
	if len(parts) == 2 {
		shortID := parts[1]
		toolName := parts[0]
		for _, ref := range cfg.ToolReferences {
			if strings.HasPrefix(ref.ID, shortID) && ref.ToolName == toolName {
				return &ref, nil
			}
		}
	}

	return nil, fmt.Errorf("tool reference not found: %s (must use format: tool_name_uuid)", symlinkName)
}

func GetReferenceSymlinkName(ref types.ToolReference) string {
	return fmt.Sprintf("%s_%s", ref.ToolName, ref.ID)
}
