<div align="center">
  <img src="assets/logo.png" alt="Discord Portable" width="128" height="128" />

  <h1>Discord Portable (Go)</h1>

  <p>
    <strong>One-folder, virtual-disk-friendly Discord for kiosks.</strong><br/>
    Silent client launcher + admin updater, written in Go.
  </p>
</div>

---

A two-binary, fully portable Discord setup for kiosks and virtual disks that
reset to a clean state on reboot. The whole client is kept in one folder so
admin updates and end-user launches stay separate, predictable, and silent.

## Why

Standard Discord installs scatter files across `%LOCALAPPDATA%\Discord` and
`%APPDATA%\discord`. On a kiosk PC where the system drive resets to a clean
state on reboot, this makes the install vanish along with logins, settings,
and cache every restart. This project keeps both folders inside the portable
project root and links them in via NTFS junctions every time the launcher
runs, so a reboot does not reset Discord state.

## What you get

| File                    | Audience | Window                | Purpose                                                                      |
| ----------------------- | -------- | --------------------- | ---------------------------------------------------------------------------- |
| `Launch_Discord.exe`    | Clients  | None (GUI subsystem)  | Restores junctions + registry, wipes `DiscordData/`, launches Discord, exits.|
| `Discord_Updater.exe`   | Admin    | Console + progress    | Downloads DiscordSetup.exe and runs it through the portable junctions, every run. |

The two binaries are designed to be the only thing anyone runs.
`Launch_Discord.exe` is intentionally silent (no console, no prompts, no
update logic) so end users get a one-click experience.

## Folder layout (after first run)

```
discord-portable-go/
  Launch_Discord.exe      <- silent launcher (clients run this)
  Discord_Updater.exe     <- terminal updater (admin runs this)
  Discord/                <- portable install dir (junction target)
  DiscordData/            <- portable user data (login, settings, cache)
  _Installer/             <- cached DiscordSetup.exe
  _Reg/                   <- exported HKCU\SOFTWARE\Classes\Discord
  _state/                 <- last_update.txt timestamp
  _src/                   <- Go source (common pkg + cmd/launcher + cmd/updater)
  build.bat               <- rebuild both binaries
```

Only the two `.exe` files, `_src/`, `build.bat`, `LICENSE`, `.gitignore`, and
this `README.md` are tracked in git. Everything else is generated at first
run and ignored by `.gitignore`.

## Junctions used

| Symlink                  | Points to            | What lives there                    |
| ------------------------ | -------------------- | ----------------------------------- |
| `%LOCALAPPDATA%\Discord` | `<root>\Discord`     | Install + `Update.exe` (Squirrel)   |
| `%APPDATA%\discord`      | `<root>\DiscordData` | Login token, settings, cache, logs  |

Both junctions are rebuilt by every launch, so a wiped C: drive on the next
reboot is fine.

## Quick start

### 1. First-time setup (admin, once)

1. Drop `discord-portable-go/` onto your portable volume.
2. Double-click `Discord_Updater.exe`.
3. Wait for the progress bar. It will:
   - create `Discord/` and `DiscordData/` if missing,
   - replace `%LOCALAPPDATA%\Discord` with a junction to `Discord/` and
     `%APPDATA%\discord` with a junction to `DiscordData/`
     (migrating any existing data first if the portable folders are empty),
   - download `DiscordSetup.exe`,
   - run the installer silently; because the junctions are already in
     place, Squirrel writes straight into `Discord/` and `DiscordData/`,
   - kill the auto-launched Discord so the install dir unlocks,
   - export `HKCU\SOFTWARE\Classes\Discord` to `_Reg\Discord.reg`,
   - delete Discord's autostart `Run` entry.

When it closes, the portable install is ready.

### 2. Daily client use

Users double-click `Launch_Discord.exe`. Nothing else.

No console window appears. The launcher kills any running Discord, rebuilds
both junctions, **wipes `DiscordData/` so every session starts clean (no
login, no settings, no cache)**, imports the registry, starts Discord, and
exits.

### 3. Periodic updates (admin)

Run `Discord_Updater.exe` again on the admin schedule of your choice (manual,
Task Scheduler, on logon, whatever). The flow is the same as first-time
setup:

1. Stop any running Discord.
2. Ensure both junctions still exist (`%LOCALAPPDATA%\Discord` -> `Discord\`,
   `%APPDATA%\discord` -> `DiscordData\`).
3. Download a fresh `DiscordSetup.exe` with a progress bar.
4. Run the silent installer. Because the junctions are already in place,
   every file Squirrel writes to `%LOCALAPPDATA%\Discord` lands in the
   portable `Discord\` folder.
5. Kill the installer-launched Discord so the install dir unlocks.
6. Refresh the registry export and disable the autostart entry.

That's the entire update workflow. The terminal closes with the updater.

## Command-line behaviour

Both binaries take zero arguments. Behaviour is fully driven by which
binary you run and the state of the project root.

`Discord_Updater.exe`:

- Always: kill Discord, ensure junctions, download `DiscordSetup.exe`,
  run it silently, kill the installer-launched Discord, refresh the
  registry export. Same flow on first install and on every later run.

`Launch_Discord.exe`:

- If `Discord/Update.exe` is missing -> exit 1 silently (admin must run the
  updater first).
- Otherwise -> kill any running Discord, ensure junctions, wipe
  `DiscordData/` for a clean session, import reg, start Discord, exit.

## Building from source

Requirements: Go 1.21+ and the `rsrc` resource tool.

```bat
go install github.com/akavel/rsrc@latest
build.bat
```

`build.bat`:

1. Copies `_src/app.ico` into both `cmd/*` directories.
2. Generates `rsrc_amd64.syso` (icon + manifest) for each binary.
3. Builds `Launch_Discord.exe` with `-H windowsgui` (no console).
4. Builds `Discord_Updater.exe` as a console app.

Output binaries land in the project root. The launcher is ~3 MB, the updater
is ~5 MB.

## Source map

```
_src/
  go.mod
  app.ico                       canonical icon source
  common/
    paths.go                    portable path resolution + AppData targets
    console.go                  AllocConsole + ANSI VT for the GUI updater
    progress.go                 \r progress bar with smoothed speed + m:ss countdown formatter
    winutil.go                  taskkill, mklink /J, reg import/export, EnsureJunction with empty-target migration
    download.go                 HTTP download with progress + silent installer runner
  cmd/
    launcher/main.go            silent client launcher
    launcher/app.manifest       Win32 manifest (DPI / UTF-8 / asInvoker)
    updater/main.go             unified install/update flow (download DiscordSetup.exe + run via junction)
    updater/app.manifest        Win32 manifest
```

## Notes

- The launcher is GUI subsystem so antivirus / SmartScreen sees a clean
  windowless executable, no flashing console. Sign it if you ship widely.
- Junctions need NTFS. They do not work on FAT32 / exFAT volumes.
- The launcher does not require admin. Junctions are created with `mklink /J`
  which works for the current user.
- The updater downloads and runs `DiscordSetup.exe -s` on every
  invocation rather than relying on Discord's in-app Squirrel updater.
  The portable junctions (`%LOCALAPPDATA%\Discord` -> `Discord\`,
  `%APPDATA%\discord` -> `DiscordData\`) make the installer write
  straight into the portable folders, so no manual relocation step is
  needed.
- First run migrates anything currently in `%APPDATA%\discord` into
  `DiscordData/` if `DiscordData/` is empty, so an existing logged-in user
  does not have to log in again on the portable copy.

## License

MIT. See `LICENSE`.

## Credits

Inspired by an earlier batch + Python loader by fahmiyufrizal@2024. This Go
rewrite preserves the same workflow as two specialized executables and adds
data-folder symlinking so logins persist on virtual-disk-reset kiosks.
