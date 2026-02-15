package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// yamlConfig — структура config.yaml (режимы, лимиты, пути).
type yamlConfig struct {
	Mode            string `yaml:"mode"`
	Source          string `yaml:"source"`
	ChunkTokens     int    `yaml:"chunk_tokens"`
	LLMConcurrency  int    `yaml:"llm_concurrency"`
	Model           string `yaml:"model"`
	CacheDir        string `yaml:"cache_dir"`
	CredentialsPath string `yaml:"credentials_path"`
	LocalDir        string `yaml:"local_dir"` // при source: local
}

// loadYAML загружает config.yaml из path; если файла нет — nil, nil.
func loadYAML(path string) (*yamlConfig, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var c yamlConfig
	if err := yaml.Unmarshal(raw, &c); err != nil {
		return nil, err
	}
	return &c, nil
}
