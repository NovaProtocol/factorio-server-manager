package companion

import (
	"archive/zip"
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
// directory as a zip and adds it to mod-list.json enabled. It runs on every
// startup, so the mod is present and on by default without anyone having to
// install it.
//
// The zip matters: the manager lists installed mods by walking .zip files
// (factorio/mod_modInfo.go), so an unpacked directory would load in the game but
// stay invisible on the Mods page.
//
// A disable is deliberately left alone: an entry that already exists in
// mod-list.json keeps whatever state it has, so switching the companion off in
// the mod list (or through the manager's own toggle) survives a restart.
func EnsureInstalled() error {
	config := bootstrap.GetConfig()

	if err := os.MkdirAll(config.FactorioModsDir, 0o755); err != nil {
		return err
	}

	version, err := embeddedVersion()
	if err != nil {
		return err
	}

	if err := writeModZip(config.FactorioModsDir, version); err != nil {
		return err
	}

	// Earlier builds installed the mod unpacked. Two copies of the same mod
	// confuse Factorio, so drop the directory now that the zip is in place.
	stale := filepath.Join(config.FactorioModsDir, ModName+"_"+version)
	if err := os.RemoveAll(stale); err != nil {
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

// writeModZip packs the embedded mod into <mods>/<name>_<version>.zip, with the
// ModName_version/ prefix Factorio expects inside a mod archive.
func writeModZip(modsDir, version string) error {
	prefix := ModName + "_" + version
	dest := filepath.Join(modsDir, prefix+".zip")

	file, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer file.Close()

	archive := zip.NewWriter(file)

	err = fs.WalkDir(modFiles, "mod", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel("mod", path)
		if err != nil {
			return err
		}

		entry, err := archive.Create(prefix + "/" + filepath.ToSlash(rel))
		if err != nil {
			return err
		}

		data, err := fs.ReadFile(modFiles, path)
		if err != nil {
			return err
		}

		_, err = entry.Write(data)
		return err
	})
	if err != nil {
		archive.Close()
		return err
	}

	return archive.Close()
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

	for _, mod := range list.Mods {
		if mod.Name == ModName {
			// Already listed: keep the state the operator chose, even a disable.
			return nil
		}
	}

	list.Mods = append(list.Mods, modListEntry{Name: ModName, Enabled: true})

	out, err := json.MarshalIndent(list, "", "    ")
	if err != nil {
		return err
	}

	return os.WriteFile(listPath, out, 0o664)
}
