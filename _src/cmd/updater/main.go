// Discord_Updater.exe
// Server-side update tool with terminal progress.
//
// Behavior (single unified flow, runs the same on first install and
// every subsequent update):
//
//   1. Stop any running Discord so its files unlock.
//   2. Ensure %LOCALAPPDATA%\Discord -> <root>\Discord       junction.
//      Ensure %APPDATA%\discord      -> <root>\DiscordData   junction.
//      On first run, if a real Discord install already exists at
//      %LOCALAPPDATA%\Discord, EnsureJunction migrates it into the
//      portable folder before swapping in the junction.
//   3. Download a fresh DiscordSetup.exe with a progress bar.
//   4. Run the installer with `-s`. Because the junctions are in place,
//      the Squirrel installer writes straight into <root>\Discord, so
//      no manual relocation is needed. The installer auto-launches
//      Discord when it finishes.
//   5. Kill the installer-launched Discord so the install dir unlocks.
//   6. Re-export HKCU\SOFTWARE\Classes\Discord and scrub the autostart
//      Run entry the installer may have re-added.
//
// This replaces the older "open Discord and wait 120 s for Squirrel
// to self-update" approach, which was unreliable: Squirrel does not
// always run an update check on each launch, so most update cycles
// did nothing and just re-opened the app.

package main

import (
	"discord-portable/common"
	"fmt"
	"os"
	"time"
)

func main() {
	p := common.Resolve()

	fmt.Println("=========================================")
	fmt.Println("  Discord Portable - Updater")
	fmt.Println("=========================================")
	fmt.Println()

	if err := run(p); err != nil {
		die(err)
	}

	fmt.Println()
	fmt.Println("[+] Update complete. Closing in 5s...")
	time.Sleep(5 * time.Second)
}

func run(p common.Paths) error {
	// Make sure every portable folder we care about exists before any
	// step touches it.
	for _, d := range []string{p.DiscordDir, p.DataDir, p.StateDir, p.InstallerDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			return err
		}
	}

	fmt.Println("[1/6] Stopping any running Discord...")
	common.KillDiscord()
	time.Sleep(1 * time.Second)

	// Junctions go up before the installer runs so the installer's
	// writes to %LOCALAPPDATA%\Discord and %APPDATA%\discord land in
	// our portable folders. EnsureJunction handles three cases:
	//   - link missing               -> create junction
	//   - link is real folder, empty -> migrate then swap
	//   - link is already a junction -> leave alone
	fmt.Println("[2/6] Ensuring portable junctions...")
	if err := common.EnsureJunction(p.DiscordDir, p.LocalAppDiscord); err != nil {
		return fmt.Errorf("install junction: %w", err)
	}
	if err := common.EnsureJunction(p.DataDir, p.RoamingDiscord); err != nil {
		return fmt.Errorf("data junction: %w", err)
	}

	fmt.Println("[3/6] Downloading Discord installer...")
	if common.FileExists(p.InstallerExe) {
		_ = os.Remove(p.InstallerExe)
	}
	if err := common.DownloadFile(common.DiscordSetupURL, p.InstallerExe, "Download"); err != nil {
		return fmt.Errorf("download: %w", err)
	}

	// DiscordSetup.exe with `-s` is fully silent. It still spawns
	// Discord at the end (Squirrel's standard post-install hook), so
	// we kill that in the next step. WaitForLocalAppDiscord is a
	// safety net for the rare case where the installer returns before
	// Update.exe is fully on disk.
	fmt.Println("[4/6] Running installer (silent, can take 30-60s)...")
	_ = common.RunInstallerSilent(p.InstallerExe)
	if err := common.WaitForLocalAppDiscord(p, 120*time.Second); err != nil {
		return err
	}

	fmt.Println("[5/6] Stopping installer-launched Discord...")
	common.KillDiscord()
	time.Sleep(2 * time.Second)
	// Second sweep covers Squirrel children that respawned during the
	// first taskkill (Discord.exe, Update.exe, DiscordCrashHandler.exe
	// like to relaunch each other once on close).
	common.KillDiscord()

	fmt.Println("[6/6] Refreshing registry export and disabling autostart...")
	_ = common.RegExport(`HKEY_CURRENT_USER\SOFTWARE\Classes\Discord`, p.RegFile)
	_ = common.RegDeleteRun("Discord")

	common.WriteTimestamp(p.LastUpdate)

	if !common.FileExists(p.UpdateExe) {
		return fmt.Errorf("Update.exe missing after install at %s", p.UpdateExe)
	}
	return nil
}

func die(err error) {
	fmt.Println()
	fmt.Println("[!] Error:", err)
	fmt.Println("Closing in 10s...")
	time.Sleep(10 * time.Second)
	os.Exit(1)
}
