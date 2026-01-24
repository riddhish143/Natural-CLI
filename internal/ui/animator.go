package ui

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"
)

const (
	hideCursor = "\033[?25l"
	showCursor = "\033[?25h"
	clearLine  = "\033[K"
	reset      = "\033[0m"
)

// AnimatedSpinner shows an animated spinner
type AnimatedSpinner struct {
	frames  []string
	message string
	done    chan struct{}
	stopped bool
	mutex   sync.Mutex
}

func NewSpinner(message string) *AnimatedSpinner {
	return &AnimatedSpinner{
		frames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		message: message,
		done:    make(chan struct{}),
	}
}

func (s *AnimatedSpinner) Start() {
	s.mutex.Lock()
	if s.stopped {
		s.mutex.Unlock()
		return
	}
	s.mutex.Unlock()

	go func() {
		if !term.IsTerminal(int(os.Stdout.Fd())) {
			fmt.Printf("  %s\n", s.message)
			return
		}

		fmt.Print(hideCursor)
		i := 0

		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-s.done:
				fmt.Print("\r" + clearLine)
				fmt.Print(showCursor)
				return
			case <-ticker.C:
				frame := s.frames[i%len(s.frames)]
				fmt.Printf("\r  \033[36m%s\033[0m %s", frame, s.message)
				i++
			}
		}
	}()
}

func (s *AnimatedSpinner) Stop() {
	s.mutex.Lock()
	if s.stopped {
		s.mutex.Unlock()
		return
	}
	s.stopped = true
	s.mutex.Unlock()

	close(s.done)
	
	// Give goroutine time to clean up
	time.Sleep(120 * time.Millisecond)
	
	// Ensure line is cleared
	fmt.Print("\r" + clearLine)
}

// wrapText wraps text to fit within a specified width
func wrapText(text string, width int) []string {
	var lines []string

	paragraphs := strings.Split(text, "\n")

	for _, para := range paragraphs {
		if para == "" {
			lines = append(lines, "")
			continue
		}

		words := strings.Fields(para)
		var currentLine string

		for _, word := range words {
			if len(currentLine)+len(word)+1 > width {
				if currentLine != "" {
					lines = append(lines, currentLine)
				}
				currentLine = word
			} else {
				if currentLine != "" {
					currentLine += " "
				}
				currentLine += word
			}
		}

		if currentLine != "" {
			lines = append(lines, currentLine)
		}
	}

	return lines
}
