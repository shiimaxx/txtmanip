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
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
	"github.com/mattn/go-shellwords"
	"github.com/nsf/termbox-go"
)

const (
	Name    = "txtmanip"
	Version = "0.1.1"
)

const (
	ExitCodeOK    = 0
	ExitCodeError = 10 + iota
)

const (
	InputAreaPos = iota
	InputErrorPos
	BorderLinePos
	TextAreaPos
)

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

func (v *MainView) DrawBorderLine() {
	for x := 0; x < v.width; x++ {
		termbox.SetCell(x, BorderLinePos, rune('-'), ColFg, ColBg)
	}
}

func (v *MainView) DrawInputArea() {
	v.inputArea.drawText(v.width, v.height)
}

func (v *MainView) DrawInputError() {
	v.inputArea.drawError()
}

func (v *MainView) DrawTextArea() {
	v.textArea.drawText()
}

func (v *MainView) InputText(ch rune) int {
	return v.inputArea.input(ch)
}

func (v *MainView) DeleteInputText() {
	v.inputArea.delete()
}

func (v *MainView) InputError(m string) {
	v.inputArea.error = []byte(m)
}

func (v *MainView) InitCursor() {
	v.inputArea.initCursor()
}

func (v *MainView) EndCursor() {
	v.inputArea.endCursor()
}

func (v *MainView) ForwardCursor(offset int) {
	v.inputArea.forwardCursor(offset)
}

func (v *MainView) BackwardCursor() {
	v.inputArea.backwardCursor()
}

func (v *MainView) SaveInvokeCommand() {
	v.inputArea.saveInvokeCommand()
}

func (v *MainView) RedoInvokeCommands() {
	v.inputArea.redoInvokeCommands()
}

func (v *MainView) SaveInputHistory() {
	v.inputArea.saveHistory()
}

func (v *MainView) DrawInputHistory() {
	v.inputArea.drawHistory()
}

func (v *MainView) ForwardInputHisotry() {
	v.inputArea.forwardHistoryIndex()
}

func (v *MainView) BackwardInputHisotry() {
	v.inputArea.backwardHistoryIndex()
}

func (v *MainView) ClearInputText() {
	v.inputArea.clear()
}

func (v *MainView) SetText(out *[]byte) {
	v.textArea.setText(out)
}

func (v *MainView) RedoText() {
	v.textArea.redo()
}

func (v *MainView) SaveTextHistory() {
	v.textArea.saveHistory()
}

// InputArea represent input area
type InputArea struct {
	text             []byte
	error            []byte
	cursorPos        int
	cursorInitialPos int
	prompt           []byte
	history          []string
	historyPos       int
	invokeCommands   []string
}

func (i *InputArea) cursorOffset() int {
	return i.cursorPos - i.cursorInitialPos
}

func (i *InputArea) input(ch rune) int {
	var buf [utf8.UTFMax]byte
	n := utf8.EncodeRune(buf[:], ch)

	//var runeWidth int
	if i.cursorOffset() < runewidth.StringWidth(string(i.text)) {
		_, size := utf8.DecodeLastRune(i.text[i.cursorOffset():])
		if size > 1 {
			i.text = append(i.text[:i.cursorOffset()+(size-1)], append(buf[:n], i.text[i.cursorOffset()+(size-1):]...)...)
			//runeWidth = 2
		} else {
			i.text = append(i.text[:i.cursorOffset()], append(buf[:n], i.text[i.cursorOffset():]...)...)
			//runeWidth = 1
		}
		return runewidth.RuneWidth(ch)
	}

	i.text = append(i.text, buf[:n]...)
	return runewidth.RuneWidth(ch)
}

func (i *InputArea) initCursor() {
	i.cursorPos = i.cursorInitialPos
}

func (i *InputArea) endCursor() {
	if i.cursorOffset() == len(i.text) {
		return
	}
	i.cursorPos = i.cursorInitialPos + len(i.text)
}

func (i *InputArea) forwardCursor(offset int) {
	if i.cursorOffset() == runewidth.StringWidth(string(i.text)) {
		return
	}

	if offset > 0 {
		i.cursorPos += offset
		return
	}

	_, size := utf8.DecodeLastRune(i.text[i.cursorOffset():])
	if size > 1 {
		i.cursorPos++
	}
	i.cursorPos++
}

func (i *InputArea) backwardCursor() {
	if i.cursorPos == i.cursorInitialPos {
		return
	}

	offset := utf8.UTFMax
	for i.cursorOffset()-offset < 0 {
		offset--
	}

	_, size := utf8.DecodeLastRune(i.text[i.cursorOffset()-offset:])
	if size > 1 {
		i.cursorPos--
	}
	i.cursorPos--
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

	i.text = append(i.text[:i.cursorOffset()], i.text[i.cursorOffset()+1:]...)
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
					view.ForwardCursor(0)
				case termbox.KeyArrowUp:
					view.BackwardInputHisotry()
					view.DrawInputHistory()
				case termbox.KeyArrowDown:
					view.ForwardInputHisotry()
					view.DrawInputHistory()
				case termbox.KeySpace:
					view.InputText(rune(' '))
					view.ForwardCursor(0)
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
						view.ClearInputText()
						if exitErr, ok := err.(*exec.ExitError); ok {
							view.InputError(string(exitErr.Stderr))
						} else {
							view.InputError(err.Error())
						}
					} else {
						view.SaveTextHistory()
						view.SetText(&out)
						view.SaveInputHistory()
						view.SaveInvokeCommand()
						view.ClearInputText()
					}
				default:
					if ev.Ch != 0 {
						runeWidth := view.InputText(ev.Ch)
						view.ForwardCursor(runeWidth)
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
