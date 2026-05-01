package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"
)

const (
	progressDelay    = 800 * time.Millisecond
	progressInterval = 120 * time.Millisecond
	progressWidth    = 18
)

type progressIndicator struct {
	w        io.Writer
	label    string
	enabled  bool
	delay    time.Duration
	interval time.Duration

	done          chan struct{}
	stopped       chan struct{}
	once          sync.Once
	renderedWidth int
}

func newProgressIndicator(w io.Writer, label string, enabled bool) *progressIndicator {
	return &progressIndicator{
		w:        w,
		label:    strings.TrimSpace(label),
		enabled:  enabled,
		delay:    progressDelay,
		interval: progressInterval,
		done:     make(chan struct{}),
		stopped:  make(chan struct{}),
	}
}

func (p *progressIndicator) Start() {
	if p == nil || !p.enabled || p.w == nil {
		return
	}
	go p.loop()
}

func (p *progressIndicator) Stop() {
	if p == nil || !p.enabled || p.w == nil {
		return
	}
	p.once.Do(func() {
		close(p.done)
		<-p.stopped
	})
}

func (p *progressIndicator) loop() {
	defer close(p.stopped)

	timer := time.NewTimer(p.delay)
	defer timer.Stop()

	select {
	case <-p.done:
		return
	case <-timer.C:
	}

	shown := false
	started := time.Now()
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for frame := 0; ; frame++ {
		p.render(frame, time.Since(started))
		shown = true
		select {
		case <-p.done:
			if shown {
				p.clear()
			}
			return
		case <-ticker.C:
		}
	}
}

func (p *progressIndicator) render(frame int, elapsed time.Duration) {
	label := p.label
	if label == "" {
		label = "working"
	}
	line := fmt.Sprintf("%s %s %s", "working", progressBar(frame), progressElapsed(elapsed)+" "+label)
	p.renderedWidth = len(line)
	fmt.Fprintf(p.w, "\r%s", line)
}

func (p *progressIndicator) clear() {
	width := p.renderedWidth
	if width < 96 {
		width = 96
	}
	fmt.Fprintf(p.w, "\r%s\r", strings.Repeat(" ", width))
}

func progressBar(frame int) string {
	if progressWidth <= 0 {
		return "[]"
	}
	segment := 5
	if segment > progressWidth {
		segment = progressWidth
	}
	span := progressWidth - segment + 1
	start := 0
	if span > 0 {
		start = frame % span
	}
	var b strings.Builder
	b.WriteByte('[')
	for i := 0; i < progressWidth; i++ {
		if i >= start && i < start+segment {
			b.WriteByte('=')
		} else {
			b.WriteByte(' ')
		}
	}
	b.WriteByte(']')
	return b.String()
}

func progressElapsed(elapsed time.Duration) string {
	seconds := int(elapsed.Round(time.Second).Seconds())
	if seconds < 1 {
		seconds = 1
	}
	return fmt.Sprintf("%ds", seconds)
}

func shouldEnableProgress(w io.Writer) bool {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("GIG_PROGRESS")), "0") {
		return false
	}
	return writerIsTerminal(w)
}

func writerIsTerminal(w io.Writer) bool {
	if file, ok := w.(*os.File); ok {
		return fileIsTerminal(file)
	}
	return false
}

func fileIsTerminal(file *os.File) bool {
	return file != nil && term.IsTerminal(int(file.Fd()))
}

func (a *App) startProgress(label string) *progressIndicator {
	progress := newProgressIndicator(a.progressWriter, label, a.progressEnabled)
	progress.Start()
	return progress
}

func (a *App) runWithProgress(label string, run func() error) error {
	progress := a.startProgress(label)
	defer progress.Stop()
	return run()
}

func progressTicketLabel(command string, tickets []string) string {
	command = strings.TrimSpace(command)
	if command == "" {
		command = "gig"
	}
	switch len(tickets) {
	case 0:
		return command
	case 1:
		return command + " " + strings.TrimSpace(tickets[0])
	default:
		return fmt.Sprintf("%s %d tickets", command, len(tickets))
	}
}
