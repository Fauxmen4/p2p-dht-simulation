package simulation

import (
	"os"
	"path"
	"strings"
)

// ConfigPath returns abs path to config by its name.
func ConfigPath(name string) string {
	if !strings.HasSuffix(name, ".yaml") {
		name = name + ".yaml"
	}
	dir, _ := os.Getwd()
	configPath := path.Join(dir, "configs", name)
	return configPath
}
