package internal

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
)

func (a *App) initCfg() error {
	f, err := os.Open("config/config.yaml")
	if err != nil {
		return fmt.Errorf("open config.yaml: %v", err)
	}
	defer f.Close()
	d := yaml.NewDecoder(f)
	if err = d.Decode(&a.config); err != nil {
		return fmt.Errorf("decode config.yaml: %v", err)
	}
	return nil
}
