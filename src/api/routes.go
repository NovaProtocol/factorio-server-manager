package api

import (
	"github.com/OpenFactorioServerManager/factorio-server-manager/api/websocket"
	"github.com/OpenFactorioServerManager/factorio-server-manager/factorio"
	"net/http"

	"github.com/gorilla/mux"
)

type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc http.HandlerFunc
	ServerOff   bool // Set to `true' if factorio server has to be turned off to call this
}

type Routes []Route

func ServerOffMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// only run if server is turned off
		server := factorio.GetFactorioServer()
		if server.GetRunning() {
			http.Error(w, "factorio server still running", http.StatusLocked)
		} else {
			next.ServeHTTP(w, r)
		}
		return
	})
}

func NewRouter() *mux.Router {
	mainRouter := mux.NewRouter().StrictSlash(true)

	// create subrouter for authenticated calls
	subRouter := mainRouter.NewRoute().Subrouter()
	subRouter.Use(AuthMiddleware)

	// API subrouter
	// Serves all JSON REST handlers prefixed with /api
	apiRouter := mainRouter.PathPrefix("/api").Subrouter()
	apiRouter.Use(AuthMiddleware)

	// use subrouter for calls, that run only, when server is turned off
	serverOffRouter := apiRouter.NewRoute().Subrouter()
	serverOffRouter.Use(ServerOffMiddleware)

	apiRouter.NewRoute().Subrouter()
	for _, route := range apiRoutes {
		var router *mux.Router
		if route.ServerOff {
			router = serverOffRouter
		} else {
			router = apiRouter
		}
		router.Methods(route.Method).
			Path(route.Pattern).
			Name(route.Name).
			Handler(route.HandlerFunc)
	}

	// The login handler does not check for authentication.
	mainRouter.Path("/api/login").
		Methods("POST").
		Name("LoginUser").
		HandlerFunc(LoginUser)

	// Route for initializing websocket connection
	// Clients connecting to /ws establish websocket connection by upgrading
	// HTTP session.
	// Ensure user is logged in with the AuthorizeHandler middleware
	subRouter.Path("/ws").
		Methods("GET").
		Name("Websocket").
		Handler(
			http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					websocket.ServeWs(w, r)
				},
			),
		)

	// Serves the frontend application from the app directory
	// Uses basic file server to serve index.html and Javascript application
	// Routes match the ones defined in React frontend application
	mainRouter.Path("/login").
		Methods("GET").
		Name("Login").
		Handler(http.StripPrefix("/login", http.FileServer(http.Dir("./app/"))))

	subRouter.Path("/saves").
		Methods("GET").
		Name("Saves").
		Handler(http.StripPrefix("/saves", http.FileServer(http.Dir("./app/"))))
	subRouter.Path("/mods").
		Methods("GET").
		Name("Mods").
		Handler(http.StripPrefix("/mods", http.FileServer(http.Dir("./app/"))))
	subRouter.Path("/server-settings").
		Methods("GET").
		Name("Server settings").
		Handler(http.StripPrefix("/server-settings", http.FileServer(http.Dir("./app/"))))
	subRouter.Path("/game-settings").
		Methods("GET").
		Name("Game settings").
		Handler(http.StripPrefix("/game-settings", http.FileServer(http.Dir("./app/"))))
	subRouter.Path("/console").
		Methods("GET").
		Name("Console").
		Handler(http.StripPrefix("/console", http.FileServer(http.Dir("./app/"))))
	subRouter.Path("/logs").
		Methods("GET").
		Name("Logs").
		Handler(http.StripPrefix("/logs", http.FileServer(http.Dir("./app/"))))
	subRouter.Path("/user-management").
		Methods("GET").
		Name("User management").
		Handler(http.StripPrefix("/user-management", http.FileServer(http.Dir("./app/"))))
	subRouter.Path("/help").
		Methods("GET").
		Name("Help").
		Handler(http.StripPrefix("/help", http.FileServer(http.Dir("./app/"))))
	subRouter.Path("/dashboard").
		Methods("GET").
		Name("Dashboard").
		Handler(http.StripPrefix("/dashboard", http.FileServer(http.Dir("./app/"))))
	subRouter.Path("/dashboard-power").
		Methods("GET").
		Name("Dashboard power").
		Handler(http.StripPrefix("/dashboard-power", http.FileServer(http.Dir("./app/"))))
	subRouter.Path("/dashboard-production").
		Methods("GET").
		Name("Dashboard production").
		Handler(http.StripPrefix("/dashboard-production", http.FileServer(http.Dir("./app/"))))
	subRouter.Path("/dashboard-science").
		Methods("GET").
		Name("Dashboard science").
		Handler(http.StripPrefix("/dashboard-science", http.FileServer(http.Dir("./app/"))))
	subRouter.Path("/credentials").
		Methods("GET").
		Name("Credentials").
		Handler(http.StripPrefix("/credentials", http.FileServer(http.Dir("./app/"))))

	// catch all route
	mainRouter.PathPrefix("/").
		Methods("GET").
		Name("Index").
		Handler(http.FileServer(http.Dir("./app/")))

	return mainRouter
}

// Defines all API REST endpoints
// All routes are prefixed with /api
//
// A route reads permission from its handler: handlers wrapped in RequireAdmin
// answer 403 to a non-admin, every other handler is open to any authenticated
// account. The `user` role is therefore read-only by construction — it can only
// reach the unwrapped handlers below.
var apiRoutes = Routes{
	// Saves — admin only
	{
		"ListSaves",
		"GET",
		"/saves/list",
		RequireAdmin(ListSaves),
		false,
	}, {
		"DlSave",
		"GET",
		"/saves/dl/{save}",
		RequireAdmin(DLSave),
		false,
	}, {
		"UploadSave",
		"POST",
		"/saves/upload",
		RequireAdmin(UploadSave),
		false,
	}, {
		"RemoveSave",
		"POST",
		"/saves/rm/{save}",
		RequireAdmin(RemoveSave),
		false,
	}, {
		"CreateSave",
		"POST",
		"/saves/create/{save}",
		RequireAdmin(CreateSaveHandler),
		true,
	}, {
		"LoadModsFromSave",
		"POST",
		"/saves/mods",
		RequireAdmin(LoadModsFromSaveHandler),
		true,
	}, {
		"SaveInfo",
		"GET",
		"/saves/info/{save}",
		RequireAdmin(SaveInfoHandler),
		false,
	}, {
		// Logs — admin only
		"LogTail",
		"GET",
		"/log/tail",
		RequireAdmin(LogTail),
		false,
	}, {
		// The Factorio config.ini, shown on the server settings page — admin only
		"LoadConfig",
		"GET",
		"/config",
		RequireAdmin(LoadConfig),
		false,
	}, {
		// Server control — admin only
		"StartServer",
		"POST",
		"/server/start",
		RequireAdmin(StartServer),
		true,
	}, {
		"StopServer",
		"POST",
		"/server/stop",
		RequireAdmin(StopServer),
		false,
	}, {
		"KillServer",
		"POST",
		"/server/kill",
		RequireAdmin(KillServer),
		false,
	}, {
		// Read-only status, needed by every page including the dashboard
		"RunningServer",
		"GET",
		"/server/status",
		CheckServer,
		false,
	}, {
		"FactorioVersion",
		"GET",
		"/server/facVersion",
		FactorioVersion,
		false,
	}, {
		"LogoutUser",
		"GET",
		"/logout",
		LogoutUser,
		false,
	}, {
		"StatusUser",
		"GET",
		"/user/status",
		GetCurrentLogin,
		false,
	}, {
		// User management — admin only
		"ListUsers",
		"GET",
		"/user/list",
		RequireAdmin(ListUsers),
		false,
	}, {
		"AddUser",
		"POST",
		"/user/add",
		RequireAdmin(AddUser),
		false,
	}, {
		"RemoveUser",
		"POST",
		"/user/remove",
		RequireAdmin(RemoveUser),
		false,
	}, {
		// Any account may change its own password
		"ChangePassword",
		"POST",
		"/user/password",
		ChangePassword,
		false,
	}, {
		// Server settings — admin only
		"GetServerSettings",
		"GET",
		"/settings",
		RequireAdmin(GetServerSettings),
		false,
	}, {
		"UpdateServerSettings",
		"POST",
		"/settings/update",
		RequireAdmin(UpdateServerSettings),
		false,
	}, {
		"GetServerSettingsDefaults",
		"GET",
		"/settings/defaults",
		RequireAdmin(GetServerSettingsDefaults),
		false,
	}, {
		// Game settings — admin only
		"GameSettings",
		"GET",
		"/game-settings",
		RequireAdmin(GameSettingsHandler),
		false,
	}, {
		"SaveGameSettings",
		"POST",
		"/game-settings",
		RequireAdmin(SaveGameSettingsHandler),
		true,
	}, {
		"DumpGameSettings",
		"POST",
		"/game-settings/dump",
		RequireAdmin(DumpGameSettingsHandler),
		true,
	},
	// Mod Portal browsing is public within the manager; installing a mod is not.
	{
		"ModPortalListAllMods",
		"GET",
		"/mods/portal/list",
		ModPortalListModsHandler,
		false,
	}, {
		"ModPortalGetModInfo",
		"GET",
		"/mods/portal/info/{mod}",
		ModPortalModInfoHandler,
		false,
	}, {
		"ModPortalInstallMod",
		"POST",
		"/mods/portal/install",
		RequireAdmin(ModPortalInstallHandler),
		true,
	}, {
		"ModPortalInstallMultiple",
		"POST",
		"/mods/portal/install/multiple",
		RequireAdmin(ModPortalInstallMultipleHandler),
		true,
	},
	// Factorio mod portal credentials — admin only. A stored credential is what
	// makes a gated mod download work, so it is managed from one place.
	{
		"CredentialsLogin",
		"POST",
		"/credentials/login",
		RequireAdmin(ModPortalLoginHandler),
		false,
	}, {
		"CredentialsToken",
		"POST",
		"/credentials/token",
		RequireAdmin(ModPortalTokenLoginHandler),
		false,
	}, {
		"CredentialsStatus",
		"GET",
		"/credentials/status",
		RequireAdmin(ModPortalLoginStatusHandler),
		false,
	}, {
		"CredentialsLogout",
		"GET",
		"/credentials/logout",
		RequireAdmin(ModPortalLogoutHandler),
		false,
	},
	// Mods — listing is open, everything that changes state is admin only.
	{
		"ListInstalledMods",
		"GET",
		"/mods/list",
		ListInstalledModsHandler,
		false,
	}, {
		"ToggleMod",
		"POST",
		"/mods/toggle",
		RequireAdmin(ModToggleHandler),
		true,
	}, {
		"DeleteMod",
		"POST",
		"/mods/delete",
		RequireAdmin(ModDeleteHandler),
		true,
	}, {
		"DeleteAllMods",
		"POST",
		"/mods/delete/all",
		RequireAdmin(ModDeleteAllHandler),
		true,
	}, {
		"UpdateMod",
		"POST",
		"/mods/update",
		RequireAdmin(ModUpdateHandler),
		true,
	}, {
		"UploadMod",
		"POST",
		"/mods/upload",
		RequireAdmin(ModUploadHandler),
		true,
	}, {
		"DownloadMods",
		"GET",
		"/mods/download",
		ModDownloadHandler,
		false,
	}, {
		"DownloadCompanion",
		"GET",
		"/mods/companion/download",
		CompanionDownloadHandler,
		false,
	}, {
		// The pinned version the manager installs, and where it came from
		"CompanionStatus",
		"GET",
		"/mods/companion/status",
		CompanionStatusHandler,
		false,
	}, {
		// Fetching installs a mod, so it follows the same rule as every other
		// mod change: refused while the server is running.
		"CompanionInstall",
		"POST",
		"/mods/companion/install",
		RequireAdmin(CompanionInstallHandler),
		true,
	},
	// Mod Packs — listing and downloading are open, the rest is admin only.
	{
		"ModPacksList",
		"GET",
		"/mods/packs/list",
		ModPackListHandler,
		false,
	}, {
		"ModPackCreate",
		"POST",
		"/mods/packs/create",
		RequireAdmin(ModPackCreateHandler),
		false,
	}, {
		"ModPackDelete",
		"POST",
		"/mods/packs/{modpack}/delete",
		RequireAdmin(ModPackDeleteHandler),
		false,
	}, {
		"ModPackDownload",
		"GET",
		"/mods/packs/{modpack}/download",
		ModPackDownloadHandler,
		false,
	}, {
		"LoadModPack",
		"POST",
		"/mods/packs/{modpack}/load",
		RequireAdmin(ModPackLoadHandler),
		true,
	},
	// Mods inside Mod Packs
	{
		"ModPackListMods",
		"GET",
		"/mods/packs/{modpack}/list",
		ModPackModListHandler,
		false,
	}, {
		"ModPackToggleMod",
		"POST",
		"/mods/packs/{modpack}/mod/toggle",
		RequireAdmin(ModPackModToggleHandler),
		false,
	}, {
		"ModPackDeleteMod",
		"POST",
		"/mods/packs/{modpack}/mod/delete",
		RequireAdmin(ModPackModDeleteHandler),
		false,
	}, {
		"ModPackDeleteAllMod",
		"POST",
		"/mods/packs/{modpack}/mod/delete/all",
		RequireAdmin(ModPackModDeleteAllHandler),
		false,
	}, {
		"ModPackUpdateMod",
		"POST",
		"/mods/packs/{modpack}/mod/update",
		RequireAdmin(ModPackModUpdateHandler),
		false,
	}, {
		"ModPackUploadMod",
		"POST",
		"/mods/packs/{modpack}/mod/upload",
		RequireAdmin(ModPackModUploadHandler),
		false,
	}, {
		"ModPackModPortalInstallMod",
		"POST",
		"/mods/packs/{modpack}/portal/install",
		RequireAdmin(ModPackModPortalInstallHandler),
		false,
	}, {
		"ModPackModPortalInstallMultiple",
		"POST",
		"/mods/packs/{modpack}/portal/install/multiple",
		RequireAdmin(ModPackModPortalInstallMultipleHandler),
		true,
	},
	// Companion dashboards — read-only, one endpoint per page so a page only
	// pays for the slice it is showing.
	{
		"CompanionOverview",
		"GET",
		"/companion/overview",
		CompanionOverviewHandler,
		false,
	}, {
		"CompanionProduction",
		"GET",
		"/companion/production",
		CompanionProductionHandler,
		false,
	}, {
		"CompanionPower",
		"GET",
		"/companion/power",
		CompanionPowerHandler,
		false,
	}, {
		"CompanionScience",
		"GET",
		"/companion/science",
		CompanionScienceHandler,
		false,
	}, {
		"CompanionGraph",
		"GET",
		"/companion/graph",
		CompanionGraphHandler,
		false,
	}, {
		"CompanionSpeed",
		"POST",
		"/companion/speed",
		RequireAdmin(CompanionSpeedHandler),
		false,
	},
}
