package companion

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"github.com/OpenFactorioServerManager/factorio-server-manager/bootstrap"
)

// ModName is the Factorio mod name of the embedded companion. It is kept in step
// with the name in the embedded mod/info.json.
const ModName = "fsm-companion"

// modFiles carries the companion mod inside the binary. The mod source lives in
// the companion-mod repository; scripts/sync-companion.sh copies it into mod/.
//
//go:embed mod
var modFiles embed.FS

type modInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type modListEntry struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

type modList struct {
	Mods []modListEntry `json:"mods"`
}

// EnsureInstalled writes the embedded companion mod into the Factorio mods
// directory and marks it enabled in mod-list.json. It runs on every startup, so
// the mod cannot be dropped from the running server. Removing it from the mods
// directory only disables the manager's companion features, nothing else.
func EnsureInstalled() error {
	config := bootstrap.GetConfig()

	if err := os.MkdirAll(config.FactorioModsDir, 0o755); err != nil {
		return err
	}

	version, err := embeddedVersion()
	if err != nil {
		return err
	}

	dest := filepath.Join(config.FactorioModsDir, ModName+"_"+version)
	if err := writeModFiles(dest); err != nil {
		return err
	}

	if err := enableMod(config.FactorioModsDir); err != nil {
		return err
	}

	log.Printf("companion: %s %s installed and enabled", ModName, version)

	return nil
}

func embeddedVersion() (string, error) {
	raw, err := fs.ReadFile(modFiles, "mod/info.json")
	if err != nil {
		return "", err
	}

	var info modInfo
	if err := json.Unmarshal(raw, &info); err != nil {
		return "", err
	}

	return info.Version, nil
}

func writeModFiles(dest string) error {
	return fs.WalkDir(modFiles, "mod", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel("mod", path)
		if err != nil {
			return err
		}

		target := filepath.Join(dest, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}

		data, err := fs.ReadFile(modFiles, path)
		if err != nil {
			return err
		}

		return os.WriteFile(target, data, 0o644)
	})
}

func enableMod(modsDir string) error {
	listPath := filepath.Join(modsDir, "mod-list.json")

	list := modList{Mods: []modListEntry{{Name: "base", Enabled: true}}}
	raw, err := os.ReadFile(listPath)
	if err == nil {
		if err := json.Unmarshal(raw, &list); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	found := false
	for i := range list.Mods {
		if list.Mods[i].Name == ModName {
			list.Mods[i].Enabled = true
			found = true
			break
		}
	}
	if !found {
		list.Mods = append(list.Mods, modListEntry{Name: ModName, Enabled: true})
	}

	out, err := json.MarshalIndent(list, "", "    ")
	if err != nil {
		return err
	}

	return os.WriteFile(listPath, out, 0o664)
}
