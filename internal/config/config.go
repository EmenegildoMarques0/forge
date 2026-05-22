package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config define a estrutura do ficheiro de configuração do Forge
type Config struct {
	GeminiAPIKey string `json:"gemini_api_key"`
}

// getConfigPath retorna o caminho para o ficheiro oculto na home do usuário (~/.forge_config.json)
func getConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".forge_config.json"), nil
}

// Load carrega as configurações do ficheiro local. Se não existir, retorna uma config vazia.
func Load() (Config, error) {
	var cfg Config

	path, err := getConfigPath()
	if err != nil {
		return cfg, err
	}

	// Se o ficheiro não existe, retorna a estrutura limpa sem erro
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}

	err = json.Unmarshal(data, &cfg)
	return cfg, err
}

// Save grava a API Key do Gemini no ficheiro de configuração local
func Save(apiKey string) error {
	cfg := Config{
		GeminiAPIKey: apiKey,
	}

	path, err := getConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}
