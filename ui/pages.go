package ui

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bastengao/gncdu/scan"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Page interface {
	SetNavigator(nav *Navigator)
	SetPrevious(previous Page)
	Previous() Page
	Show()
	Dispose()
}

type BasePage struct {
	app       *tview.Application
	previous  Page
	navigator *Navigator
}

func (p *BasePage) SetNavigator(nav *Navigator) {
	p.navigator = nav
}

func (p *BasePage) SetPrevious(previous Page) {
	p.previous = previous
}

func (p *BasePage) Previous() Page {
	return p.previous
}

func (p *BasePage) Dispose() {
}

type ScanningPage struct {
	BasePage
	done chan bool
}

func NewScanningPage(app *tview.Application) *ScanningPage {
	done := make(chan bool)
	return &ScanningPage{BasePage: BasePage{app: app}, done: done}
}

func (page *ScanningPage) Show() {
	modal := tview.NewModal().
		SetText("Scanning       \n\nTime 0s")

	info := tview.NewTextView().
		SetText("[ctrl+c] close")

	layout := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(modal, 0, 1, true).
		AddItem(info, 1, 1, false)

	go func() {
		tick := time.Tick(time.Millisecond * 500)
		dots := []byte{'.', '.', '.', '.', '.', '.'}
		spaces := []byte{' ', ' ', ' ', ' ', ' ', ' '}
		count := 0
		start := time.Now()
		for {
			select {
			case <-tick:
				count++
				p := count % 7
				s := string(dots[0:p])
				b := string(spaces[0:(6 - p)])

				// 获取当前进度
				path := "/"
				if cp := scan.CurrentPath.Load(); cp != nil {
					path = cp.(string)
					if len(path) > 40 {
						path = "..." + path[len(path)-37:]
					}
				}
				size := scan.TotalSize.Load()
				items := scan.TotalItems.Load()

				page.app.QueueUpdateDraw(func() {
					modal.SetText(fmt.Sprintf("Scanning %s%s\nPath: %s\nSize: %s    Items: %d\nTime %ds",
						s, b, path, scan.ToHumanSize(size), items, int(time.Now().Sub(start).Seconds())))
				})
			case <-page.done:
				return
			}
		}
	}()

	page.app.SetRoot(layout, true).SetFocus(layout)
}

func (p *ScanningPage) Dispose() {
	close(p.done)
}

type ResultPage struct {
	BasePage
	files  []*scan.FileData
	parent *scan.FileData
}

func NewResultPage(app *tview.Application, files []*scan.FileData, parent *scan.FileData) *ResultPage {
	return &ResultPage{
		BasePage: BasePage{app: app},
		files:    files,
		parent:   parent,
	}
}

func (p *ResultPage) Show() {
	sort.Slice(p.files, func(i, j int) bool {
		return p.files[i].Size() > p.files[j].Size()
	})

	offset := 1
	var title string
	if p.parent != nil {
		if !p.parent.Root() && p.parent.Parent != nil {
			offset = 2
		}
		title = fmt.Sprintf("%s  %s", scan.ToHumanSize(p.parent.Size()), p.parent.Path())
	}

	selectedStyle := tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorWhite)

	table := tview.NewTable().
		SetFixed(1, 1).
		SetSelectable(true, false).
		SetSelectedStyle(selectedStyle).
		SetSelectedFunc(func(row, column int) {
			if row == 0 {
				return
			}

			if row == offset-1 && p.parent != nil && !p.parent.Root() && p.parent.Parent != nil {
				page := NewResultPage(p.app, p.parent.Parent.Children, p.parent.Parent)
				p.navigator.Push(page)
				return
			}

			file := p.files[row-offset]
			if !file.IsDir() {
				return
			}
			page := NewResultPage(p.app, file.Children, file)
			p.navigator.Push(page)
		})

	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyBackspace || event.Key() == tcell.KeyBackspace2 {
			if p.parent != nil && !p.parent.Root() && p.parent.Parent != nil {
				page := NewResultPage(p.app, p.parent.Parent.Children, p.parent.Parent)
				p.navigator.Push(page)
				return nil
			}
			return event
		}

		if event.Rune() == 'd' {
			row, _ := table.GetSelection()
			if row == 0 {
				return event
			}
			if row == offset-1 {
				return event
			}
			i := row - offset
			file := p.files[i]
			confirm := func() {
				err := file.Delete()
				if err != nil {
					return
				}
				p.files = append(p.files[:i], p.files[i+1:]...)
				p.parent.SetChildren(p.files)
			}
			p.navigator.Push(NewDeleteConfirmPage(p.app, file.Name(), confirm))
		}
		return event
	})

	color := tcell.ColorYellow
	table.SetCell(0, 0, tview.NewTableCell("Name").SetTextColor(color).SetSelectable(false))
	table.SetCell(0, 1, tview.NewTableCell("Size").SetTextColor(color).SetSelectable(false))
	table.SetCell(0, 2, tview.NewTableCell("").SetTextColor(color).SetSelectable(false))
	table.SetCell(0, 3, tview.NewTableCell("Items").SetTextColor(color).SetSelectable(false))

	if p.parent != nil && !p.parent.Root() && p.parent.Parent != nil {
		table.SetCell(1, 0, tview.NewTableCell("/.."))
	}

	var maxSize int64
	for i, file := range p.files {
		nameColor := tcell.ColorWhite
		if file.IsDir() {
			nameColor = tcell.ColorDeepSkyBlue
		}

		if i == 0 {
			maxSize = file.Size()
		}

		table.SetCell(i+offset, 0,
			tview.NewTableCell(file.Label()).
				SetTextColor(nameColor))
		table.SetCell(i+offset, 1,
			tview.NewTableCell(scan.ToHumanSize(file.Size())).
				SetAlign(tview.AlignRight))
		table.SetCell(i+offset, 2,
			tview.NewTableCell(percentageText(maxSize, file.Size())).
				SetAlign(tview.AlignLeft))
		table.SetCell(i+offset, 3,
			tview.NewTableCell(strconv.Itoa((file.Count()))).
				SetAlign(tview.AlignRight))
	}

	layout := newLayout(title, table)
	p.app.SetRoot(layout, true).SetFocus(layout)
}

func percentageText(total int64, part int64) string {
	var sb strings.Builder
	sb.WriteString("[")

	percentage := int(math.Round(float64(part) / float64(total) * 20))
	for i := 1; i <= 20; i++ {
		if i <= percentage {
			sb.WriteString("#")
		} else {
			sb.WriteString(" ")
		}
	}

	sb.WriteString("]")
	return sb.String()
}

type HelpPage struct {
	BasePage
}

func NewHelpPage(app *tview.Application) *HelpPage {
	return &HelpPage{BasePage: BasePage{app: app}}
}

func (p *HelpPage) Show() {
	text := fmt.Sprintf(`GNCDU %s

	https://github.com/bastengao/gncdu
	`, Version)
	modal := tview.NewModal().
		SetText(text).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(i int, l string) {
			if i == 0 {
				p.navigator.Pop()
			}
		})

	layout := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(modal, 0, 1, true).
		AddItem(newInfoView(), 1, 1, false)

	p.app.SetRoot(layout, true).SetFocus(layout)
}

type DeleteConfirmPage struct {
	BasePage
	name    string
	confirm func()
}

func NewDeleteConfirmPage(app *tview.Application, name string, confirm func()) *DeleteConfirmPage {
	return &DeleteConfirmPage{BasePage: BasePage{app: app}, name: name, confirm: confirm}
}

func (p *DeleteConfirmPage) Show() {
	modal := tview.NewModal().
		SetText(fmt.Sprintf("Are you sure want to delete \"%s\" ?", p.name)).
		AddButtons([]string{"Cacnel", "OK"}).
		SetDoneFunc(func(i int, l string) {
			if i == 1 {
				p.confirm()
			}
			p.navigator.Pop()
		})

	layout := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(modal, 0, 1, true).
		AddItem(newInfoView(), 1, 1, false)

	p.app.SetRoot(layout, true).SetFocus(layout)
}
