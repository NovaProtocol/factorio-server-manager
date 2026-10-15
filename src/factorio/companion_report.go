package factorio

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ScopedReport is one companion report scope, cached briefly.
//
// Reports are fetched per scope rather than as one document because a body has to
// survive RCON, which caps a frame near 4096 bytes: a page only pays for the slice
// it is actually showing.
type ScopedReport struct {
	Data      map[string]interface{}
	Scope     string
	FetchedAt time.Time
	Error     string
}

const (
	companionCommand      = "/companion-report"
	companionChunkCommand = "/companion-chunk"

	// companionCacheTTL keeps a page on one RCON round per cadence. The browser
	// polls every 5s and the cache expires at the same rate, so a page sees a
	// fresh number each poll without more than one fetch per scope per 5s.
	companionCacheTTL = 5 * time.Second

	// companionChunksMax refuses a manifest that would make the manager read for
	// ever; the largest scope measures about five chunks.
	companionChunksMax = 64

	// companionAttempts covers the case where a competing caller rebuilds the
	// cached report between the manifest and the chunks.
	companionAttempts = 3
)

var (
	// rconMutex serialises the write/read pair. RCON carries no notion of who a
	// response belongs to beyond the request id, so two overlapping callers would
	// read each other's answers.
	rconMutex sync.Mutex

	// fetchMutex serialises whole reports, so one slow scope cannot be interleaved
	// with another.
	fetchMutex sync.Mutex

	cacheMutex sync.Mutex
	cache      = map[string]*ScopedReport{}
)

// errStaleReport means another caller replaced the mod's cached report while this
// one was reading it, so the chunks on offer belong to a different report.
var errStaleReport = errors.New("companion report was replaced mid-fetch")

// GetScopedReport returns a scope, fetching it when the entry is missing or older
// than the TTL. The cache is keyed by the scope and its arguments, so each page
// and window is fetched at most once per TTL no matter how many viewers are on it.
//
// Fetches never queue. If another report is being fetched right now, the caller
// is answered immediately with what the cache holds — fresh if it happens to be,
// stale otherwise — and the next poll picks up whatever the in-flight fetch
// produced. A browser poll is a request for new data "only when the manager can",
// never a wait in a behind-the-fetch queue.
func GetScopedReport(scope string, args ...string) *ScopedReport {
	// An empty argument has to keep its position: the mod splits the command into
	// words, so a gap would shift every argument after it — which once put the
	// time window where the series belongs.
	positional := make([]string, len(args))
	for index, arg := range args {
		if strings.TrimSpace(arg) == "" {
			positional[index] = "-"
		} else {
			positional[index] = arg
		}
	}

	key := strings.Join(append([]string{scope}, positional...), " ")

	if entry := cachedReport(key); entry != nil {
		return entry
	}

	if !fetchMutex.TryLock() {
		// A fetch is already running. Serve the scope's last report, however old
		// it is, and let the running fetch finish rather than queueing behind it.
		return reportAnyAge(key)
	}
	defer fetchMutex.Unlock()

	// Another caller may have fetched while this one reached the lock.
	if entry := cachedReport(key); entry != nil {
		return entry
	}

	entry := fetchScopedReport(key)

	cacheMutex.Lock()
	cache[key] = entry
	cacheMutex.Unlock()

	return entry
}

// reportAnyAge returns the cached scope whatever its age, or a friendly empty
// entry when the scope has never been fetched and cannot be fetched right now.
func reportAnyAge(key string) *ScopedReport {
	cacheMutex.Lock()
	entry, ok := cache[key]
	cacheMutex.Unlock()

	if ok && entry != nil {
		return entry
	}

	return &ScopedReport{
		Scope:     key,
		FetchedAt: time.Now(),
		Error:     "a fetch of another report is in progress; this scope has never been fetched",
	}
}

func cachedReport(key string) *ScopedReport {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	entry, ok := cache[key]
	if !ok || time.Since(entry.FetchedAt) >= companionCacheTTL {
		return nil
	}

	return entry
}

// CompanionSpeed bounds. Speed and max UPS are one equation apart
// (max_ups = speed x 60), so the panel can offer either field while the wire
// format carries only the speed.
const (
	companionMinSpeed = 0.01
	companionMaxSpeed = 3.0
)

// SetCompanionSpeed applies a game speed through the companion mod, which keeps
// it across save loads and clamps nothing — the value is validated here so an
// unusable number never reaches the game. It waits for rather than bypasses a
// running fetch: this is an explicit user action, not a poll.
func SetCompanionSpeed(speed float64) (map[string]interface{}, error) {
	if speed < companionMinSpeed || speed > companionMaxSpeed {
		return nil, fmt.Errorf("speed must be between %.2f (%.1f max UPS) and %.2f (%.1f max UPS)",
			companionMinSpeed, companionMinSpeed*60, companionMaxSpeed, companionMaxSpeed*60)
	}

	fetchMutex.Lock()
	defer fetchMutex.Unlock()

	// Full precision on the wire: rounding here is what turned "1 max UPS" into
	// speed 0.016667 and back into 1.00002 UPS. The mod parses whatever it gets.
	response, err := runCompanionCommand("/companion-speed " + strconv.FormatFloat(speed, 'f', -1, 64))
	if err != nil {
		return nil, err
	}

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(response)), &payload); err != nil {
		return nil, err
	}

	if message, ok := payload["error"].(string); ok && message != "" {
		return nil, errors.New(message)
	}

	return payload, nil
}

func fetchScopedReport(key string) *ScopedReport {
	entry := &ScopedReport{Scope: key, FetchedAt: time.Now()}

	server := GetFactorioServer()
	if !server.GetRunning() {
		entry.Error = "server is not running"
		return entry
	}

	data, err := fetchScopedData(key)
	if err != nil {
		entry.Error = describeCompanionError(err)
		log.Printf("Companion report %q failed: %s", key, err)
		return entry
	}

	entry.Data = data

	return entry
}

// describeCompanionError turns the transport's own words into something the page
// can act on. A closed connection is what a stopped game server looks like from
// here, and "EOF" on its own says nothing to anyone reading the dashboard.
func describeCompanionError(err error) string {
	if errors.Is(err, io.EOF) {
		return "lost the RCON connection: the game server has probably stopped"
	}

	return err.Error()
}

func fetchScopedData(key string) (map[string]interface{}, error) {
	var stale error

	for attempt := 0; attempt < companionAttempts; attempt++ {
		manifest, err := runCompanionCommand(companionCommand + " " + key)
		if err != nil {
			return nil, err
		}

		data, err := collectScopedBody(manifest)
		if errors.Is(err, errStaleReport) {
			stale = err
			continue
		}
		if err != nil {
			return nil, err
		}

		return data, nil
	}

	return nil, fmt.Errorf("%w (gave up after %d attempts)", stale, companionAttempts)
}

// collectScopedBody reads a report back in chunks. The manifest carries the token
// the mod cached that report under, and every chunk must be requested with it: a
// clobbered fetch then fails here rather than assembling someone else's body.
func collectScopedBody(manifest string) (map[string]interface{}, error) {
	var info struct {
		Token  int    `json:"token"`
		Length int    `json:"length"`
		Chunks int    `json:"chunks"`
		Error  string `json:"error"`
	}

	if err := json.Unmarshal([]byte(manifest), &info); err != nil {
		return nil, errors.New("unexpected manifest from companion mod: " + manifest)
	}
	if info.Error != "" {
		return nil, errors.New("companion mod reported: " + info.Error)
	}
	if info.Chunks <= 0 || info.Chunks > companionChunksMax {
		return nil, fmt.Errorf("companion mod returned an unusable chunk count (%s)", manifest)
	}

	var body strings.Builder
	for index := 0; index < info.Chunks; index++ {
		part, err := runCompanionCommand(fmt.Sprintf("%s %d %d", companionChunkCommand, info.Token, index))
		if err != nil {
			return nil, err
		}
		if strings.HasPrefix(part, "stale report") {
			return nil, errStaleReport
		}
		body.WriteString(part)
	}

	assembled := body.String()
	if len(assembled) != info.Length {
		return nil, fmt.Errorf("report is incomplete: expected %d bytes, assembled %d", info.Length, len(assembled))
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(assembled), &data); err != nil {
		return nil, errors.New("could not parse companion response: " + err.Error())
	}

	return data, nil
}

// runCompanionCommand writes a command and returns the response carrying its
// request id. Responses are read one frame at a time: the transport caps a frame
// at 4096 bytes, and callers chunk anything larger.
func runCompanionCommand(command string) (string, error) {
	server := GetFactorioServer()
	if server.Rcon == nil {
		return "", errors.New("no rcon session")
	}

	rconMutex.Lock()
	defer rconMutex.Unlock()

	requestID, err := server.Rcon.Write(command)
	if err != nil {
		return "", err
	}

	// A response left over from an interrupted call can arrive first, so a few
	// mismatched frames are skipped before giving up.
	for attempts := 0; attempts < 8; attempts++ {
		response, id, err := server.Rcon.Read()
		if err != nil {
			return "", err
		}
		if id == requestID {
			return strings.TrimSpace(response), nil
		}
	}

	return "", errors.New("no response for request " + strings.TrimSpace(command))
}
