// Launch_Discord.exe
// Fully silent, GUI-subsystem launcher intended for client machines.
// On every run it:
//   1. kills any running Discord so its files unlock
//   2. ensures %LOCALAPPDATA%\Discord  -> <root>\Discord       (junction)
//   3. ensures %APPDATA%\discord       -> <root>\DiscordData   (junction)
//   4. wipes <root>\DiscordData so every session starts logged-out / blank
//   5. imports _Reg\Discord.reg if present
//   6. launches Discord via Update.exe and exits
//
// No console window, no prompts, no update logic. Use Discord_Updater.exe
// to perform updates ahead of time on the server side.

package main

import (
	"discord-portable/common"
	"os"
)

func main() {
	p := common.Resolve()

	// Without Update.exe there is nothing we can launch. Refuse silently
	// instead of throwing UI at a kiosk client.
	if !common.FileExists(p.UpdateExe) {
		os.Exit(1)
	}

	// Free any locks on DiscordData before we wipe it. Safe no-op if
	// nothing is running.
	common.KillDiscord()

	// Restore portable links every launch in case the disk was reset.
	// Done before the wipe so EnsureJunction's "migrate when target is
	// empty" path cannot repopulate DiscordData from %APPDATA%\discord
	// after we clear it.
	_ = common.EnsureJunction(p.DiscordDir, p.LocalAppDiscord)
	_ = common.EnsureJunction(p.DataDir, p.RoamingDiscord)

	// Mandatory clean session: wipe all user data (login token, settings,
	// cache, logs) so every Discord launch starts blank.
	_ = common.CleanDir(p.DataDir)

	common.EnsureRegistry(p)

	if err := common.LaunchDiscord(p); err != nil {
		os.Exit(2)
	}
}
