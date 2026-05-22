package common

import (
	"fmt"
	"io"
	"strings"
	"time"
)

// Unit selects how Current/Total are rendered in the progress bar.
type Unit int

const (
	// UnitBytes renders Current/Total as "1.2 MiB / 5.0 MiB @ 800 KiB/s".
	UnitBytes Unit = iota
	// UnitSeconds renders Current/Total as "0:42 / 2:00" with no rate
	// suffix. Used for fixed-time countdowns where Current is ticked
	// once per second.
	UnitSeconds
)

// ProgressWriter renders a single-line progress bar to stdout.
type ProgressWriter struct {
	Total   int64
	Current int64
	Label   string
	Width   int
	Unit    Unit

	startedAt  time.Time
	lastDraw   time.Time
	lastSpeedT time.Time
	lastBytes  int64
	speed      float64
}

func NewProgress(label string, total int64) *ProgressWriter {
	now := time.Now()
	return &ProgressWriter{
		Total: total, Label: label, Width: 30, Unit: UnitBytes,
		startedAt: now, lastSpeedT: now,
	}
}

// NewCountdown returns a progress writer pre-configured for a
// fixed-duration countdown (Current is ticked in seconds).
func NewCountdown(label string, seconds int64) *ProgressWriter {
	now := time.Now()
	return &ProgressWriter{
		Total: seconds, Label: label, Width: 30, Unit: UnitSeconds,
		startedAt: now, lastSpeedT: now,
	}
}

func (p *ProgressWriter) Write(b []byte) (int, error) {
	n := len(b)
	p.Current += int64(n)
	p.draw(false)
	return n, nil
}

func (p *ProgressWriter) Draw(force bool) { p.draw(force) }

func (p *ProgressWriter) draw(force bool) {
	now := time.Now()
	if !force && now.Sub(p.lastDraw) < 100*time.Millisecond {
		return
	}
	p.lastDraw = now

	if elapsed := now.Sub(p.lastSpeedT).Seconds(); elapsed > 0.5 {
		p.speed = float64(p.Current-p.lastBytes) / elapsed
		p.lastBytes = p.Current
		p.lastSpeedT = now
	}

	var pct float64
	if p.Total > 0 {
		pct = float64(p.Current) / float64(p.Total)
		if pct > 1 {
			pct = 1
		}
	}
	filled := int(pct * float64(p.Width))
	if filled > p.Width {
		filled = p.Width
	}
	bar := strings.Repeat("=", filled) + strings.Repeat(" ", p.Width-filled)

	switch p.Unit {
	case UnitSeconds:
		fmt.Printf("\r  %s [%s] %5.1f%%  %s / %s     ",
			p.Label, bar, pct*100,
			formatDuration(p.Current), formatDuration(p.Total))
	default:
		if p.Total > 0 {
			fmt.Printf("\r  %s [%s] %5.1f%%  %s / %s  @ %s/s     ",
				p.Label, bar, pct*100,
				HumanBytes(p.Current), HumanBytes(p.Total),
				HumanBytes(int64(p.speed)))
		} else {
			fmt.Printf("\r  %s  %s  @ %s/s     ",
				p.Label, HumanBytes(p.Current), HumanBytes(int64(p.speed)))
		}
	}
}

func (p *ProgressWriter) Done() {
	p.draw(true)
	fmt.Println()
}

// formatDuration renders a number of seconds as "m:ss" (or "h:mm:ss"
// when an hour or more).
func formatDuration(seconds int64) string {
	if seconds < 0 {
		seconds = 0
	}
	h := seconds / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

func HumanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for x := n / unit; x >= unit; x /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

// CountReader wraps a reader and pumps byte counts into a ProgressWriter.
type CountReader struct {
	R io.Reader
	P *ProgressWriter
}

func (c *CountReader) Read(b []byte) (int, error) {
	n, err := c.R.Read(b)
	c.P.Current += int64(n)
	c.P.draw(false)
	return n, err
}
