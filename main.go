package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/nsf/termbox-go"
)

// TextArea represent text area
type TextArea struct {
	text []byte
}

func (t *TextArea) Draw() {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)

	x := 0
	y := 2
	for _, t := range t.text {
		if t == byte('\n') {
			y++
			x = 0
			continue
		}
		termbox.SetCell(x, y, rune(t), termbox.ColorWhite, termbox.ColorDefault)
		x++
	}
}

// InputArea represent input area
type InputArea struct {
	text         []byte
	cursorOffset int
}

//func (i *InputArea) DrawText() {
//	if len(i.text) < 1 {
//		return
//	}
//}

func (i *InputArea) Input(ch rune) {
	//if len(i.text) > i.cursorOffset && i.text[i.cursorOffset] != 0 {
	//	before := i.text[i.cursorOffset:]
	//	after := i.text[:i.cursorOffset]
	//	i.text = append(before, byte(ch))
	//	i.text = append(i.text, after...)
	//	for i, t := range i.text {
	//		termbox.SetCell(i, 0, rune(t), termbox.ColorWhite, termbox.ColorDefault)
	//	}
	//	return
	//}

	termbox.SetCell(i.cursorOffset, 0, ch, termbox.ColorWhite, termbox.ColorDefault)
	i.text = append(i.text, byte(ch))
}

func (i *InputArea) InitCursor() {
	if i.cursorOffset < 1 {
		return
	}
	i.cursorOffset = 0
	termbox.SetCursor(i.cursorOffset, 0)
}

func (i *InputArea) EndCursor() {
	if i.cursorOffset >= len(i.text) {
		return
	}
	i.cursorOffset = len(i.text)
	termbox.SetCursor(i.cursorOffset, 0)
}

func (i *InputArea) ForwardCursor() {
	if i.cursorOffset >= len(i.text) {
		return
	}
	i.cursorOffset++
	termbox.SetCursor(i.cursorOffset, 0)
}

func (i *InputArea) BackwardCursor() {
	if i.cursorOffset < 1 {
		return
	}
	i.cursorOffset--
	termbox.SetCursor(i.cursorOffset, 0)
}

func initDraw() {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	w, _ := termbox.Size()

	for x := 0; x < w; x++ {
		termbox.SetCell(x, 1, rune('-'), termbox.ColorWhite, termbox.ColorDefault)
	}

	termbox.SetCursor(0, 0)

}

func main() {
	err := termbox.Init()
	if err != nil {
		panic(err)
	}
	defer termbox.Close()
	termbox.SetInputMode(termbox.InputEsc)

	file, err := os.Open("./base.shiimaxx.com_ssl_access_log")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	text, err := ioutil.ReadAll(file)
	if err != nil {
		panic(err)
	}

	var textArea TextArea
	textArea.text = text

	initDraw()

	textArea.Draw()

	if err := termbox.Flush(); err != nil {
		panic(err)
	}

	var inputArea InputArea

mainloop:
	for {
		switch ev := termbox.PollEvent(); ev.Type {
		case termbox.EventKey:
			switch ev.Key {
			case termbox.KeyEsc, termbox.KeyCtrlC:
				break mainloop
			case termbox.KeyCtrlA:
				inputArea.InitCursor()
			case termbox.KeyCtrlE:
				inputArea.EndCursor()
			case termbox.KeyArrowLeft, termbox.KeyCtrlH:
				inputArea.BackwardCursor()
			case termbox.KeyArrowRight, termbox.KeyCtrlF:
				inputArea.ForwardCursor()
			case termbox.KeySpace:
				inputArea.Input(rune(' '))
				inputArea.ForwardCursor()
			case termbox.KeyEnter:
				if err := ioutil.WriteFile("./tmp.txt", textArea.text, os.ModePerm); err != nil {
					panic(err)
				}

				commandLine := strings.Split(fmt.Sprint(string(inputArea.text), " ./tmp.txt"), " ")
				baseCommand, opts := commandLine[0], commandLine[1:]

				cmd := exec.Command(baseCommand, opts...)

				out, err := cmd.Output()
				if err != nil {
					//if exitErr, ok := err.(*exec.ExitError); ok {
					//	panic(string(exitErr.Stderr))
					//}
					panic(strings.Join(commandLine, " "))
				}

				textArea.text = out
				textArea.Draw()
				inputArea.InitCursor()

				os.Remove("./tmp.txt")
				inputArea.text = []byte("")
			default:
				if ev.Ch != 0 {
					inputArea.Input(ev.Ch)
					inputArea.ForwardCursor()
				}
			}
		}
		termbox.Flush()
	}
}
