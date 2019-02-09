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

	"github.com/mattn/go-shellwords"
	"github.com/nsf/termbox-go"
)

const (
	Name    = "txtmanip"
	Version = "0.0.0"
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
	if len(v.inputArea.text) < 1 {
		return
	}

	for x := 0; x < v.width; x++ {
		if x < len(v.inputArea.text) {
			termbox.SetCell(x, InputAreaPos, rune(v.inputArea.text[x]), ColFg, ColBg)
		} else {
			termbox.SetCell(x, InputAreaPos, rune(' '), ColFg, ColBg)
		}
	}
}

func (v *MainView) DrawInputError() {
	v.inputArea.drawError()
}

func (v *MainView) DrawTextArea() {
	y := TextAreaPos
	x := 0
	for _, t := range v.textArea.text {
		if t == byte('\n') {
			y++
			x = 0
			continue
		}
		termbox.SetCell(x, y, rune(t), ColFg, ColBg)
		x++
	}
}

func (v *MainView) InputText(ch rune) {
	v.inputArea.input(ch)
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

func (v *MainView) ForwardCursor() {
	v.inputArea.forwardCursor()
}

func (v *MainView) BackwardCursor() {
	v.inputArea.backwardCursor()
}

func (v *MainView) RedoInputHistory() {
	v.inputArea.redoHistory()
}

func (v *MainView) SaveInputHistory() {
	v.inputArea.saveHistory()
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
	text      []byte
	error     []byte
	cursorPos int
	history   []string
}

func (i *InputArea) input(ch rune) {
	if len(i.text) > i.cursorPos && i.text[i.cursorPos] != 0 {
		i.text = append(i.text[:i.cursorPos], append([]byte{byte(ch)}, i.text[i.cursorPos:]...)...)
		return
	}

	i.text = append(i.text, byte(ch))
}

func (i *InputArea) initCursor() {
	if i.cursorPos < 1 {
		return
	}
	i.cursorPos = 0
}

func (i *InputArea) endCursor() {
	if i.cursorPos >= len(i.text) {
		return
	}
	i.cursorPos = len(i.text)
}

func (i *InputArea) forwardCursor() {
	if i.cursorPos >= len(i.text) {
		return
	}
	i.cursorPos++
}

func (i *InputArea) backwardCursor() {
	if i.cursorPos < 1 {
		return
	}
	i.cursorPos--
}

func (i *InputArea) saveHistory() {
	i.history = append(i.history, string(i.text))
}

func (i *InputArea) clear() {
	i.initCursor()
	i.text = []byte("")
}

func (i *InputArea) drawError() {
	if len(i.error) < 1 {
		return
	}

	for x, t := range i.error {
		termbox.SetCell(x, InputErrorPos, rune(t), ColErr, ColBg)
	}
	i.error = []byte("")
}

func (i *InputArea) redoHistory() {
	i.history = i.history[:len(i.history)-1]
}

func (i *InputArea) delete() {
	if len(i.text) < 1 {
		return
	}

	i.text = append(i.text[:i.cursorPos], i.text[i.cursorPos+1:]...)
}

// TextArea represent text area
type TextArea struct {
	text    []byte
	history []string
}

func (t *TextArea) setText(out *[]byte) {
	t.text = *out
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

	cmdHistoryCh := make(chan []string)
	errCh := make(chan error)

	go func() {
		if err := termbox.Init(); err != nil {
			errCh <- errors.New(fmt.Sprint("initialize failed: ", err.Error()))
			return
		}

		w, h := termbox.Size()
		view := &MainView{
			textArea: TextArea{
				text: text,
			},
			inputArea: InputArea{},
			width:     w,
			height:    h,
		}
		defer func() {
			termbox.Close()
			cmdHistoryCh <- view.inputArea.history
		}()

		termbox.SetInputMode(termbox.InputEsc)

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
					view.ForwardCursor()
				case termbox.KeySpace:
					view.InputText(rune(' '))
					view.ForwardCursor()
				case termbox.KeyCtrlZ:
					if len(view.textArea.history) < 1 {
						continue
					}
					view.RedoText()
					view.RedoInputHistory()
				case termbox.KeyBackspace, termbox.KeyBackspace2:
					view.BackwardCursor()
					view.DeleteInputText()
				case termbox.KeyDelete, termbox.KeyCtrlD:
					view.DeleteInputText()
				case termbox.KeyEnter:
					if len(view.inputArea.text) < 1 {
						break mainloop
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
						view.ClearInputText()
					}
				default:
					if ev.Ch != 0 {
						view.InputText(ev.Ch)
						view.ForwardCursor()
					}
				}
			}
		}
	}()

	select {
	case err := <-errCh:
		fmt.Fprintf(os.Stderr, err.Error())
		return ExitCodeError
	case cmdHistory := <-cmdHistoryCh:
		if f == "" {
			f = "<source file>"
		}
		fmt.Println(fmt.Sprintf("cat %s | ", f), strings.Join(cmdHistory, " | "))
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
  Ctrl+Z       Redo text.
  Ctrl+C       Quit interactive mode.
`)
}
