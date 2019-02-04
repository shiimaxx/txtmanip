package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/mattn/go-shellwords"
	"github.com/nsf/termbox-go"
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

	for x := 0; x < v.width; x++ {
		termbox.SetCell(x, BorderLinePos, rune('-'), ColFg, ColBg)
	}

	v.inputArea.Clear()
	v.inputArea.DrawError()
	v.textArea.Draw()

	return termbox.Flush()
}

// TextArea represent text area
type TextArea struct {
	text    []byte
	history []string
}

func (t *TextArea) Draw() {
	x := 0
	y := TextAreaPos
	for _, t := range t.text {
		if t == byte('\n') {
			y++
			x = 0
			continue
		}
		termbox.SetCell(x, y, rune(t), ColFg, ColBg)
		x++
	}
}

func (t *TextArea) Redo() {
	t.text = []byte(t.history[len(t.history)-1])
	t.history = t.history[:len(t.history)-1]
}

// InputArea represent input area
type InputArea struct {
	text      []byte
	error     []byte
	cursorPos int
	history   []string
}

//func (i *InputArea) DrawText() {
//	if len(i.text) < 1 {
//		return
//	}
//}

func (i *InputArea) Input(ch rune) {
	//if len(i.text) > i.cursorPos && i.text[i.cursorPos] != 0 {
	//	before := i.text[i.cursorPos:]
	//	after := i.text[:i.cursorPos]
	//	i.text = append(before, byte(ch))
	//	i.text = append(i.text, after...)
	//	for i, t := range i.text {
	//		termbox.SetCell(i, 0, rune(t), termbox.ColorWhite, termbox.ColorDefault)
	//	}
	//	return
	//}

	termbox.SetCell(i.cursorPos, InputAreaPos, ch, termbox.ColorWhite, termbox.ColorDefault)
	i.text = append(i.text, byte(ch))
}

func (i *InputArea) InitCursor() {
	if i.cursorPos < 1 {
		return
	}
	i.cursorPos = 0
	termbox.SetCursor(i.cursorPos, 0)
}

func (i *InputArea) EndCursor() {
	if i.cursorPos >= len(i.text) {
		return
	}
	i.cursorPos = len(i.text)
	termbox.SetCursor(i.cursorPos, 0)
}

func (i *InputArea) ForwardCursor() {
	if i.cursorPos >= len(i.text) {
		return
	}
	i.cursorPos++
	termbox.SetCursor(i.cursorPos, 0)
}

func (i *InputArea) BackwardCursor() {
	if i.cursorPos < 1 {
		return
	}
	i.cursorPos--
	termbox.SetCursor(i.cursorPos, 0)
}

func (i *InputArea) SaveHistory() {
	i.history = append(i.history, string(i.text))
}

func (i *InputArea) Clear() {
	i.InitCursor()
	i.text = []byte("")
}

func (i *InputArea) DrawError() {
	if len(i.error) < 1 {
		return
	}

	for x, t := range i.error {
		termbox.SetCell(x, InputErrorPos, rune(t), ColErr, ColBg)
	}
	i.error = []byte("")
}

func (i *InputArea) RedoHistory() {
	i.history = i.history[:len(i.history)-1]
}

func main() {
	if err := termbox.Init(); err != nil {
		panic(err)
	}
	//defer termbox.Close()
	termbox.SetInputMode(termbox.InputEsc)

	f := os.Args[1]
	if _, err := os.Stat(f); os.IsNotExist(err) {
		panic(err)
	}
	text, err := ioutil.ReadFile(f)
	if err != nil {
		panic(err)
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

	if err := view.Flush(); err != nil {
		panic(err)
	}

mainloop:
	for {
		switch ev := termbox.PollEvent(); ev.Type {
		case termbox.EventKey:
			switch ev.Key {
			case termbox.KeyEsc, termbox.KeyCtrlC:
				break mainloop
			case termbox.KeyCtrlA:
				view.inputArea.InitCursor()
				termbox.Flush()
			case termbox.KeyCtrlE:
				view.inputArea.EndCursor()
				termbox.Flush()
			case termbox.KeyArrowLeft, termbox.KeyCtrlB:
				view.inputArea.BackwardCursor()
				termbox.Flush()
			case termbox.KeyArrowRight, termbox.KeyCtrlF:
				view.inputArea.ForwardCursor()
				termbox.Flush()
			case termbox.KeySpace:
				view.inputArea.Input(rune(' '))
				view.inputArea.ForwardCursor()
				termbox.Flush()
			case termbox.KeyCtrlZ:
				if len(view.textArea.history) < 1 {
					continue
				}
				view.textArea.Redo()
				view.inputArea.RedoHistory()
				view.Flush()
			case termbox.KeyEnter:
				if len(view.inputArea.text) < 1 {
					break mainloop
				}

				args, err := shellwords.Parse(string(view.inputArea.text))
				if err != nil {
					panic(err)
				}

				baseCommand, opts := args[0], args[1:]
				cmd := exec.Command(baseCommand, opts...)
				cmd.Stdin = bufio.NewReader(bytes.NewBuffer(view.textArea.text))

				out, err := cmd.Output()
				if err != nil {
					if exitErr, ok := err.(*exec.ExitError); ok {
						view.inputArea.error = exitErr.Stderr
					} else {
						view.inputArea.error = []byte(err.Error())
					}
				} else {
					view.textArea.history = append(view.textArea.history, string(view.textArea.text))
					view.textArea.text = out
					view.inputArea.SaveHistory()
				}
				view.Flush()
			default:
				if ev.Ch != 0 {
					view.inputArea.Input(ev.Ch)
					view.inputArea.ForwardCursor()
					termbox.Flush()
				}
			}
		}
	}
	termbox.Flush()

	fmt.Println(fmt.Sprintf("cat %s | ", f), strings.Join(view.inputArea.history, " | "))
}
