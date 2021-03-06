package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
	"github.com/mattn/go-shellwords"
	"github.com/nsf/termbox-go"
)

const (
	// Name is the application name
	Name = "txtmanip"
	// Version is the application version
	Version = "0.2.1"
)

// Exit codes are int values that represent an exit code for a particular error.
const (
	ExitCodeOK    = 0
	ExitCodeError = 10 + iota
)

// Positions are y-coordinate for areas and line
const (
	InputAreaPos = iota
	InputErrorPos
	BorderLinePos
	TextAreaPos
)

// Color Settings for text
const (
	ColBg  = termbox.ColorDefault
	ColFg  = termbox.ColorWhite
	ColErr = termbox.ColorRed
)

// MainView represent main view
type MainView struct {
	textArea  TextArea
	inputArea InputArea
	height    int
	width     int
}

// Flush invokes termbox.Flush() after updates back buffers and set cursor
func (v *MainView) Flush() error {
	if err := termbox.Clear(ColBg, ColBg); err != nil {
		return err
	}

	termbox.SetCursor(v.inputArea.cursorPos, InputAreaPos)
	v.DrawBorderLine()
	v.DrawInputArea()
	v.DrawInputError()
	v.DrawTextArea()

	return termbox.Flush()
}

// DrawBorderLine draws line between input area and text area
func (v *MainView) DrawBorderLine() {
	for x := 0; x < v.width; x++ {
		termbox.SetCell(x, BorderLinePos, rune('-'), ColFg, ColBg)
	}
}

// DrawInputArea updates back buffer for input area
func (v *MainView) DrawInputArea() {
	v.inputArea.drawText(v.width, v.height)
}

// DrawInputError updates back buffer for input error area
func (v *MainView) DrawInputError() {
	v.inputArea.drawError()
}

// DrawTextArea updates back buffer for text area
func (v *MainView) DrawTextArea() {
	v.textArea.drawText()
}

// InputText adds byte by input
func (v *MainView) InputText(ch rune) {
	v.inputArea.input(ch)
}

// DeleteInputText deletes byte by input
func (v *MainView) DeleteInputText() {
	v.inputArea.delete()
}

// InputError sets error message
func (v *MainView) InputError(m string) {
	v.inputArea.error = []byte(m)
}

// InitCursor sets cursor position to initial one
func (v *MainView) InitCursor() {
	v.inputArea.initCursor()
}

// EndCursor sets cursor position to end of input text
func (v *MainView) EndCursor() {
	v.inputArea.endCursor()
}

// ForwardCursor forward cursor position
func (v *MainView) ForwardCursor(ch rune) {
	v.inputArea.forwardCursor(ch)
}

// ForwardOneRuneCursor forward cursor position one rune
func (v *MainView) ForwardOneRuneCursor() {
	v.inputArea.forwardOneRuneCursor()
}

// BackwardCursor backward cursor position
func (v *MainView) BackwardCursor() {
	v.inputArea.backwardCursor()
}

// SaveInvokeCommand saves invoked commands as list
func (v *MainView) SaveInvokeCommand() {
	v.inputArea.saveInvokeCommand()
}

// RedoInvokeCommands reverts invoked command list to preview version
func (v *MainView) RedoInvokeCommands() {
	v.inputArea.redoInvokeCommands()
}

// SaveInputHistory saves invoked commands as history list
func (v *MainView) SaveInputHistory() {
	v.inputArea.saveHistory()
}

// DrawInputHistory updates back buffer for input area with history
func (v *MainView) DrawInputHistory() {
	v.inputArea.drawHistory()
}

// ForwardInputHistory forward input history index
func (v *MainView) ForwardInputHistory() {
	v.inputArea.forwardHistoryIndex()
}

// BackwardInputHistory backward input history index
func (v *MainView) BackwardInputHistory() {
	v.inputArea.backwardHistoryIndex()
}

// ClearInputText clears content on input text
func (v *MainView) ClearInputText() {
	v.inputArea.clear()
}

// SetText sets content on text area
func (v *MainView) SetText(out *[]byte) {
	v.textArea.setText(out)
}

// RedoText reverts content of text area to preview version
func (v *MainView) RedoText() {
	v.textArea.redo()
}

// SaveTextHistory saves manipulated text history as list
func (v *MainView) SaveTextHistory() {
	v.textArea.saveHistory()
}

// InputArea represent input area
type InputArea struct {
	text             []byte
	error            []byte
	cursorPos        int
	cursorInitialPos int
	cursorByteOffset int
	prompt           []byte
	history          []string
	historyPos       int
	invokeCommands   []string
}

func (i *InputArea) cursorOffset() int {
	return i.cursorPos - i.cursorInitialPos
}

func (i *InputArea) input(ch rune) {
	var buf [utf8.UTFMax]byte
	n := utf8.EncodeRune(buf[:], ch)

	if i.cursorOffset() < runewidth.StringWidth(string(i.text)) {
		i.text = append(i.text[:i.cursorByteOffset], append(buf[:n], i.text[i.cursorByteOffset:]...)...)
		return
	}

	i.text = append(i.text, buf[:n]...)
}

func (i *InputArea) initCursor() {
	i.cursorPos = i.cursorInitialPos
	i.cursorByteOffset = 0
}

func (i *InputArea) endCursor() {
	if i.cursorOffset() == len(i.text) {
		return
	}
	i.cursorPos = i.cursorInitialPos + runewidth.StringWidth(string(i.text))
	i.cursorByteOffset = len(i.text)
}

func (i *InputArea) forwardCursor(ch rune) {
	if i.cursorOffset() == runewidth.StringWidth(string(i.text)) {
		return
	}

	i.cursorPos += runewidth.RuneWidth(ch)
	i.cursorByteOffset += utf8.RuneLen(ch)
}

func (i *InputArea) forwardOneRuneCursor() {
	if i.cursorOffset() == runewidth.StringWidth(string(i.text)) {
		return
	}

	_, size := utf8.DecodeRune(i.text[i.cursorByteOffset:])
	if size > 1 {
		i.cursorPos++
	}
	i.cursorPos++
	i.cursorByteOffset += size
}

func (i *InputArea) backwardCursor() {
	if i.cursorPos == i.cursorInitialPos {
		return
	}

	_, size := utf8.DecodeLastRune(i.text[:i.cursorByteOffset])
	if size > 1 {
		i.cursorPos--
	}
	i.cursorPos--
	i.cursorByteOffset -= size
}

func (i *InputArea) saveInvokeCommand() {
	i.invokeCommands = append(i.invokeCommands, string(i.text))
}

func (i *InputArea) saveHistory() {
	i.history = append(i.history, string(i.text))
	i.historyPos = len(i.history)
}

func (i *InputArea) forwardHistoryIndex() {
	if i.historyPos == len(i.history) {
		return
	}

	i.historyPos++
}

func (i *InputArea) backwardHistoryIndex() {
	if i.historyPos == 0 {
		return
	}

	i.historyPos--
}

func (i *InputArea) drawText(width, hight int) {
	for x, t := range i.prompt {
		termbox.SetCell(x, InputAreaPos, rune(t), ColFg, ColBg)
	}

	if len(i.text) < 1 {
		return
	}

	var x int
	for _, c := range string(i.text) {
		termbox.SetCell(i.cursorInitialPos+x, InputAreaPos, c, ColFg, ColBg)
		x += runewidth.RuneWidth(c)
	}

	for x := x; x < width; x++ {
		termbox.SetCell(i.cursorInitialPos+x, InputAreaPos, rune(' '), ColFg, ColBg)
	}
}

func (i *InputArea) drawHistory() {
	if i.historyPos == len(i.history) {
		i.clear()
		return
	}

	i.text = []byte(i.history[i.historyPos])
}

func (i *InputArea) clear() {
	i.initCursor()
	i.text = []byte("")
}

func (i *InputArea) drawError() {
	if len(i.error) < 1 {
		return
	}

	var x int
	for _, t := range string(i.error) {
		termbox.SetCell(x, InputErrorPos, t, ColErr, ColBg)
		x += runewidth.RuneWidth(t)
	}
	i.error = []byte("")
}

func (i *InputArea) redoInvokeCommands() {
	i.invokeCommands = i.invokeCommands[:len(i.invokeCommands)-1]
}

func (i *InputArea) delete() {
	if len(i.text) < 1 {
		return
	}

	_, size := utf8.DecodeRune(i.text[i.cursorByteOffset:])
	i.text = append(i.text[:i.cursorByteOffset], i.text[i.cursorByteOffset+size:]...)
}

// TextArea represent text area
type TextArea struct {
	text    []byte
	history []string
}

func (t *TextArea) setText(out *[]byte) {
	t.text = *out
}

func (t *TextArea) drawText() {
	y := TextAreaPos
	x := 0
	for _, t := range string(t.text) {
		if t == '\n' {
			y++
			x = 0
			continue
		}
		termbox.SetCell(x, y, t, ColFg, ColBg)
		x += runewidth.RuneWidth(t)
	}
}

func (t *TextArea) redo() {
	t.text = []byte(t.history[len(t.history)-1])
	t.history = t.history[:len(t.history)-1]
}

func (t *TextArea) saveHistory() {
	t.history = append(t.history, string(t.text))
}

func main() {
	os.Exit(_main())
}

func _main() int {
	var (
		config  string
		version bool
	)

	flags := flag.NewFlagSet(Name, flag.ContinueOnError)
	flags.Usage = usage
	flags.StringVar(&config, "c", "txtmanip.toml", "")
	flags.StringVar(&config, "config", "txtmanip.toml", "")
	flags.BoolVar(&version, "version", false, "")
	if err := flags.Parse(os.Args[1:]); err != nil {
		return ExitCodeError
	}

	if version {
		fmt.Printf("%s version %s\n", Name, Version)
		return ExitCodeOK
	}
	var src *os.File
	var f string

	if len(flags.Args()) < 1 {
		src = os.Stdin
	} else {
		f = flags.Arg(0)
		if _, err := os.Stat(f); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "%s is not exist: %s\n", f, err.Error())
			return ExitCodeError
		}
		file, err := os.Open(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Open file failed: %s\n", err.Error())
			return ExitCodeError
		}
		src = file
	}

	fi, err := src.Stat()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Get file stat failed: %s\n", err.Error())
		return ExitCodeError
	}

	if fi.Size() < 1 {
		fmt.Fprintf(os.Stderr, "Missing input\n")
		return ExitCodeError
	}

	text, err := ioutil.ReadAll(src)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Reading from src failed: %s\n", err.Error())
		return ExitCodeError
	}

	enableCommands, err := GetEnableCommands(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Read config failed: %s\n", err.Error())
		return ExitCodeError
	}

	invokeCommandsCh := make(chan []string)
	errCh := make(chan error)

	go func() {
		if err := termbox.Init(); err != nil {
			errCh <- errors.New(fmt.Sprint("initialize failed: ", err.Error()))
			return
		}

		w, h := termbox.Size()
		prompt := []byte(Name + "> ")
		view := &MainView{
			textArea: TextArea{
				text: text,
			},
			inputArea: InputArea{
				cursorInitialPos: len(prompt),
				prompt:           prompt,
			},
			width:  w,
			height: h,
		}
		defer func() {
			termbox.Close()
			invokeCommandsCh <- view.inputArea.invokeCommands
		}()

		termbox.SetInputMode(termbox.InputEsc)
		view.InitCursor()

	mainloop:
		for {
			view.Flush()

			switch ev := termbox.PollEvent(); ev.Type {
			case termbox.EventKey:
				switch ev.Key {
				case termbox.KeyEsc, termbox.KeyCtrlC:
					break mainloop
				case termbox.KeyCtrlA:
					view.InitCursor()
				case termbox.KeyCtrlE:
					view.EndCursor()
				case termbox.KeyArrowLeft, termbox.KeyCtrlB:
					view.BackwardCursor()
				case termbox.KeyArrowRight, termbox.KeyCtrlF:
					view.ForwardOneRuneCursor()
				case termbox.KeyArrowUp:
					view.BackwardInputHistory()
					view.DrawInputHistory()
				case termbox.KeyArrowDown:
					view.ForwardInputHistory()
					view.DrawInputHistory()
				case termbox.KeySpace:
					view.InputText(rune(' '))
					view.ForwardCursor(rune(' '))
				case termbox.KeyCtrlZ:
					if len(view.textArea.history) < 1 {
						continue
					}
					view.RedoText()
					view.RedoInvokeCommands()
				case termbox.KeyBackspace, termbox.KeyBackspace2:
					view.BackwardCursor()
					view.DeleteInputText()
				case termbox.KeyDelete, termbox.KeyCtrlD:
					view.DeleteInputText()
				case termbox.KeyEnter:
					if len(view.inputArea.text) < 1 {
						continue
					}

					args, err := shellwords.Parse(string(view.inputArea.text))
					if err != nil {
						errCh <- errors.New(fmt.Sprint("parse command failed: ", err.Error()))
						return
					}

					baseCommand, opts := args[0], args[1:]

					var enabled bool
					for _, c := range enableCommands {
						if baseCommand == c {
							enabled = true
						}
					}
					if !enabled {
						view.ClearInputText()
						view.InputError(fmt.Sprint(baseCommand, " cannot be executed"))
						continue
					}

					cmd := exec.Command(baseCommand, opts...)
					cmd.Stdin = bufio.NewReader(bytes.NewBuffer(view.textArea.text))

					out, err := cmd.Output()
					if err != nil {
						var isNotError bool
						if exitErr, ok := err.(*exec.ExitError); ok {
							// Workaround:
							// "grep" exits with return status 1 when no lines matched.
							// In this case is not error and I want to avoid deal with error it case.
							if baseCommand == "grep" {
								if ws, ok := exitErr.ProcessState.Sys().(syscall.WaitStatus); ok && ws.ExitStatus() == 1 {
									isNotError = true
								}
							}
							view.InputError(string(exitErr.Stderr))
						} else {
							view.InputError(err.Error())
						}

						if !isNotError {
							view.ClearInputText()
							continue
						}
					}

					view.SaveTextHistory()
					view.SetText(&out)
					view.SaveInputHistory()
					view.SaveInvokeCommand()
					view.ClearInputText()
				default:
					if ev.Ch != 0 {
						view.InputText(ev.Ch)
						view.ForwardCursor(ev.Ch)
					}
				}
			}
		}
	}()

	select {
	case err := <-errCh:
		fmt.Fprintf(os.Stderr, err.Error())
		return ExitCodeError
	case invokeCommands := <-invokeCommandsCh:
		var base string
		if f == "" {
			base = "<source>"
		} else {
			base = fmt.Sprintf("cat %s", f)
		}
		fmt.Println(strings.Join(append([]string{base}, invokeCommands...), " | "))
		return ExitCodeOK
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `Usage: textmanip [options] [FILE]

  txtmanip is a tool for text manipulation in interactive console with os commands.

  Run the txtmanip, starts interactive mode and you can text manipulation. 
  The initial output content is either of a file specified by arguments or standard input.

  After quit, prints one-liner of generating the same output for your made final result in interactive mode.

Options:
  -config, -c    Set configuration file path (default "txtmanip.toml")

Commands in interactive mode:
  Ctrl+C, Esc    Quit interactive mode
  Ctrl+Z         Redo text
  Up, Down       Print history  
`)
}
