// Launch_Discord.exe
// Fully silent, GUI-subsystem launcher intended for client machines.
// On every run it:
//   1. ensures %LOCALAPPDATA%\Discord  -> <root>\Discord       (junction)
//   2. ensures %APPDATA%\discord       -> <root>\DiscordData   (junction)
//   3. imports _Reg\Discord.reg if present
//   4. launches Discord via Update.exe and exits
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

	// Restore portable links every launch in case the disk was reset.
	_ = common.EnsureJunction(p.DiscordDir, p.LocalAppDiscord)
	_ = common.EnsureJunction(p.DataDir, p.RoamingDiscord)

	common.EnsureRegistry(p)

	if err := common.LaunchDiscord(p); err != nil {
		os.Exit(2)
	}
}
