package companion

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	neturl "net/url"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/OpenFactorioServerManager/factorio-server-manager/bootstrap"
	"github.com/OpenFactorioServerManager/factorio-server-manager/factorio"
)

const (
	// ModName is the Factorio mod name of the companion.
	ModName = "fsm-companion"

	// CompanionVersion is the only companion version the manager installs.
	//
	// This const is the compatibility boundary between the two halves: the
	// manager's report parser (factorio/companion_report.go) reads the shape
	// this version produces. Releasing a new companion therefore means editing
	// this line and building a new manager, which is deliberate — the mod may
	// move on its own cadence, but never further than a manager release allows.
	CompanionVersion = "0.3.11"

	// githubOwner and githubRepo identify the public repository whose releases
	// carry the mod. It is public, so fetching needs no credential.
	githubOwner = "NovaProtocol"
	githubRepo  = "factorio-companion-mod"

	// fetchTimeout bounds one remote download. The zip is a few kilobytes.
	fetchTimeout = 30 * time.Second

	// maxZipBytes caps what will be read from a remote, so a wrong URL that
	// streams something endless cannot fill the disk.
	maxZipBytes = 8 << 20
)

// Status is what the manager knows about the companion mod.
type Status struct {
	ModName   string `json:"modName"`
	Pinned    string `json:"pinned"`
	Installed bool   `json:"installed"`
	Version   string `json:"version"`
	Source    string `json:"source"`
	Error     string `json:"error"`
}

// The outcome of the last Install is kept in memory so the Mods page can say
// where the mod came from. It is not persisted: after a restart the answer is
// honestly just "installed".
var (
	lastMu     sync.Mutex
	lastStatus Status
)

func record(status Status) Status {
	lastMu.Lock()
	lastStatus = status
	lastMu.Unlock()

	return status
}

// CurrentStatus reports the state without touching the network, so the Mods page
// can render it on every load.
func CurrentStatus() Status {
	lastMu.Lock()
	status := lastStatus
	lastMu.Unlock()

	if status.ModName == "" {
		status = Status{ModName: ModName, Pinned: CompanionVersion}
	}

	// Re-check the disk, so a zip removed by hand is reported honestly rather
	// than echoed from memory.
	if _, err := os.Stat(installedZipPath()); err == nil {
		status.Installed = true
		status.Version = CompanionVersion
		status.Error = ""
	} else {
		status.Installed = false
		status.Version = ""
		status.Source = ""
	}

	return status
}

// EnsureInstalled installs the pinned companion version at startup, logging the
// outcome. A failure is not fatal: the manager is still useful without the
// companion, and the Mods page reports what is missing and why.
func EnsureInstalled() {
	status, err := Install()
	if err != nil {
		log.Printf("companion: %s %s is not installed: %s", ModName, CompanionVersion, err)
		return
	}

	log.Printf("companion: %s %s is installed and enabled (source: %s)", ModName, status.Version, status.Source)
}

// Install makes sure exactly CompanionVersion is present in the server's mods
// directory. It is a no-op when that version is already there, so an ordinary
// restart never touches the network. Otherwise it fetches the zip from the
// GitHub release and, when that fails and a mod portal credential is stored,
// from the mod portal.
func Install() (Status, error) {
	status := Status{ModName: ModName, Pinned: CompanionVersion}

	config := bootstrap.GetConfig()
	if err := os.MkdirAll(config.FactorioModsDir, 0o755); err != nil {
		return record(withError(status, err)), err
	}

	dest := filepath.Join(config.FactorioModsDir, zipName())

	_, statErr := os.Stat(dest)
	switch {
	case statErr == nil:
		status.Installed = true
		status.Version = CompanionVersion
		status.Source = "installed"
	case !os.IsNotExist(statErr):
		return record(withError(status, statErr)), statErr
	default:
		data, source, err := fetch()
		if err != nil {
			return record(withError(status, err)), err
		}

		if err := writeZip(dest, data); err != nil {
			return record(withError(status, err)), err
		}

		status.Installed = true
		status.Version = CompanionVersion
		status.Source = source
	}

	if err := removeOtherVersions(config.FactorioModsDir); err != nil {
		// Not fatal: the pinned version is in place and enabled.
		log.Printf("companion: could not clear older versions: %s", err)
	}

	if err := enableMod(config.FactorioModsDir); err != nil {
		return record(withError(status, err)), err
	}

	return record(status), nil
}

func withError(status Status, err error) Status {
	status.Error = err.Error()
	return status
}

func zipName() string {
	return ModName + "_" + CompanionVersion + ".zip"
}

func installedZipPath() string {
	return filepath.Join(bootstrap.GetConfig().FactorioModsDir, zipName())
}

// fetch returns the pinned version's zip, preferring the GitHub release because
// it needs no credential.
func fetch() ([]byte, string, error) {
	data, githubErr := fetchFromGitHub()
	if githubErr == nil {
		return data, "github", nil
	}

	log.Printf("companion: GitHub release v%s unusable (%s); trying the mod portal", CompanionVersion, githubErr)

	data, portalErr := fetchFromPortal()
	if portalErr == nil {
		return data, "portal", nil
	}

	return nil, "", fmt.Errorf("GitHub release: %v; mod portal: %v", githubErr, portalErr)
}

func fetchFromGitHub() ([]byte, error) {
	url := fmt.Sprintf("https://github.com/%s/%s/releases/download/v%s/%s",
		githubOwner, githubRepo, CompanionVersion, zipName())

	return download(url)
}

// fetchFromPortal is the fallback. It needs the Factorio credentials the
// Credentials page stores: the mod portal refuses anonymous downloads, and the
// portal API key does not authorise one either.
func fetchFromPortal() ([]byte, error) {
	var credentials factorio.Credentials
	loggedIn, err := credentials.Load()
	if err != nil {
		return nil, fmt.Errorf("reading the stored mod portal credentials: %w", err)
	}
	if !loggedIn {
		return nil, errors.New("no mod portal credential is stored")
	}

	details, err, statusCode := factorio.ModPortalModDetails(ModName)
	if err != nil {
		return nil, fmt.Errorf("reading %s from the mod portal (HTTP %d): %w", ModName, statusCode, err)
	}

	wanted := factorio.Version{}
	if err := wanted.UnmarshalText([]byte(CompanionVersion)); err != nil {
		return nil, fmt.Errorf("the pinned version %q is not a version: %w", CompanionVersion, err)
	}

	for _, release := range details.Releases {
		if !release.Version.Equals(wanted) {
			continue
		}

		url := "https://mods.factorio.com" + release.DownloadURL +
			"?username=" + neturl.QueryEscape(credentials.Username) +
			"&token=" + neturl.QueryEscape(credentials.Userkey)

		return download(url)
	}

	return nil, fmt.Errorf("the mod portal has no release %s", CompanionVersion)
}

// download fetches a zip and refuses anything that is not the pinned mod, so a
// redirect to an HTML page can never be installed.
func download(url string) ([]byte, error) {
	client := &http.Client{Timeout: fetchTimeout}

	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s answered HTTP %d", url, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxZipBytes+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxZipBytes {
		return nil, fmt.Errorf("%s answered more than %d bytes", url, maxZipBytes)
	}

	if err := validateModZip(body); err != nil {
		return nil, fmt.Errorf("%s: %w", url, err)
	}

	return body, nil
}

// validateModZip checks the archive is the pinned mod, not merely a zip.
func validateModZip(data []byte) error {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return errors.New("the download is not a zip archive")
	}

	want := ModName + "_" + CompanionVersion + "/info.json"

	for _, file := range reader.File {
		if file.Name != want {
			continue
		}

		raw, err := readZipFile(file)
		if err != nil {
			return err
		}

		var info struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		}
		if err := json.Unmarshal(raw, &info); err != nil {
			return fmt.Errorf("%s is not readable: %w", want, err)
		}
		if info.Name != ModName {
			return fmt.Errorf("the download is %q, not %q", info.Name, ModName)
		}
		if info.Version != CompanionVersion {
			return fmt.Errorf("the download is version %q, not the pinned %q", info.Version, CompanionVersion)
		}

		return nil
	}

	return fmt.Errorf("the download has no %s", want)
}

func readZipFile(file *zip.File) ([]byte, error) {
	handle, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer handle.Close()

	return io.ReadAll(io.LimitReader(handle, maxZipBytes))
}

// writeZip lands the zip atomically, so a half-written file is never left where
// Factorio would load it.
func writeZip(dest string, data []byte) error {
	temp, err := os.CreateTemp(filepath.Dir(dest), ModName+"-*.zip")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())

	if _, err := temp.Write(data); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}

	// CreateTemp makes the file 0600; the mods directory is read by the game
	// server, so give the zip the same permissions a plain create would.
	if err := os.Chmod(temp.Name(), 0o644); err != nil {
		return err
	}

	return os.Rename(temp.Name(), dest)
}

// removeOtherVersions drops every other companion version — both other zips and
// the unpacked directory older manager builds installed, since two copies of one
// mod confuse Factorio.
func removeOtherVersions(modsDir string) error {
	keep := zipName()

	zips, err := filepath.Glob(filepath.Join(modsDir, ModName+"_*.zip"))
	if err != nil {
		return err
	}

	for _, zip := range zips {
		if filepath.Base(zip) == keep {
			continue
		}

		if err := os.Remove(zip); err != nil {
			return err
		}
		log.Printf("companion: removed the other version %s", filepath.Base(zip))
	}

	dirs, err := filepath.Glob(filepath.Join(modsDir, ModName+"_*"))
	if err != nil {
		return err
	}

	for _, dir := range dirs {
		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			continue
		}

		if err := os.RemoveAll(dir); err != nil {
			return err
		}
		log.Printf("companion: removed the unpacked copy %s", filepath.Base(dir))
	}

	return nil
}

type modListEntry struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

type modList struct {
	Mods []modListEntry `json:"mods"`
}

// enableMod adds the companion to mod-list.json enabled.
//
// A disable is deliberately left alone: an entry that already exists keeps
// whatever state it has, so switching the companion off in the mod list (or
// through the manager's own toggle) survives a restart.
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
