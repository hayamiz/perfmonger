package viewer

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"time"

	gocui "github.com/jroimartin/gocui"
	termbox "github.com/nsf/termbox-go"
)

func Run(args []string) {
	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		log.Panicln(err)
	}
	defer g.Close()

	g.SetManagerFunc(layout)

	setupKeybind(g)

	go func() {
		for {
			time.Sleep(1 * time.Second)
			g.Update(layout)
		}
	}()

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Panicln(err)
	}
}

func setupKeybind(g *gocui.Gui) {
	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		log.Panicln(err)
	}

	if err := g.SetKeybinding("", 'c', gocui.ModNone, changeColor); err != nil {
		log.Panicln(err)
	}

	if err := g.SetKeybinding("", 'q', gocui.ModNone, quit); err != nil {
		log.Panicln(err)
	}
}

var colmode bool = false

func changeColor(g *gocui.Gui, v *gocui.View) error {
	if colmode {
		colmode = false
	} else {
		colmode = true
	}
	return nil
}

// Vertical bar characters
var vbar_runes []rune = []rune(" ▁▂▃▄▅▆▇█")

func drawPercent(g *gocui.Gui, x int, y int, height int, pct float64, fg, bg termbox.Attribute) {
	pct = math.Max(0.0, math.Min(100.0, pct))
	pct_ndiv := 9
	if height > 1 {
		pct_ndiv += 8 * (height - 1)
	}
	pct_class := int(pct / (100.0 / float64(pct_ndiv)))

	col := termbox.ColorBlue
	if 25 <= pct && pct < 50 {
		col = termbox.ColorGreen
	} else if 50 <= pct && pct < 75 {
		col = termbox.ColorYellow
	} else if 75 <= pct {
		col = termbox.ColorRed
	}

	if !colmode {
		col = termbox.ColorWhite
	}

	for cell_pos := height - 1; cell_pos >= 0; cell_pos-- {
		vbar_idx := intMin(pct_class, len(vbar_runes)-1)
		termbox.SetCell(x, y+cell_pos, vbar_runes[vbar_idx], col, bg)
		pct_class -= vbar_idx
	}
}

func intMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func intMax(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	if v, err := g.SetView("hello", maxX/2-7, maxY/2, maxX/2+7, maxY/2+2); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		fmt.Fprintln(v, "Hello world!")
		fmt.Fprintln(v, "Press q to quit")
	}

	// Demo CPU usage bars
	for i := 0; i < intMin(8, maxY-2); i++ {
		usage := rand.Float64() * 100
		drawPercent(g, i*2, 1, 1, usage, termbox.ColorWhite, termbox.ColorDefault)
	}

	return nil
}