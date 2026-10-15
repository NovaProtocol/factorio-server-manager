package factorio

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/OpenFactorioServerManager/factorio-server-manager/api/websocket"
	"github.com/OpenFactorioServerManager/factorio-server-manager/bootstrap"
	"github.com/OpenFactorioServerManager/rcon"
)

type Server struct {
	Cmd            *exec.Cmd              `json:"-"`
	Savefile       string                 `json:"savefile"`
	Latency        int                    `json:"latency"`
	BindIP         string                 `json:"bindip"`
	Port           int                    `json:"port"`
	Running        bool                   `json:"running"`
	Ups            float64                `json:"ups"`
	GameSpeed      float64                `json:"game_speed"`
	Version        Version                `json:"fac_version"`
	BaseModVersion string                 `json:"base_mod_version"`
	StdOut         io.ReadCloser          `json:"-"`
	StdErr         io.ReadCloser          `json:"-"`
	StdIn          io.WriteCloser         `json:"-"`
	Settings       map[string]interface{} `json:"-"`
	Rcon           *rcon.RemoteConsole    `json:"-"`
	LogChan        chan []string          `json:"-"`
}

var instantiated Server
var once sync.Once
var upsSamplerOnce sync.Once

const upsSampleInterval = 5 * time.Second

// startUpsSampler watches the game's tick against wall-clock time, so the status
// pill can report the update rate the server is actually achieving. It is one
// small RCON command every 5s while a game runs; with no game the sampler does
// nothing beyond the boolean check. Reads run under rconMutex like every other
// RCON caller, so they never interleave with dashboard fetches.
func startUpsSampler() {
	upsSamplerOnce.Do(func() {
		go func() {
			var lastTick uint64
			var lastAt time.Time
			for {
				time.Sleep(upsSampleInterval)

				if !instantiated.GetRunning() || instantiated.Rcon == nil {
					lastAt = time.Time{}
					if instantiated.Ups != 0 {
						instantiated.Ups = 0
						broadcastServerStatus(&instantiated)
					}
					continue
				}

				tick, speed, ok := readGameStatus()
				if !ok {
					lastAt = time.Time{}
					if instantiated.Ups != 0 {
						instantiated.Ups = 0
						broadcastServerStatus(&instantiated)
					}
					continue
				}

				// The speed rides with the tick sample, so the speed panel's
				// live figure is never a separate query.
				instantiated.GameSpeed = speed

				now := time.Now()
				// A restart resets the tick, which would measure a negative or
				// enormous rate; those samples restart the series instead.
				fresh := lastAt.IsZero() || tick < lastTick
				if !fresh {
					seconds := now.Sub(lastAt).Seconds()
					if seconds > 0.5 {
						instantiated.Ups = math.Round(float64(tick-lastTick)/seconds*10) / 10
						broadcastServerStatus(&instantiated)
					}
				}
				lastTick = tick
				lastAt = now
			}
		}()
	})
}

// readGameTick asks the companion mod for the game tick and its current speed:
// game.speed travels with the tick, since the status pill and the speed panel
// both want it. It is the `/companion-tick` command, a plain mod command and not
// a script command — the only game calls that never stop achievements.
func readGameStatus() (tick uint64, speed float64, ok bool) {
	response, err := runCompanionCommand("/companion-tick")
	if err != nil {
		return 0, 0, false
	}

	var payload struct {
		Tick  uint64  `json:"tick"`
		Speed float64 `json:"speed"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(response)), &payload); err != nil {
		return 0, 0, false
	}

	return payload.Tick, payload.Speed, true
}

func broadcastServerStatus(server *Server) {
	wsRoom := websocket.WebsocketHub.GetRoom("server_status")
	response, _ := json.Marshal(server)
	wsRoom.Send(string(response))
}

func (server *Server) SetRunning(newState bool) {
	if server.Running != newState {
		log.Println("new state, will also send to correct room")
		server.Running = newState
		wsRoom := websocket.WebsocketHub.GetRoom("server_status")
		response, _ := json.Marshal(server)
		wsRoom.Send(string(response))
	}
}

func (server *Server) GetRunning() bool {
	return server.Running
}

func (server *Server) autostart() {
	var err error
	if server.BindIP == "" {
		server.BindIP = "0.0.0.0"

	}
	if server.Port == 0 {
		server.Port = 34197
	}
	server.Savefile = "Load Latest"

	err = server.Run()

	if err != nil {
		log.Printf("Error starting Factorio server: %+v", err)
		return
	}

}

func SetFactorioServer(server Server) {
	instantiated = server
}

func NewFactorioServer() (err error) {
	server := Server{}
	server.Settings = make(map[string]interface{})
	config := bootstrap.GetConfig()
	if err = os.MkdirAll(config.FactorioConfigDir, 0755); err != nil {
		log.Printf("failed to create config directory: %v", err)
		return
	}

	settingsPath := config.SettingsFile
	var settings *os.File

	if _, err = os.Stat(settingsPath); os.IsNotExist(err) {
		// copy example settings to supplied settings file, if not exists
		log.Printf("Server settings at %s not found, copying example server settings.\n", settingsPath)

		examplePath := filepath.Join(config.FactorioDir, "data", "server-settings.example.json")

		var example *os.File
		example, err = os.Open(examplePath)
		if err != nil {
			log.Printf("failed to open example server settings: %v", err)
			return
		}
		defer example.Close()

		settings, err = os.Create(settingsPath)
		if err != nil {
			log.Printf("failed to create server settings file: %v", err)
			return
		}
		defer settings.Close()

		_, err = io.Copy(settings, example)
		if err != nil {
			log.Printf("failed to copy example server settings: %v", err)
			return
		}

		err = example.Close()
		if err != nil {
			log.Printf("failed to close example server settings: %s", err)
			return
		}
	} else {
		// otherwise, open file normally
		settings, err = os.Open(settingsPath)
		if err != nil {
			log.Printf("failed to open server settings file: %v", err)
			return
		}
		defer settings.Close()
	}

	// before reading reset offset
	if _, err = settings.Seek(0, 0); err != nil {
		log.Printf("error while seeking in settings file: %v", err)
		return
	}

	if err = json.NewDecoder(settings).Decode(&server.Settings); err != nil {
		log.Printf("error reading %s: %v", settingsPath, err)
		return
	}

	log.Printf("Loaded Factorio settings from %s\n", settingsPath)

	out := []byte{}
	//Load factorio version
	if config.GlibcCustom == "true" {
		out, err = exec.Command(config.GlibcLocation, "--library-path", config.GlibcLibLoc, config.FactorioBinary, "--version").Output()
	} else {
		out, err = exec.Command(config.FactorioBinary, "--version").Output()
	}

	if err != nil {
		log.Printf("error on loading factorio version: %s", err)
		return
	}

	reg := regexp.MustCompile("Version.*?((\\d+\\.)?(\\d+\\.)?(\\*|\\d+)+)")
	found := reg.FindStringSubmatch(string(out))
	err = server.Version.UnmarshalText([]byte(found[1]))
	if err != nil {
		log.Printf("could not parse version: %v", err)
		return
	}

	//Load baseMod version
	baseModInfoFile := filepath.Join(config.FactorioBaseModDir, "info.json")
	bmifBa, err := ioutil.ReadFile(baseModInfoFile)
	if err != nil {
		log.Printf("couldn't open baseMods info.json: %s", err)
		return
	}
	var modInfo ModInfo
	err = json.Unmarshal(bmifBa, &modInfo)
	if err != nil {
		log.Printf("error unmarshalling baseMods info.json to a modInfo: %s", err)
		return
	}

	server.BaseModVersion = modInfo.Version

	// load admins from additional file
	if (server.Version.Greater(Version{0, 17, 0})) {
		if _, err = os.Stat(config.FactorioAdminFile); os.IsNotExist(err) {
			//save empty admins-file
			err = ioutil.WriteFile(config.FactorioAdminFile, []byte("[]"), 0664)
			server.Settings["admins"] = make([]string, 0)
		} else {
			var data []byte
			data, err = ioutil.ReadFile(config.FactorioAdminFile)
			if err != nil {
				log.Printf("Error loading FactorioAdminFile: %s", err)
				return
			}

			var jsonData interface{}
			err = json.Unmarshal(data, &jsonData)
			if err != nil {
				log.Printf("Error unmarshalling FactorioAdminFile: %s", err)
				return
			}

			server.Settings["admins"] = jsonData
		}
	}

	SetFactorioServer(server)

	// The status pill carries the update rate, so the sampler starts with the
	// manager itself and stays out of the way while no game is running.
	startUpsSampler()

	// autostart factorio is configured to do so
	if config.Autostart == "true" {
		go instantiated.autostart()
	}

	return
}

func GetFactorioServer() (f *Server) {
	return &instantiated
}

func (server *Server) Run() error {
	var err error
	config := bootstrap.GetConfig()
	data, err := json.MarshalIndent(server.Settings, "", "  ")
	if err != nil {
		log.Println("Failed to marshal FactorioServerSettings: ", err)
	} else {
		ioutil.WriteFile(config.SettingsFile, data, 0644)
	}

	saves, err := ListSaves()
	if err != nil {
		log.Println("Failed to get saves list: ", err)
	}

	if len(saves) == 0 {
		return errors.New("No savefile exists on the server")
	}

	args := []string{}

	//The factorio server refenences its executable-path, since we execute the ld.so file and pass the factorio binary as a parameter
	//the game would use the path to the ld.so file as it's executable path and crash, to prevent this the parameter "--executable-path" is added
	if config.GlibcCustom == "true" {
		log.Println("Custom glibc selected, glibc.so location:", config.GlibcLocation, " lib location:", config.GlibcLibLoc)
		args = append(args, "--library-path", config.GlibcLibLoc, config.FactorioBinary, "--executable-path", config.FactorioBinary)
	}

	args = append(args,
		"--bind", server.BindIP,
		"--port", strconv.Itoa(server.Port),
		"--server-settings", config.SettingsFile,
		"--rcon-port", strconv.Itoa(config.FactorioRconPort),
		"--rcon-password", config.FactorioRconPass)

	if (server.Version.Greater(Version{0, 17, 0})) {
		args = append(args, "--server-adminlist", config.FactorioAdminFile)
	}

	if strings.HasPrefix(server.Savefile, "Load Latest") {
		args = append(args, "--start-server-load-latest")
	} else {
		args = append(args, "--start-server", filepath.Join(config.FactorioSavesDir, server.Savefile))
	}

	// Write chat log to a different file if requested (if not it will be mixed-in with the default logfile)
	if config.ChatLogFile != "" {
		args = append(args, "--console-log", config.ChatLogFile)
	}

	if config.GlibcCustom == "true" {
		log.Println("Starting server with command: ", config.GlibcLocation, args)
		server.Cmd = exec.Command(config.GlibcLocation, args...)
	} else {
		log.Println("Starting server with command: ", config.FactorioBinary, args)
		server.Cmd = exec.Command(config.FactorioBinary, args...)
	}

	server.StdOut, err = server.Cmd.StdoutPipe()
	if err != nil {
		log.Printf("Error opening stdout pipe: %s", err)
		return err
	}

	server.StdIn, err = server.Cmd.StdinPipe()
	if err != nil {
		log.Printf("Error opening stdin pipe: %s", err)
		return err
	}

	server.StdErr, err = server.Cmd.StderrPipe()
	if err != nil {
		log.Printf("Error opening stderr pipe: %s", err)
		return err
	}

	go server.parseRunningCommand(server.StdOut)
	go server.parseRunningCommand(server.StdErr)

	err = server.Cmd.Start()
	if err != nil {
		log.Printf("Factorio process failed to start: %s", err)
		return err
	}
	server.SetRunning(true)

	err = server.Cmd.Wait()
	log.Printf("Factorio process is closed")
	server.SetRunning(false)
	if err != nil {
		log.Printf("Factorio process exited with error: %s", err)
		return err
	}

	return nil
}

func (server *Server) parseRunningCommand(std io.ReadCloser) (err error) {
	stdScanner := bufio.NewScanner(std)
	for stdScanner.Scan() {
		text := stdScanner.Text()

		log.Printf("Factorio Server: %s", text)
		if err := server.writeLog(text); err != nil {
			log.Printf("Error: %s", err)
		}

		// send the reported line per websocket
		wsRoom := websocket.WebsocketHub.GetRoom("gamelog")
		go wsRoom.Send(text)

		line := strings.Fields(text)
		// Ensure logline slice is in bounds
		if len(line) > 1 {
			// Check if Factorio Server reports any errors if so handle it
			if line[1] == "Error" {
				err := server.checkLogError(line)
				if err != nil {
					log.Printf("Error checking Factorio Server Error: %s", err)
				}
			}
			// If rcon port opens indicated in log connect to rcon
			rconLog := "Starting RCON interface at IP"
			// check if slice index is greater than 2 to prevent panic
			if len(line) > 2 {
				// log line for opened rcon connection
				if strings.Contains(text, rconLog) {
					log.Printf("Rcon running on Factorio Server")
					err = connectRC()
					if err != nil {
						log.Printf("Error: %s", err)
					}
				}

				server.checkProcessHealth(text)
			}
		}
	}
	if err := stdScanner.Err(); err != nil {
		log.Printf("Error reading std buffer: %s", err)
		return err
	}
	return nil
}

func (server *Server) writeLog(logline string) error {
	config := bootstrap.GetConfig()
	logfileName := config.ConsoleLogFile
	file, err := os.OpenFile(logfileName, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Printf("Cannot open logfile %s for appending Factorio Server output: %s", logfileName, err)
		return err
	}
	defer file.Close()

	logline = logline + "\n"

	if _, err = file.WriteString(logline); err != nil {
		log.Printf("Error appending to %s: %s", logfileName, err)
		return err
	}

	return nil
}

func (server *Server) checkLogError(logline []string) error {
	// TODO Handle errors generated by running Factorio Server
	log.Println(logline)

	return nil
}

func init() {
	websocket.WebsocketHub.RegisterControlHandler <- serverWebsocketControl
}

// react to websocket control messages and run the command if it is requested
func serverWebsocketControl(controls websocket.WsControls) {
	log.Println(controls)
	if controls.Type == "command" {
		command := controls.Value
		server := GetFactorioServer()
		if server.GetRunning() {
			log.Printf("Received command: %v", command)

			reqId, err := server.Rcon.Write(command)
			if err != nil {
				log.Printf("Error sending rcon command: %s", err)
				return
			}

			log.Printf("Command send to Factorio: %s, with rcon request id: %v", command, reqId)
		}
	}
}
