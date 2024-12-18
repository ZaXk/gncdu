package ui

import (
	"fmt"
	"sync/atomic"

	"github.com/bastengao/gncdu/scan"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func ShowUI(scanDir func() ([]*scan.FileData, error)) {
	app := tview.NewApplication()

	var scanningPage Page = NewScanningPage(app)

	var isDone atomic.Value
	done := make(chan bool)
	go func() {
		files, err := scanDir()
		if err != nil {
			fmt.Println(err)
			app.Stop()
			return
		}
		close(done)
		isDone.Store(true)
		app.QueueUpdateDraw(func() {
			var parent *scan.FileData
			if len(files) > 0 {
				parent = files[0].Parent
			}
			resultPage := NewResultPage(app, files, parent)
			navigator.Push(resultPage)
		})
	}()

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if isDone.Load() != nil && event.Rune() == '?' {
			help := NewHelpPage(app)
			navigator.Push(help)
		}
		return event
	})

	navigator.Push(scanningPage)
	if err := app.Run(); err != nil {
		panic(err)
	}
}

func newInfoView() tview.Primitive {
	return tview.NewTextView().
		SetText("[ctrl+c] close    [d] delete    [backspace] back    [?] help")
}

func newLayout(title string, content tview.Primitive) tview.Primitive {
	t := tview.NewTextView().
		SetText(title)
	info := newInfoView()

	layout := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(t, 1, 1, false).
		AddItem(content, 0, 1, true).
		AddItem(info, 1, 1, false)

	return layout
}
