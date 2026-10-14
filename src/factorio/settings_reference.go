package factorio

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/OpenFactorioServerManager/factorio-server-manager/bootstrap"
)

// SettingDeclaration is what the game reports about one setting: its type, the
// section it belongs to, its default and any bounds. It comes from the dump the
// game writes with --dump-data, which is the same information the game works
// from, so none of it is inferred.
type SettingDeclaration struct {
	// Mod is the mod the setting came from, worked out from whose locale file
	// named it, since the dump does not attribute settings to mods.
	Mod                  string      `json:"mod,omitempty"`
	Type                 string      `json:"type"`
	SettingType          string      `json:"setting_type,omitempty"`
	Default              interface{} `json:"default_value,omitempty"`
	AllowedValues        interface{} `json:"allowed_values,omitempty"`
	Minimum              interface{} `json:"minimum_value,omitempty"`
	Maximum              interface{} `json:"maximum_value,omitempty"`
	Order                string      `json:"order,omitempty"`
	LocalisedName        interface{} `json:"localised_name,omitempty"`
	LocalisedDescription interface{} `json:"localised_description,omitempty"`
	// Label and Description are the readable forms, resolved from the mod's locale
	// files where the dump gave a translation key rather than text.
	Label       string `json:"label,omitempty"`
	Description string `json:"description,omitempty"`
}

// SettingsReference is what the game itself reports about every setting the
// currently enabled mods declare. It is produced by running the game with
// --dump-data, which executes the mods' settings.lua for us, so nothing here has
// to reimplement or guess at Lua.
//
// The dump only reflects the mods that are enabled when it runs, which is what
// makes staleness detectable: the enabled set is recorded alongside it.
type SettingsReference struct {
	Mods     []string                      `json:"mods"`
	TakenAt  time.Time                     `json:"taken_at"`
	Settings map[string]SettingDeclaration `json:"settings"`
}

// SettingsReferencePath is where the reference is cached: beside the manager's
// own database, not in the game's directory.
func SettingsReferencePath() string {
	config := bootstrap.GetConfig()
	return filepath.Join(filepath.Dir(config.SQLiteDatabaseFile), "mod-settings-reference.json")
}

// modListFile is the shape of the game's mods/mod-list.json.
type modListFile struct {
	Mods []struct {
		Name    string `json:"name"`
		Enabled bool   `json:"enabled"`
	} `json:"mods"`
}

// EnabledModSet is the sorted list of mods the game would load right now. It is
// the key the reference is taken with, so a change to it marks the reference
// stale.
func EnabledModSet() []string {
	config := bootstrap.GetConfig()

	raw, err := os.ReadFile(filepath.Join(config.FactorioModsDir, "mod-list.json"))
	if err != nil {
		return nil
	}

	var list modListFile
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil
	}

	set := []string{}
	for _, mod := range list.Mods {
		if mod.Enabled {
			set = append(set, mod.Name)
		}
	}
	sort.Strings(set)

	// The installed files matter too: a mod can be updated in place with the same
	// name and a new version.
	entries, err := os.ReadDir(config.FactorioModsDir)
	if err == nil {
		files := []string{}
		for _, entry := range entries {
			if strings.HasSuffix(entry.Name(), ".zip") {
				files = append(files, "file:"+entry.Name())
			}
		}
		sort.Strings(files)
		set = append(set, files...)
	}

	return set
}

func sameModSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// LoadSettingsReference reads the cached reference. It returns nil when there is
// none, and IsCurrent on the result reports whether it still matches the enabled
// mods.
func LoadSettingsReference() *SettingsReference {
	raw, err := os.ReadFile(SettingsReferencePath())
	if err != nil {
		return nil
	}

	var reference SettingsReference
	if err := json.Unmarshal(raw, &reference); err != nil {
		log.Printf("ignoring an unreadable settings reference: %v", err)
		return nil
	}

	return &reference
}

// Current reports whether the reference describes the mods that are enabled now.
func (r *SettingsReference) Current() bool {
	return r != nil && sameModSet(r.Mods, EnabledModSet())
}

// DumpSettingsReference runs the game with --dump-data, which loads every enabled
// mod's settings.lua in the game's own Lua, and caches what it reports.
//
// The game takes an exclusive lock on its directory, so the server has to be
// stopped; the caller is responsible for that.
func DumpSettingsReference() (*SettingsReference, error) {
	config := bootstrap.GetConfig()

	command := exec.Command(config.FactorioBinary, "--dump-data")
	output, err := command.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("the game could not dump its data: %v (%s)",
			err, strings.TrimSpace(string(output)))
	}

	dumpPath := filepath.Join(config.FactorioDir, "script-output", "mod-settings-dump.json")
	raw, err := os.ReadFile(dumpPath)
	if err != nil {
		return nil, fmt.Errorf("the game produced no settings dump at %s: %v", dumpPath, err)
	}

	reference, err := parseSettingsDump(raw)
	if err != nil {
		return nil, err
	}

	// The dump usually reports a translation key rather than words, so the mods'
	// locale files are read now and the readable name is stored with the rest.
	// Doing it here keeps every page load cheap.
	// Most mods do not set localised_name at all, so the game falls back to its
	// own convention for settings: the "mod-setting-name" section of the mod's
	// locale, keyed by the setting's name. "mod-setting-description" is the same
	// for the description. Without these, a setting would show as its raw name.
	texts := LoadLocaleTexts("en")
	for name, declaration := range reference.Settings {
		label := RenderLocalised(declaration.LocalisedName, texts.Texts)
		if label == "" {
			label = texts.Texts["mod-setting-name."+name]
		}
		declaration.Label = StripRichText(label)

		description := RenderLocalised(declaration.LocalisedDescription, texts.Texts)
		if description == "" {
			description = texts.Texts["mod-setting-description."+name]
		}
		declaration.Description = StripRichText(description)

		// The locale key that named the setting lives inside the mod that wrote
		// it, so that is what identifies its owner.
		declaration.Mod = texts.Mods["mod-setting-name."+name]
		if declaration.Mod == "" {
			declaration.Mod = texts.Mods["mod-setting-description."+name]
		}

		reference.Settings[name] = declaration
	}

	reference.Mods = EnabledModSet()
	reference.TakenAt = time.Now()

	encoded, err := json.MarshalIndent(reference, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(SettingsReferencePath(), encoded, 0o644); err != nil {
		return nil, fmt.Errorf("could not cache the reference: %v", err)
	}

	log.Printf("settings reference taken: %d settings from %d enabled mods",
		len(reference.Settings), len(reference.Mods))

	return reference, nil
}

// parseSettingsDump reads the file the game writes: one object per setting type,
// each holding its settings by name.
func parseSettingsDump(raw []byte) (*SettingsReference, error) {
	var byType map[string]map[string]SettingDeclaration
	if err := json.Unmarshal(raw, &byType); err != nil {
		return nil, fmt.Errorf("could not read the settings dump: %v", err)
	}

	reference := &SettingsReference{Settings: map[string]SettingDeclaration{}}

	for _, settings := range byType {
		for name, declaration := range settings {
			// The dump repeats the name and type inside each entry, so nothing is
			// lost by keying them by name.
			reference.Settings[name] = declaration
		}
	}

	return reference, nil
}
