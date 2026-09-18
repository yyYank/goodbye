package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/yyYank/goodbye/internal/uninstall"
)

// stty is used on the application's supported Unix terminals to avoid adding a
// terminal library dependency. Input returns to canonical mode before confirmation.
func selectUninstallTUI(items []uninstall.Item, in *bufio.Reader, input io.Reader, out io.Writer) (selected []uninstall.Item, err error) {
	rows, columns := 10, 80
	if file, ok := input.(*os.File); ok {
		stty := func(args ...string) ([]byte, error) {
			c := exec.Command("stty", args...)
			c.Stdin = file
			return c.Output()
		}
		state, stateErr := stty("-g")
		if stateErr != nil {
			return nil, fmt.Errorf("--tui requires a terminal with stty: %w", stateErr)
		}
		if size, sizeErr := stty("size"); sizeErr == nil {
			var height, width int
			if _, sizeErr = fmt.Sscan(string(size), &height, &width); sizeErr == nil && height >= 6 && width >= 20 {
				rows, columns = height-4, width
			}
		}
		// Defer before changing modes so even a partially failed stty is restored.
		defer func() {
			if _, restoreErr := stty(strings.TrimSpace(string(state))); restoreErr != nil && err == nil {
				err = fmt.Errorf("restore terminal: %w", restoreErr)
			}
		}()
		if _, err = stty("raw", "-echo"); err != nil {
			return nil, fmt.Errorf("prepare terminal: %w", err)
		}
	}
	return selectCursorItems(items, in, out, rows, columns)
}

func selectCursorItems(items []uninstall.Item, in *bufio.Reader, out io.Writer, rows, columns int) ([]uninstall.Item, error) {
	if len(items) == 0 {
		return nil, nil
	}
	if rows < 1 {
		rows = 1
	}
	if columns < 20 {
		columns = 20
	}
	if _, err := fmt.Fprint(out, "\x1b[?1049h\x1b[?25l"); err != nil {
		return nil, err
	}
	defer fmt.Fprint(out, "\x1b[?25h\x1b[?1049l")
	cursor := 0
	checked := make([]bool, len(items))
	for {
		var screen strings.Builder
		screen.WriteString("\x1b[H\x1b[2J")
		write := func(text string) {
			if len(text) > columns-1 {
				text = text[:columns-4] + "..."
			}
			screen.WriteString(text + "\r\n")
		}
		write("Select packages to uninstall")
		start := cursor / rows * rows
		for i := start; i < len(items) && i < start+rows; i++ {
			mark := "[ ]"
			if checked[i] {
				mark = "[x]"
			}
			pointer := " "
			if i == cursor {
				pointer = ">"
			}
			item := items[i]
			// Escape terminal controls and non-ASCII bytes to keep rows from wrapping.
			label := strconv.QuoteToASCII(item.Name + " " + item.Version + " " + item.Path)
			write(pointer + " " + mark + " " + label[1:len(label)-1])
		}
		write(fmt.Sprintf("%d/%d  arrows/jk: move  Space: toggle  h/l: off/on", cursor+1, len(items)))
		write("a: toggle all  Enter: confirm selection  q/Ctrl-C: cancel")
		if _, err := io.WriteString(out, screen.String()); err != nil {
			return nil, err
		}
		key, err := in.ReadByte()
		if err != nil {
			return nil, fmt.Errorf("read selection: %w", err)
		}
		if key == 27 {
			prefix, err := in.ReadByte()
			if err != nil {
				return nil, err
			}
			if prefix != '[' && prefix != 'O' {
				return nil, fmt.Errorf("unsupported terminal key sequence")
			}
			direction, err := in.ReadByte()
			if err != nil {
				return nil, err
			}
			switch direction {
			case 'A':
				key = 'k'
			case 'B':
				key = 'j'
			case 'C':
				key = 'l'
			case 'D':
				key = 'h'
			default:
				return nil, fmt.Errorf("unsupported terminal key sequence")
			}
		}
		switch key {
		case 'j':
			if cursor < len(items)-1 {
				cursor++
			}
		case 'k':
			if cursor > 0 {
				cursor--
			}
		case 'h':
			checked[cursor] = false
		case 'l':
			checked[cursor] = true
		case ' ':
			checked[cursor] = !checked[cursor]
		case 'a':
			all := true
			for _, value := range checked {
				all = all && value
			}
			for i := range checked {
				checked[i] = !all
			}
		case 'q', 3:
			return nil, nil
		case '\r', '\n':
			var result []uninstall.Item
			for i, item := range items {
				if checked[i] {
					result = append(result, item)
				}
			}
			return result, nil
		default:
			return nil, fmt.Errorf("unsupported selection key; use arrows, hjkl, Space, a, Enter or q")
		}
	}
}
