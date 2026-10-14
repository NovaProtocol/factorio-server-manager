package factorio

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// The whole editor rests on this. Encoding what was just parsed has to
// reproduce the original file byte for byte; if it does not, nothing may write.
func TestEncodeModSettingsRoundTrip(t *testing.T) {
	fixtures, err := filepath.Glob("../factorio_mod_settings_testfiles/*.dat")
	if err != nil {
		t.Fatalf("globbing fixtures: %v", err)
	}
	if len(fixtures) == 0 {
		t.Fatal("no mod-settings fixtures found")
	}

	for _, path := range fixtures {
		t.Run(filepath.Base(path), func(t *testing.T) {
			original, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("reading %s: %v", path, err)
			}

			settings, err := LoadModSettingsFile(path)
			if err != nil {
				t.Fatalf("parsing %s: %v", path, err)
			}
			if settings == nil {
				t.Fatalf("no settings parsed from %s", path)
			}

			encoded, err := EncodeModSettings(settings)
			if err != nil {
				t.Fatalf("encoding %s: %v", path, err)
			}

			if bytes.Equal(original, encoded) {
				return
			}

			t.Errorf("%s: re-encoded output differs from the original (%d vs %d bytes)",
				filepath.Base(path), len(original), len(encoded))

			for i := 0; i < len(original) && i < len(encoded); i++ {
				if original[i] != encoded[i] {
					from := i - 8
					if from < 0 {
						from = 0
					}
					t.Errorf("  first difference at byte %d\n    original: % x\n    encoded : % x",
						i, original[from:i+8], encoded[from:i+8])
					break
				}
			}
		})
	}
}

// The fixtures pair a real mod-settings.dat with the settings it holds, so the
// reader is checked against something the game actually wrote rather than
// against itself.
func TestLoadModSettings(t *testing.T) {
	fixtures, err := filepath.Glob("../factorio_mod_settings_testfiles/*.dat")
	if err != nil {
		t.Fatalf("globbing fixtures: %v", err)
	}
	if len(fixtures) == 0 {
		t.Fatal("no mod-settings fixtures found")
	}

	for _, path := range fixtures {
		t.Run(filepath.Base(path), func(t *testing.T) {
			// Some fixtures exist only for the round-trip test, with no expected
			// JSON to compare against.
			expected := strings.TrimSuffix(path, ".dat") + ".json"
			raw, err := os.ReadFile(expected)
			if err != nil {
				t.Skipf("no expected settings alongside %s", filepath.Base(path))
			}

			var want map[string]interface{}
			if err := json.Unmarshal(raw, &want); err != nil {
				t.Fatalf("parsing %s: %v", expected, err)
			}

			settings, err := LoadModSettingsFile(path)
			if err != nil {
				t.Fatalf("LoadModSettingsFile(%s): %v", path, err)
			}
			if settings == nil {
				t.Fatalf("no settings parsed from %s", path)
			}

			got := settings.Sections

			if !reflect.DeepEqual(plain(got), want) {
				t.Errorf("parsed settings do not match %s", filepath.Base(expected))

				for _, section := range []string{"startup", "runtime-global", "runtime-per-user"} {
					wantSection, _ := want[section].(map[string]interface{})
					raw, _ := got.Get(section)
					gotSection, _ := raw.(SettingsDict)
					if len(gotSection) != len(wantSection) {
						t.Errorf("  section %q: parsed %d entries, expected %d",
							section, len(gotSection), len(wantSection))
					}
				}
			}
		})
	}
}

// plain turns the ordered tree into the shape encoding/json produces, so the
// parsed result can be compared with the fixture's expected JSON.
func plain(value interface{}) interface{} {
	switch typed := value.(type) {
	case SettingsDict:
		out := map[string]interface{}{}
		for _, entry := range typed {
			out[entry.Key] = plain(entry.Value)
		}
		return out

	case []interface{}:
		out := make([]interface{}, 0, len(typed))
		for _, item := range typed {
			out = append(out, plain(item))
		}
		return out

	default:
		return value
	}
}

// A missing file is a normal state: the game has not written one yet.
func TestLoadModSettingsMissingFile(t *testing.T) {
	got, err := LoadModSettingsFile(filepath.Join(t.TempDir(), "mod-settings.dat"))
	if err != nil {
		t.Fatalf("a missing file should not be an error, got: %v", err)
	}
	if got != nil {
		t.Errorf("expected no settings, got %v", got)
	}
}
