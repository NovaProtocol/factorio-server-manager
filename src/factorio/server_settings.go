package factorio

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/OpenFactorioServerManager/factorio-server-manager/bootstrap"
)

// ServerSettingsDefaults reads the game's own server-settings.example.json. It is
// what the settings page shows as each field's default, the same way the mod
// settings page shows the default a mod declares.
func ServerSettingsDefaults() (map[string]interface{}, error) {
	config := bootstrap.GetConfig()
	path := filepath.Join(config.FactorioDir, "data", "server-settings.example.json")

	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("the game ships no %s: %v", path, err)
	}

	var defaults map[string]interface{}
	if err := json.Unmarshal(raw, &defaults); err != nil {
		return nil, fmt.Errorf("could not read %s: %v", path, err)
	}

	return defaults, nil
}
