// Discord_Updater.exe
// Server-side update tool with terminal progress.
//
// One unified flow, run the same way every invocation (no more two-path
// "open Discord and wait 120s" mode):
//
//   1. Stop Discord.
//   2. Ensure %APPDATA%\discord -> <root>\DiscordData junction (data
//      side; idempotent, handles first-run migrate-when-empty).
//   3. Clear %LOCALAPPDATA%\Discord. RemoveAll special-cases reparse
//      points so this only drops the link and never touches portable
//      Discord/. Required because Squirrel's silent installer refuses
//      to write to %LOCALAPPDATA%\Discord if anything is already there.
//   4. Download a fresh DiscordSetup.exe with a progress bar.
//   5. Run `DiscordSetup.exe -s`. Squirrel installs fresh into the
//      now-clean real %LOCALAPPDATA%\Discord and auto-launches Discord
//      at the end; we kill that.
//   6. Replace portable Discord/ with the new install: CleanDir the
//      old contents, copy %LOCALAPPDATA%\Discord into Discord/, remove
//      %LOCALAPPDATA%\Discord, junction it back at portable Discord/.
//   7. Refresh the registry export and scrub Discord's autostart Run
//      entry.
//
// This replaces the older "open Discord and wait 120s for Squirrel to
// self-update" approach, which was unreliable: Squirrel's in-app
// updater rarely runs an update check on demand, so most update cycles
// did nothing and just re-opened the app.

package main

import (
	"discord-portable/common"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
	for _, d := range []string{p.DiscordDir, p.DataDir, p.StateDir, p.InstallerDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			return err
		}
	}

	fmt.Println("[1/6] Stopping any running Discord...")
	common.KillDiscord()
	time.Sleep(1 * time.Second)

	// Data junction goes up before the installer runs so any data the
	// post-install Discord auto-launch writes (boot prefs, dispatch
	// telemetry, etc.) lands in the portable DiscordData/ folder.
	// EnsureJunction is idempotent and also handles the first-ever-run
	// migrate-when-empty case for an existing system Discord login.
	if err := common.EnsureJunction(p.DataDir, p.RoamingDiscord); err != nil {
		return fmt.Errorf("data junction: %w", err)
	}

	// Clear %LOCALAPPDATA%\Discord. RemoveAll special-cases reparse
	// points (only removes the link, never the target), so portable
	// Discord/ is preserved if a junction is currently in place.
	// Squirrel's silent installer refuses to write here if anything
	// already exists, so this step is mandatory every run.
	fmt.Println("[2/6] Clearing %LOCALAPPDATA%\\Discord ...")
	if common.FileExists(p.LocalAppDiscord) {
		if err := common.RemoveAll(p.LocalAppDiscord); err != nil {
			return fmt.Errorf("clear LOCALAPPDATA\\Discord: %w", err)
		}
	}

	fmt.Println("[3/6] Downloading Discord installer...")
	if common.FileExists(p.InstallerExe) {
		_ = os.Remove(p.InstallerExe)
	}
	if err := common.DownloadFile(common.DiscordSetupURL, p.InstallerExe, "Download"); err != nil {
		return fmt.Errorf("download: %w", err)
	}

	fmt.Println("[4/6] Running installer (silent). This can take up to a minute...")
	_ = common.RunInstallerSilent(p.InstallerExe)
	if err := common.WaitForLocalAppDiscord(p, 120*time.Second); err != nil {
		return err
	}
	// Squirrel auto-launches Discord at the end of install. Kill it
	// before we move files around. Two passes because Update.exe and
	// Discord.exe like to respawn each other once on close.
	common.KillDiscord()
	time.Sleep(2 * time.Second)
	common.KillDiscord()

	fmt.Println("[5/6] Replacing portable Discord/ with new install...")
	// At this point %LOCALAPPDATA%\Discord is a real folder containing
	// the fresh install. Portable Discord/ may still hold the previous
	// install -- wipe it before relocating so we don't end up with a
	// merged old + new tree.
	if err := common.CleanDir(p.DiscordDir); err != nil {
		return fmt.Errorf("clean Discord/: %w", err)
	}
	if err := relocateInstall(p); err != nil {
		return err
	}

	fmt.Println("[6/6] Exporting registry and disabling autostart...")
	_ = common.RegExport(`HKEY_CURRENT_USER\SOFTWARE\Classes\Discord`, p.RegFile)
	_ = common.RegDeleteRun("Discord")

	common.WriteTimestamp(p.LastUpdate)
	if !common.FileExists(p.UpdateExe) {
		return fmt.Errorf("Update.exe missing after install at %s", p.UpdateExe)
	}
	return nil
}

// relocateInstall copies %LOCALAPPDATA%\Discord into <root>\Discord and
// replaces the original with a junction pointing back. Caller is
// responsible for ensuring portable Discord/ is empty first.
func relocateInstall(p common.Paths) error {
	src := p.LocalAppDiscord
	if !common.FileExists(src) {
		return fmt.Errorf("install source missing: %s", src)
	}

	var total int64
	_ = filepath.Walk(src, func(_ string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			total += info.Size()
		}
		return nil
	})
	pw := common.NewProgress("Copying ", total)

	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dst := filepath.Join(p.DiscordDir, rel)
		if info.IsDir() {
			return os.MkdirAll(dst, info.Mode())
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
		if err != nil {
			return err
		}
		defer out.Close()
		buf := make([]byte, 256*1024)
		for {
			n, rerr := in.Read(buf)
			if n > 0 {
				if _, werr := out.Write(buf[:n]); werr != nil {
					return werr
				}
				pw.Current += int64(n)
				pw.Draw(false)
			}
			if rerr != nil {
				if errors.Is(rerr, io.EOF) {
					return nil
				}
				return rerr
			}
		}
	})
	pw.Done()
	if err != nil {
		return err
	}

	if err := common.RemoveAll(src); err != nil {
		return err
	}
	return common.MakeJunction(p.DiscordDir, src)
}

func die(err error) {
	fmt.Println()
	fmt.Println("[!] Error:", err)
	fmt.Println("Closing in 10s...")
	time.Sleep(10 * time.Second)
	os.Exit(1)
}
