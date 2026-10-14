package factorio

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ApplySettingChange updates one setting in the parsed tree, checking the new
// value against the mod's declaration before accepting it.
//
// The stored value keeps the type it already had, so a write never changes a
// setting's encoding; only the value moves. That is what lets the rest of the
// file be reproduced untouched.
func ApplySettingChange(settings *ModSettings, declaration SettingDeclaration, known bool, section, name, value string) error {
	stored, found := settings.Sections.Get(section)
	if !found {
		return fmt.Errorf("the settings file has no %s section", section)
	}

	entries, ok := stored.(SettingsDict)
	if !ok {
		return fmt.Errorf("%s is not a settings section", section)
	}

	for i := range entries {
		if entries[i].Key != name {
			continue
		}

		setting, ok := entries[i].Value.(SettingsDict)
		if !ok {
			return fmt.Errorf("%s is not stored as a setting", name)
		}

		current, _ := setting.Get("value")

		next, err := coerceSettingValue(value, current, declaration, known)
		if err != nil {
			return err
		}

		setting.Set("value", next)
		entries[i].Value = setting
		settings.Sections.Set(section, entries)

		return nil
	}

	return fmt.Errorf("%s is not in the %s section", name, section)
}

// coerceSettingValue reads the text from the editor as the type the setting is
// already stored as, so the encoding cannot drift, and checks it against what
// the mod declared: its type, its allowed values and its bounds.
func coerceSettingValue(text string, current interface{}, declaration SettingDeclaration, known bool) (interface{}, error) {
	trimmed := strings.TrimSpace(text)

	switch current.(type) {
	case bool:
		value, err := strconv.ParseBool(trimmed)
		if err != nil {
			return nil, fmt.Errorf("expected true or false, got %q", text)
		}
		if known && declaration.Type != "" && declaration.Type != "bool-setting" {
			return nil, fmt.Errorf("the mod declares this as %s", declaration.Type)
		}
		return value, nil

	case int64:
		value, err := strconv.ParseInt(trimmed, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("expected a whole number, got %q", text)
		}
		if known {
			if err := checkBounds(float64(value), declaration); err != nil {
				return nil, err
			}
		}
		return value, nil

	case float64:
		value, err := strconv.ParseFloat(trimmed, 64)
		if err != nil {
			return nil, fmt.Errorf("expected a number, got %q", text)
		}
		if known {
			if err := checkBounds(value, declaration); err != nil {
				return nil, err
			}
		}
		return value, nil

	case string:
		// An undeclared setting is written at the caller's risk, so its values
		// are not checked against a declaration we do not have.
		if known {
			if allowed, ok := allowedValues(declaration); ok && len(allowed) > 0 {
				for _, candidate := range allowed {
					if candidate == trimmed {
						return trimmed, nil
					}
				}
				return nil, fmt.Errorf("%q is not one of %s", text, strings.Join(allowed, ", "))
			}
		}
		return trimmed, nil
	}

	return nil, fmt.Errorf("cannot store a value for this setting")
}

func allowedValues(declaration SettingDeclaration) ([]string, bool) {
	list, ok := declaration.AllowedValues.([]interface{})
	if !ok {
		return nil, false
	}

	values := make([]string, 0, len(list))
	for _, item := range list {
		values = append(values, fmt.Sprintf("%v", item))
	}

	return values, true
}

func checkBounds(value float64, declaration SettingDeclaration) error {
	if minimum, ok := declaration.Minimum.(int64); ok && value < float64(minimum) {
		return fmt.Errorf("the mod allows no less than %d", minimum)
	}
	if minimum, ok := declaration.Minimum.(float64); ok && value < minimum {
		return fmt.Errorf("the mod allows no less than %v", minimum)
	}
	if maximum, ok := declaration.Maximum.(int64); ok && value > float64(maximum) {
		return fmt.Errorf("the mod allows no more than %d", maximum)
	}
	if maximum, ok := declaration.Maximum.(float64); ok && value > maximum {
		return fmt.Errorf("the mod allows no more than %v", maximum)
	}

	return nil
}

// WriteModSettings keeps a copy of the current file and then replaces it. The
// copy is what makes a bad write recoverable.
func WriteModSettings(path string, encoded []byte) (backup string, err error) {
	if current, readErr := os.ReadFile(path); readErr == nil {
		backup = path + ".bak"
		if writeErr := os.WriteFile(backup, current, 0o644); writeErr != nil {
			return "", fmt.Errorf("could not write the backup %s: %v", backup, writeErr)
		}
	}

	if err := os.WriteFile(path, encoded, 0o644); err != nil {
		return backup, fmt.Errorf("could not write %s: %v", path, err)
	}

	return backup, nil
}
