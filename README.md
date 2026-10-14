[![.github/workflows/test-workflow.yml](https://github.com/OpenFactorioServerManager/factorio-server-manager/workflows/.github/workflows/test-workflow.yml/badge.svg)](https://github.com/OpenFactorioServerManager/factorio-server-manager/actions)
[![Discord](https://img.shields.io/discord/779512040934342687?label=Discord)](https://discord.gg/SB647WmSbU)

# Factorio Server Manager

### A tool for managing Factorio servers.
This tool runs on a Factorio server and allows management of the Factorio server, saves, mods and many other features.

## Features
* Allows control of the Factorio Server, starting and stopping the Factorio binary.
* Allows the management of save files, upload, download and delete saves.
* Manage installed mods, upload new ones and more
* Manage modpacks, so it is easier to play with different configurations
* Allow viewing of the server logs and current configuration.
* Authentication for protecting against unauthorized users
* Available as a Docker container

#### Manage Factorio Server
![Factorio Server Manager Screenshot](screenshots/Screenshot_Controls.png)

#### Manage save files
![Factorio Server Manager Screenshot](screenshots/Screenshot_Saves.png)

#### Manage mods
![Factorio Server Manager Screenshot](screenshots/Screenshot_Mods.png)

## Companion mod

This fork embeds a companion mod, `fsm-companion`. On every startup the manager
writes it into the server's `mods/` directory as `fsm-companion_<version>.zip` and
adds it to `mod-list.json` enabled, so it is installed and on by default with
nothing to do. The zip is deliberate: the manager lists mods by walking `.zip`
files, so an unpacked directory would load in the game but never appear on the
Mods page. Disabling it is remembered: an entry that is already in
`mod-list.json` keeps whatever state it has, so turning the companion off in the
mod list is not undone by a restart. The
mod source lives in the `companion-mod` repository and is vendored into
`src/companion/mod/` by `make sync-companion`, then embedded into the binary with
Go `//go:embed`.

## Configuration

Four variables are required. The container refuses to start if any of them is
missing, and every one of them is re-applied on each start:

| Variable | Meaning |
|---|---|
| `ADMIN_USER` | Username of the admin account. |
| `ADMIN_PASS` | Password of the admin account. |
| `RCON_PASS` | RCON password the manager and the game server share. |
| `COOKIE_ENCRYPTION_KEY` | Base64 key that signs the login session cookie; must decode to 16, 24 or 32 bytes (`openssl rand -base64 32`). |

The admin credentials are configuration, not state. They are written into
`conf.json` and enforced in the user database on every launch, so a restart puts
them back even if they were changed in the interface: the account cannot be
locked out or drifted from what the operator configured. Additional users can be
added as normal; deleting the last admin is still refused. `FACTORIO_VERSION` is
the one optional variable and defaults to `stable`.

## Container

The `compose.yaml` and `Dockerfile` at the repository root build the fork from
source (webpack UI plus the Go binary) and run it with the Factorio runtime
dependencies. The game port is `34197/udp` and the web UI binds to
`127.0.0.1:7090` on the host. State lives in `./fsm-data` and `./factorio`.

## [Installation and Usage](https://github.com/OpenFactorioServerManager/factorio-server-manager/wiki/Installation-and-Usage)

## [Development](https://github.com/OpenFactorioServerManager/factorio-server-manager/wiki/Development)

## Contributing
1. Fork it!
2. Checkout the develop branch, only use that as a base: `git checkout develop`
2. Create your feature branch: `git checkout -b my-new-feature`
3. Commit your changes: `git commit -am 'Add some feature'`
4. Add your changes a in human readable way into CHANGELOG.md
4. Push to the branch: `git push origin my-new-feature`
5. Submit a pull request, with `develop` as base :D

## Authors

* **Mitch Roote** - [roote.ca](https://roote.ca)
* **[knoxfighter](https://github.com/knoxfighter)**
* **[Jannaahs](https://github.com/jannaahs)**

## Special Thanks
- **[All Contributions](https://github.com/OpenFactorioServerManager/factorio-server-manager/graphs/contributors)**
- **mickael9** for reverseengineering the factorio-save-file: https://forums.factorio.com/viewtopic.php?f=5&t=8568#

## License

This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details
