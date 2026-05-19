package common

import (
	"os"
	"path/filepath"
)

// Paths bundles every filesystem location the launcher and updater need.
// All portable paths are resolved relative to the executable's own directory
// so the whole folder stays movable across drives.
type Paths struct {
	Root         string // folder containing the exe
	DiscordDir   string // <root>\Discord       (install dir, has Update.exe)
	DataDir      string // <root>\DiscordData   (user data: login, settings)
	UpdateExe    string // <root>\Discord\Update.exe
	InstallerDir string // <root>\_Installer
	InstallerExe string // <root>\_Installer\DiscordSetup.exe
	RegDir       string // <root>\_Reg
	RegFile      string // <root>\_Reg\Discord.reg
	StateDir     string // <root>\_state
	LastUpdate   string // <root>\_state\last_update.txt
	ConfigFile   string // <root>\config.ini

	LocalAppDiscord string // %LOCALAPPDATA%\Discord (install junction target)
	RoamingDiscord  string // %APPDATA%\discord      (user data junction target)
}

func Resolve() Paths {
	exe, err := os.Executable()
	root := ""
	if err == nil {
		root = filepath.Dir(exe)
	} else {
		root, _ = os.Getwd()
	}
	p := Paths{Root: root}
	p.DiscordDir = filepath.Join(root, "Discord")
	p.DataDir = filepath.Join(root, "DiscordData")
	p.UpdateExe = filepath.Join(p.DiscordDir, "Update.exe")
	p.InstallerDir = filepath.Join(root, "_Installer")
	p.InstallerExe = filepath.Join(p.InstallerDir, "DiscordSetup.exe")
	p.RegDir = filepath.Join(root, "_Reg")
	p.RegFile = filepath.Join(p.RegDir, "Discord.reg")
	p.StateDir = filepath.Join(root, "_state")
	p.LastUpdate = filepath.Join(p.StateDir, "last_update.txt")
	p.ConfigFile = filepath.Join(root, "config.ini")

	p.LocalAppDiscord = filepath.Join(os.Getenv("LOCALAPPDATA"), "Discord")
	p.RoamingDiscord = filepath.Join(os.Getenv("APPDATA"), "discord")
	return p
}

func FileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
