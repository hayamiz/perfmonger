//usr/bin/env go run $0 $@ ; exit

package main

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"time"

	gocui "github.com/jroimartin/gocui"
	termbox "github.com/nsf/termbox-go"
)

func main() {
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
// ' ': \u0020
// '▁': \u2581
// '▂': \u2582
// '▃': \u2583
// '▄': \u2584
// '▅': \u2585
// '▆': \u2586
// '▇': \u2587
// '█': \u2588
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

	// termbox.SetCell(x, y, vbar_runes[pct_class], fg, bg)
}

func drawRunes(g *gocui.Gui, x int, y int, runes []rune) {
	for _, r := range runes {
		termbox.SetCell(x, y, r, termbox.ColorDefault, termbox.ColorDefault)
		x += 1
	}
}

func intMin(x, y int) int {
	if x > y {
		return y
	} else {
		return x
	}
}

var start_time int64 = time.Now().Unix()

func layout(g *gocui.Gui) error {
	// maxX, maxY := g.Size()
	// if v, err := g.SetView("hello", maxX/2-7, maxY/2, maxX/2+7, maxY/2+2); err != nil {
	// 	if err != gocui.ErrUnknownView {
	// 		return err
	// 	}
	// 	fmt.Fprintln(v, "Hello world!")
	// }

	for coreid := 0; coreid < 8; coreid++ {
		s := int64(coreid + 1)
		rand.Seed(s)

		inc := int(time.Now().Unix() - start_time)
		for i := 0; i < inc; i++ {
			rand.Float64()
		}

		for i := 0; i < 100; i++ {
			// drawPercent(g, i+5, 1+coreid*5, 4, float64(i)/2.0, termbox.ColorDefault, termbox.ColorDefault)
			drawPercent(g, i+6, 1+coreid*5, 4, rand.Float64()*100, termbox.ColorDefault, termbox.ColorDefault)

			ch := '-'
			if (i+inc)%10 == 0 {
				ch = '+'
			}
			termbox.SetCell(i+6, 1+coreid*5+4, ch, termbox.ColorDefault, termbox.ColorDefault)
		}
		drawRunes(g, 0, 1+coreid*5, []rune(fmt.Sprintf("core%d", coreid)))

		drawRunes(g, 100+6+1, 1+coreid*5, []rune(fmt.Sprintf("%%usr: % 2.1f", rand.Float64()*30)))
		drawRunes(g, 100+6+1, 1+coreid*5+1, []rune(fmt.Sprintf("%%sys: % 2.1f", rand.Float64()*30)))
		drawRunes(g, 100+6+1, 1+coreid*5+2, []rune(fmt.Sprintf("%%iow: % 2.1f", rand.Float64()*30)))
		drawRunes(g, 100+6+1, 1+coreid*5+3, []rune(fmt.Sprintf("%%oth: % 2.1f", rand.Float64()*30)))
	}

	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}
