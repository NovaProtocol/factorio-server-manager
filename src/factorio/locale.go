package factorio

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/OpenFactorioServerManager/factorio-server-manager/bootstrap"
)

func readZipMember(file *zip.File) ([]byte, error) {
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return io.ReadAll(reader)
}

// Locale files hold the words a mod uses for a setting. The settings dump gives
// each setting a localised_name, but that is usually a translation key rather
// than text, so the mod's own locale/<language>/*.cfg has to be read to render
// it. The format is a section header followed by key=value lines.

// LocaleTexts holds the strings a mod gives for its settings, and which mod each
// one came from. The owning mod is known because a locale file lives inside the
// mod that wrote it, which is what lets a setting be attributed at all: the
// settings dump does not say who declared what.
type LocaleTexts struct {
	Texts map[string]string
	Mods  map[string]string
}

// LoadLocaleTexts reads the locale files of every installed mod, plus the game's
// own, and returns them keyed the way a localised_name refers to them, which is
// "section.key".
func LoadLocaleTexts(language string) LocaleTexts {
	config := bootstrap.GetConfig()
	texts := LocaleTexts{Texts: map[string]string{}, Mods: map[string]string{}}

	entries, err := os.ReadDir(config.FactorioModsDir)
	if err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".zip") {
				readLocaleFromArchive(filepath.Join(config.FactorioModsDir, entry.Name()), language, texts)
			}
		}
	}

	// The base game's own locale, in case a mod refers to a base key.
	baseDirs, err := filepath.Glob(filepath.Join(config.FactorioDir, "data", "*", "locale", language))
	if err == nil {
		for _, dir := range baseDirs {
			files, err := filepath.Glob(filepath.Join(dir, "*.cfg"))
			if err != nil {
				continue
			}
			for _, file := range files {
				raw, err := os.ReadFile(file)
				if err == nil {
					parseLocaleFile(string(raw), texts, "")
				}
			}
		}
	}

	return texts
}

func readLocaleFromArchive(path, language string, into LocaleTexts) {
	archive, err := zip.OpenReader(path)
	if err != nil {
		return
	}
	defer archive.Close()

	// A mod archive is named <mod>_<version>.zip, and the version is not part of
	// the name anyone recognises.
	mod := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	if cut := strings.LastIndex(mod, "_"); cut > 0 && isVersionSuffix(mod[cut+1:]) {
		mod = mod[:cut]
	}

	suffix := "/locale/" + language + "/"
	for _, file := range archive.File {
		name := filepath.ToSlash(file.Name)
		if !strings.Contains(name, suffix) || !strings.HasSuffix(name, ".cfg") {
			continue
		}

		raw, err := readZipMember(file)
		if err != nil {
			continue
		}
		parseLocaleFile(string(raw), into, mod)
	}
}

// parseLocaleFile reads "[section]" headers and "key=value" lines. A UTF-8 BOM
// is stripped, since Factorio writes one.
func parseLocaleFile(source string, into LocaleTexts, mod string) {
	source = strings.TrimPrefix(source, "\ufeff")
	section := ""

	for _, line := range strings.Split(source, "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = line[1 : len(line)-1]
			continue
		}

		equals := strings.Index(line, "=")
		if equals < 0 {
			continue
		}

		key := strings.TrimSpace(line[:equals])
		if key == "" {
			continue
		}
		if section != "" {
			key = section + "." + key
		}

		into.Texts[key] = strings.TrimSpace(line[equals+1:])
		if mod != "" {
			if _, taken := into.Mods[key]; !taken {
				into.Mods[key] = mod
			}
		}
	}
}

func isVersionSuffix(text string) bool {
	if text == "" {
		return false
	}
	for _, r := range text {
		if (r < '0' || r > '9') && r != '.' {
			return false
		}
	}
	return true
}

// StripRichText removes Factorio's inline markup, which a description may use,
// for example "[color=red][!][/color] Biter modifier.". Left alone it would show
// up as literal tags in a table.
func StripRichText(text string) string {
	var out strings.Builder

	for i := 0; i < len(text); {
		if text[i] != '[' {
			out.WriteByte(text[i])
			i++
			continue
		}

		end := strings.IndexByte(text[i:], ']')
		if end < 0 || strings.ContainsAny(text[i:i+end], "\n") {
			out.WriteByte(text[i])
			i++
			continue
		}

		i += end + 1
	}

	return strings.TrimSpace(out.String())
}

// RenderLocalised turns what the dump reports as a localised_name into text. It
// accepts a plain string, or the list form Factorio uses, where the first
// element is the key and the rest are parameters substituted into __1__, __2__
// and so on. It returns "" when the text is not known, so a caller can fall back
// to the raw name.
func RenderLocalised(value interface{}, texts map[string]string) string {
	switch typed := value.(type) {
	case string:
		if typed == "" {
			return ""
		}
		if text, ok := texts[typed]; ok {
			return text
		}
		// A literal, already readable string from the mod.
		if strings.ContainsAny(typed, " .") {
			return typed
		}
		return ""

	case []interface{}:
		if len(typed) == 0 {
			return ""
		}

		key, ok := typed[0].(string)
		if !ok {
			return ""
		}

		text, ok := texts[key]
		if !ok {
			return ""
		}

		for i, parameter := range typed[1:] {
			rendered := RenderLocalised(parameter, texts)
			if rendered == "" {
				rendered = fmt.Sprintf("%v", parameter)
			}
			text = strings.ReplaceAll(text, fmt.Sprintf("__%d__", i+1), rendered)
		}

		return text
	}

	return ""
}
