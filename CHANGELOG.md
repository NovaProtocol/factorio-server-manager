# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [0.11.0] - TBD
### Added
- A Game Speed panel on the Controls page: set the game's speed as a speed (1 = normal) or as a max UPS (0.5 = 30 UPS), Save applies it in game at once and the companion mod re-applies it after every save load. Only usable while the game is running, admin only. Each field owns its value while focused, the twin is solved on focus-out, and Save sends what was typed rather than what is displayed, so 1 max UPS stays 1 instead of 1.00002
- Every manager-to-game call is now a plain mod command. The UPS sampler used to ask the game with `/sc` (a script command, which permanently disables achievements for a save); the companion mod's `companion-tick` and `companion-speed` commands are plain commands and used instead
- The Server Status pill in the sidebar and on the Controls page reports the game's real update rate: the manager asks the game for its tick every 5 seconds and divides by wall time, so the pill reads `Running - 20.6 UPS`, and it goes to `Running - 0.0 UPS` when the game is paused
- The Saves list sorts by clicking a column header: name, last modified or size, and clicking the same header again reverses the order (size and date start with the biggest and newest first). The Factorio version column is deliberately not sortable, since its value arrives from a separate header read and rows would reorder under the cursor
- Credentials page under FSM Administration: sign in to the Factorio mod portal with a password or with a personal token, sign out, and see the account data the stored credentials reach (the account's bookmarked mods)
- Role-based access. `admin` keeps full access; `user` is read-only: console, mod list, dashboard and server status only, with no logs, settings, saves, users or credentials
- The Create User form can now create `user`-role accounts; it previously only ever created admins
- The Mods page reports the companion mod's pinned version, where the running copy came from, and fetches it on demand
- Three dashboard pages under Game replacing the single Dashboard: **Power**, **Production** and **Science**, each with a time-window picker (5s to 1000h, plus all time) and a graph of the 300 samples behind the game's own statistics graphs
- Power is shown in real units (kW/MW/GW) converted from the game's joules-per-tick, with both production and consumption
- The Power page opens on every planet's grid at once, and the Production page on every planet's output; clicking a row graphs that item, fluid or prototype

### Changed
- The dashboards keep their last report in memory for the life of the manager instead of expiring it after five seconds, so a page opens on the last known numbers — minutes, days or weeks old — and only says how old they are. Nothing is written to disk, and refreshes still happen behind the request every five seconds
- Removed the explanatory paragraphs under the dashboards, the console, the credentials page, the companion card and the game settings page. The panels show their data and their controls, nothing else
- The compose file can be pointed at an existing installation: `FSM_DATA_DIR` and `FACTORIO_DIR` say where the manager's state and the game's data live, so an upgrade keeps the users, saves, mods and server settings already there instead of starting empty. `FSM_WEB_BIND`, `FSM_WEB_PORT` and `FSM_GAME_PORT` move the published ports, so a second installation can sit beside the first
- The dashboards read the companion mod one report scope at a time, cached for 5 seconds and shared between viewers, instead of polling one large document every 20 seconds; a browser poll is never queued behind a running fetch — it is answered with the last data and refreshed on the next poll
- Time windows are a row of tabs like the settings pages, not a dropdown; graphs are Chart.js so bars can be hovered for a readout
- Every dashboard list has a filter box, and long tables scroll with a themed scrollbar instead of the browser default
- Science per minute counts only the packs the current research needs, and a second figure is measured from the research bar to show the effect of lab productivity
- The measured science figure survives a server pause: the mod keeps the last good reading with its game-time age, and the tile says when a number was held rather than freshly measured
- Graphs overlay production and consumption as two lines, with both numbers on hover
- `tailwind.config.js` lists `transparent` and `current` explicitly: `colors` replaces the palette, so `bg-transparent` was silently generating nothing
- The companion mod is no longer embedded in the manager binary. The manager installs the one version named by `CompanionVersion`, fetched from the companion repository's GitHub release (public, so no credential is needed), with the mod portal as a fallback when a credential is stored
- Removed `make sync-companion`, `scripts/sync-companion.sh` and the vendored `src/companion/mod/`; the `companion-mod` repository is now the only copy of the mod source
- Configuration environment variables are now uppercase and prefixed with FSM
- updated all dependencies - Thanks to @jannaahs and @knoxfighter
- removed CGO as dependency

### Security
- Nothing the manager sends the game on its own can stop achievements: every call is one of the companion mod's plain commands (`/companion-report`, `/companion-chunk`, `/companion-tick`, `/companion-speed`). The console page passes text through as typed, so a script command typed there is the admin's own doing
- The login endpoint is rate limited per address and username: eight failed attempts inside five minutes answer `429` with a `Retry-After`, and a successful login clears the counter. It is the only unauthenticated endpoint, so it is the only throttled one
- The routes that change server state — stop, kill, remove save, create save — are `POST`, not `GET`, so a link or an `<img>` cannot trigger them; the session cookie states `SameSite=Lax` explicitly rather than relying on the browser default
- `conf.json` (admin and RCON passwords in clear) is `chmod 0600` on every start, and `factorio.auth` (the mod portal token) is written `0600`

### Fixed
- Every form in the interface saw its fields as empty when submitted — signing in, adding a user, storing a mod portal credential — because `Input` replaced react-hook-form's own `onChange`/`onBlur` with whatever the caller passed. It calls both now, and `InputPassword`, which leaned on that same override to pass the registration through as props, is fixed with it
- The status broadcast no longer remounts the page being viewed. The route guards in `App` and the sidebar's `Status` and `Link` were declared inside their parents, so every render created new component types and React unmounted the whole subtree: the five-second UPS broadcast refetched each page and discarded its state, so the saves list reloaded itself and lost its sort
- Adding a user whose name already exists is refused with `400` and "User <name> already exists" instead of quietly creating a second account, and startup warns when the database already holds duplicate names — the username's unique index cannot be created while they are there
- `go vet` is clean: `lockfile.NewLock` returns a pointer instead of copying a mutex, and the three save/version readers no longer shadow the `io.ReaderFrom` contract with a mismatched signature
- Removed the unused `fuse.js` dependency; the lockfile is back in sync with `package.json`
- Every compose file caps the container log at 10 MB × 3 files and carries a healthcheck, instead of leaving the docker log unbounded
- `docker/docker-compose.yaml` and `docker-compose.simple.yaml` now build this repository. They used to pull `ofsm/ofsm:latest`, which is upstream and has none of the features in this changelog

## [0.10.1] - 2021-03-09
### Fixed
- Single admin user can no longer be deleted (so there is always a user)
- fixed incompatibility to glibc 2.32 by linking dynamic on linux
- Moved from alpine to ubuntu docker image base, to prevent factorio not running correctly

## [0.10.0] - 2021-02-10
### Added
- Config files can be defined with absolute paths. - Thanks to @FoxAmes
- Support for >= 1.1.14 factorio saves - Thanks to @knoxfighter
- Setting in `info.json` to allow usage without ssl/tls - Thanks to @knoxfighter

### Changed
- Rework of the authentication, to have a bit more security. - Thanks to @knoxfighter
- Changed from leveldb to sqlite3 as backend database. - Thanks to @knoxfighter
- generate new random passwords, if no exist, or if they are "factorio". - Thanks to @knoxfighter
- Use "OpenFactorioServerManager" instead of "mroote" as go package name. - Thanks to @mroote
- Disable mods-page, while server is running - Thanks to @knoxfighter
- Renamed GO-package from `mroote` to `OpenFactorioServerManager` to match git repo - Thanks to @mroote

### Fixed
- old factorio versions depended by mods always shown as compatible - Thanks to @knoxfighter
- Crosscompilation with mingw-w64 on linux. (Broke with sqlite3) - Thanks to @knoxfighter
- Crash on async writing to websocket room array. - Thanks to @knoxfighter

## [0.9.0] - 2021-01-07
### Added
- Autostart factorio, when starting the server-manager - Thanks to @Psychomantis71

### Changed
- Complete rework of the UI - Thanks to @jannaahs
- Backend is refactored and improved - Thanks to @knoxfighter and @jannaahs
- Rework of the docker image, so it allows easy updating of factorio - Thanks to @ita-sammann

### Fixed
- Console page is now working correctly - Thanks to @jannaahs
- Mod Search fixed by new implementation, which does not rely on the search endpoint of the mod portal - Thanks to @jannaahs
- Listen on port 80, previously port 8080 was used. Can be changed with `--port <port>`
- Update version numbers in Docker containers

## [0.8.2] - 2020-01-08
Many bugfixes and a few small features in this release.
- Adds a flag for a custom glibc version, required on some distros such as CentOS
- bugfixes with file handling
- UI fixes and improvements
- CI bug fixes and build improvements
- and more bugfixes

Special thanks to @knoxfighter for all the contributions.

### Added
- Support for 0.17 server-adminlist.json
- Support for custom glibc location (RHEL/CENTOS)

### Changed
- Use bootstrap-fileinputs for savefile upload
- Login-Page uses bootstrap 4

### Fixed
- Login Page Design
- Sweetalert2 API changes
- allow_commands not misinterpreted as boolean anymore
- Fixed some filepaths on windows
- Fixed hardcoded Settings Path
- Fixed Upgrading, Removing Mods on Windows results in error

## [0.8.1] - 2019-03-01
### Fixed
- Fixed redirect, when not logged in
- Fixed login page completely white

## [0.8.0] - 2019-02-27
This release contains many bug fixes and features. Thanks to @knoxfighter @sean-callahan for the contributions!
- Fixes error in Factorio 0.17 saves
- Refactors and various bug fixes

## [0.7.5] - 2018-08-08
## Fixed
- fixes crash when mods have no basemodversion defined

## [0.7.4] - 2018-08-04
- Ability to auto download mods used in a save file courtesy @knoxfighter
- Fix bug in mod logging courtesy @c0nnex

## [0.7.3] - 2018-06-02
- Fixes fields in the settings dialog unable to be set to false. Courtesy @winadam.
- Various bugfixes in the mod settings page regarding version compatability. Courtesy @knoxfighter.

## [0.7.2] - 2018-05-02
### Fixed
- Fixes an error when searching in the mod portal.

## [0.7.1] - 2018-02-11
### Fixed
- Fixes an error in the configuration form where some fields were not editable.

## [0.7.0] - 2018-01-21
- Rewritten mods section now supporting installing mods directly from the Factorio mod portal and many other features courtesy @knoxfighter
- Various bug fixes

## [0.6.1] - 2017-12-22
- Adds the ability to specify the IP address for the Factorio game server to bind too.
- Updates the --rcon-password flag
- Small fixes

## [0.6.0] - 2017-01-25
This release adds a console feature using rcon to send commands and chat from the management interface.

## [0.5.2] - 2016-11-01
This release moves the server-settings.json config file. It will now save the file in the factorio/config directory.

## [0.5.1] - 2016-10-31
- Fixed bug where server-settings.json file is overwritten with default settings
- Started adding UI for editing the server-settings.json file

## [0.5.0] - 2016-10-11
- This release adds beta support for Windows users.
- Various updates for Factorio 0.14 are also included.

## [0.4.3] - 2016-09-15
This release has some small bug fixes in order to support Factorio server 0.14.
- Made the --latency-ms optional as it is removed in version 0.14
- Improved some error handling messages when starting the server.

## [0.4.2] - 2016-07-13
This release fixes a bug with Factorio 0.13 where the full path for save files must be specified when launching the server.

## [0.4.1] - 2016-05-15
This release fixes a bug where the UI reports an error when the Factorio Server was successfully started.

## [0.4.0] - 2016-05-15
### New features
- Abillity to create modpacks for easy distribution of mods
- Multiple users are now supported, create and delete users in the settings menu

### Features
- Allows control of the Factorio Server, starting and stopping the Factorio binary.
- Allows the management of save files, upload, download and delete saves.
- Manage installed mods, upload new ones, delete uneeded mods. Enable or disable individual mods.
- Allow viewing of the server logs and current configuration.
- Authentication for protecting against unauthorized users
- Available as a Docker container
- Abillity to create modpacks for easy distribution of mods
- Multiple users are now supported, create and delete users in the settings menu

## [0.3.1] - 2016-05-03
### Fixed
Fixes bug in #24 where Docker container cannot find conf.json file.

## [0.3.0] - 2016-05-01
### New features
- This release adds an authentication feature in Factorio Server Manager.
- Now able to be installed as a Docker container.
- Admin user credentials are configured in the conf.json file included in the release zip file.

### Features
- Allows control of the Factorio Server, starting and stopping the Factorio binary.
- Allows the management of save files, upload, download and delete saves.
- Manage installed mods, upload new ones, delete uneeded mods. Enable or disable individual mods.
- Allow viewing of the server logs and current configuration.
- Authentication for protecting against unauthorized users
- Available as a Docker container

## [0.2.0] - 2016-04-27
This release adds the ability to control the Factorio server. Allows stopping and starting of the server binary with advanced options.

### Features
- Allows control of the Factorio Server, starting and stopping the Factorio binary.
- Allows the management of save files, upload, download and delete saves.
- Manage installed mods, upload new ones, delete uneeded mods. Enable or disable individual mods.
- Allow viewing of the server logs and current configuration.

## [0.1.0] - 2016-04-25
First release of Factorio Server Manager

### Features
- Managing save files, create, download, delete saves
- Managing installed mods
- Factorio log tailing
- Factorio server configuration viewing
