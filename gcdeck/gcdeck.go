// gcdeck: render deck markup using the gio canvas
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"math"
	"os"
	"strconv"
	"strings"
	"unicode"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"gioui.org/app"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/io/system"
	"gioui.org/op"
	"gioui.org/unit"
	"github.com/ajstarks/deck"
	gc "github.com/ajstarks/giocanvas"
)

const (
	mm2pt       = 2.83464 // mm to pt conversion
	linespacing = 1.8
	listspacing = 1.8
	fontfactor  = 1.0
	listwrap    = 95.0
)

// PageDimen describes page dimensions
// the unit field is used to convert to pt.
type PageDimen struct {
	width, height, unit float64
}

// fontmap maps generic font names to specific implementation names
var fontmap = map[string]string{}

// pagemap defines page dimensions
var pagemap = map[string]PageDimen{
	"Letter":     {792, 612, 1},
	"Legal":      {1008, 612, 1},
	"Tabloid":    {1224, 792, 1},
	"ArchA":      {864, 648, 1},
	"Widescreen": {1152, 648, 1},
	"4R":         {432, 288, 1},
	"Index":      {360, 216, 1},
	"A2":         {420, 594, mm2pt},
	"A3":         {420, 297, mm2pt},
	"A4":         {297, 210, mm2pt},
	"A5":         {210, 148, mm2pt},
}

var codemap = strings.NewReplacer("\t", "    ")

// setpagesize parses the page size string (wxh)
func setpagesize(s string) (float64, float64) {
	var width, height float64
	var err error
	d := strings.FieldsFunc(s, func(c rune) bool { return !unicode.IsNumber(c) })
	if len(d) != 2 {
		return 0, 0
	}
	width, err = strconv.ParseFloat(d[0], 64)
	if err != nil {
		return 0, 0
	}
	height, err = strconv.ParseFloat(d[1], 64)
	if err != nil {
		return 0, 0
	}
	return width, height
}

// pagedim converts a named pagesize to width, height
func pagedim(s string) (float32, float32) {
	pw, ph := setpagesize(s)
	if pw == 0 && ph == 0 {
		p, ok := pagemap[s]
		if !ok {
			p = pagemap["Letter"]
		}
		pw = p.width * p.unit
		ph = p.height * p.unit
	}
	return float32(pw), float32(ph)
}

// includefile returns the contents of a file as string
func includefile(filename string) string {
	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return ""
	}
	return codemap.Replace(string(data))
}

// pct converts percentages to canvas measures
func pct(p, m float64) float64 {
	return (p / 100.0) * m
}

// radians converts degrees to radians
func radians(degrees float64) float64 {
	return degrees * math.Pi / 180
}

// fontlookup maps font aliases to implementation font names
func fontlookup(s string) string {
	font, ok := fontmap[s]
	if ok {
		return font
	}
	return "sans"
}

// setop sets the opacity as a truncated fraction of 255
func setop(v float64) uint8 {
	if v > 0.0 {
		return uint8(255.0 * (v / 100.0))
	}
	return 255
}

// gradient sets the background color gradient
func gradient(doc *gc.Canvas, w, h float64, gc1, gc2 string, gp float64) {
}

// doline draws a line
func doline(doc *gc.Canvas, xp1, yp1, xp2, yp2, sw float64, color string, opacity float64) {
	c := gc.ColorLookup(color)
	c.A = setop(opacity)
	doc.Line(float32(xp1), float32(yp1), float32(xp2), float32(yp2), float32(sw), c)
}

// doarc draws an arc
func doarc(doc *gc.Canvas, x, y, w, h, a1, a2, sw float64, color string, opacity float64) {
	c := gc.ColorLookup(color)
	c.A = setop(opacity)
	doc.Arc(float32(x), float32(y), float32(w*2), radians(a1), radians(a2), c)
}

// docurve draws a bezier curve
func docurve(doc *gc.Canvas, xp1, yp1, xp2, yp2, xp3, yp3, sw float64, color string, opacity float64) {
	c := gc.ColorLookup(color)
	c.A = setop(opacity)
	doc.StrokedCurve(float32(xp1), float32(yp1), float32(xp2), float32(yp2), float32(xp3), float32(yp3), float32(sw), c)
}

// dorect draws a rectangle
func dorect(doc *gc.Canvas, x, y, w, h float64, color string, opacity float64) {
	c := gc.ColorLookup(color)
	c.A = setop(opacity)
	doc.CenterRect(float32(x), float32(y), float32(w), float32(h), c)
}

// doellipse draws an ellipse
func doellipse(doc *gc.Canvas, x, y, w, h float64, color string, opacity float64) {
	c := gc.ColorLookup(color)
	c.A = setop(opacity)
	doc.Ellipse(float32(x), float32(y), float32(w/2), float32(h/2), c)
}

// dopoly draws a polygon
func dopoly(doc *gc.Canvas, xc, yc string, cw, ch float64, color string, opacity float64) {
	xs := strings.Split(xc, " ")
	ys := strings.Split(yc, " ")
	if len(xs) != len(ys) {
		return
	}
	if len(xs) < 3 || len(ys) < 3 {
		return
	}
	px := make([]float32, len(xs))
	py := make([]float32, len(xs))
	for i := 0; i < len(xs); i++ {
		x, err := strconv.ParseFloat(xs[i], 32)
		if err != nil {
			px[i] = 0
		} else {
			px[i] = float32(x)
		}
		y, err := strconv.ParseFloat(ys[i], 32)
		if err != nil {
			py[i] = 0
		} else {
			py[i] = float32(y)
		}
	}
	c := gc.ColorLookup(color)
	c.A = setop(opacity)
	doc.Polygon(px, py, c)
}

// dotext places text elements on the canvas according to type
func dotext(doc *gc.Canvas, x, y, fs, wp, rotation, spacing float64, tdata, font, align, ttype, color string, opacity float64) {
	td := strings.Split(tdata, "\n")
	c := gc.ColorLookup(color)
	var tstack op.TransformStack
	if rotation > 0 {
		tstack = doc.Rotate(float32(x), float32(y), float32(rotation*(math.Pi/180)))
	}
	if ttype == "code" {
		font = "mono"
		ch := float64(len(td)) * spacing * fs
		bx := (x + (wp / 2))
		by := (y - (ch / 2)) + (spacing * fs)
		dorect(doc, bx, by, wp+fs, ch+fs, "rgb(240,240,240)", 100)
	}
	if ttype == "block" {
		textwrap(doc, x, y, fs, wp, tdata, color, opacity)
	} else {
		ls := spacing * fs
		for _, t := range td {
			showtext(doc, x, y, t, fs, c, font, align)
			y -= ls
		}
	}
	if rotation > 0 {
		gc.EndTransform(tstack)
	}
}

// textwrap places and wraps text at a width
func textwrap(doc *gc.Canvas, x, y, fs, wp float64, tdata, color string, opacity float64) {
	c := gc.ColorLookup(color)
	c.A = setop(opacity)
	doc.TextWrap(float32(x), float32(y), float32(fs), float32(wp), tdata, c)
}

// whitespace determines if a rune is whitespace
func whitespace(r rune) bool {
	return r == ' ' || r == '\n' || r == '\t'
}

// loadfont loads a font at the specified size
func loadfont(doc *gc.Canvas, s string, size float64) {
}

// showtext places fully attributed text at the specified location
func showtext(doc *gc.Canvas, x, y float64, s string, fs float64, color color.NRGBA, font, align string) {
	loadfont(doc, font, fs)
	tx := float32(x)
	ty := float32(y)
	tfs := float32(fs)
	switch align {
	case "center", "middle", "mid", "c":
		doc.TextMid(tx, ty, tfs, s, color)
	case "right", "end", "e":
		doc.TextEnd(tx, ty, tfs, s, color)
	default:
		doc.Text(tx, ty, tfs, s, color)
	}
}

// dolist places lists on the canvas
func dolist(doc *gc.Canvas, cw, x, y, fs, lwidth, rotation, spacing float64, list []deck.ListItem, font, ltype, align, color string, opacity float64) {
	if font == "" {
		font = "sans"
	}
	var tstack op.TransformStack
	if rotation > 0 {
		tstack = doc.Rotate(float32(x), float32(y), float32(rotation*(math.Pi/180)))
	}
	c := gc.ColorLookup(color)
	ls := listspacing * fs * 1.4
	for i, tl := range list {
		loadfont(doc, font, fs)
		if len(tl.Color) > 0 {
			c = gc.ColorLookup(tl.Color)
		}
		switch ltype {
		case "number":
			showtext(doc, x, y, fmt.Sprintf("%d. ", i+1)+tl.ListText, fs, c, font, align)
		case "bullet":
			doc.Circle(float32(x), float32(y+fs/3), float32(fs/4), c)
			showtext(doc, x+fs, y, tl.ListText, fs, c, font, align)
		case "center":
			showtext(doc, x, y, tl.ListText, fs, c, font, align)
		default:
			showtext(doc, x, y, tl.ListText, fs, c, font, align)
		}
		y -= ls
	}
	if rotation > 0 {
		gc.EndTransform(tstack)
	}
}

// showslide shows a slide
func showslide(doc *gc.Canvas, d *deck.Deck, n int) {
	if n < 0 || n > len(d.Slide)-1 {
		return
	}
	cw := float64(d.Canvas.Width)
	ch := float64(d.Canvas.Height)
	slide := d.Slide[n]
	// set default background
	if slide.Bg == "" {
		slide.Bg = "white"
	}
	doc.Background(gc.ColorLookup(slide.Bg))

	if slide.GradPercent <= 0 || slide.GradPercent > 100 {
		slide.GradPercent = 100
	}
	// set gradient background, if specified. You need both colors
	if len(slide.Gradcolor1) > 0 && len(slide.Gradcolor2) > 0 {
		gradient(doc, cw, ch, slide.Gradcolor1, slide.Gradcolor2, slide.GradPercent)
	}
	// set the default foreground
	if slide.Fg == "" {
		slide.Fg = "black"
	}
	// for every image on the slide...
	for _, im := range slide.Image {
		iw, ih := im.Width, im.Height
		// scale the image to a percentage of the canvas width
		if im.Height == 0 && im.Width > 0 {
			nw, nh := imageInfo(im.Name)
			if nh > 0 {
				var fw, fh float64
				imscale := (float64(iw) / 100) * cw
				fw = imscale
				fh = imscale / (float64(nw) / float64(nh))
				iw = int(fw)
				ih = int(fh)
			}
			//println(im.Name, im.Xp, im.Yp, iw, ih, nw, nh)
		}
		doc.Image(im.Name, float32(im.Xp), float32(im.Yp), iw, ih, float32(im.Scale))
		if len(im.Caption) > 0 {
			capsize := 1.5
			if im.Font == "" {
				im.Font = "sans"
			}
			if im.Color == "" {
				im.Color = slide.Fg
			}
			if im.Align == "" {
				im.Align = "center"
			}
			var cx, cy float64
			iw := float64(im.Width) * (im.Scale / 100)
			ih := float64(im.Height) * (im.Scale / 100)
			cimx := im.Xp
			switch im.Align {
			case "center", "c", "mid":
				cx = im.Xp
			case "end", "e", "right":
				cx = cimx + pct((iw/2), cw)
			default:
				cx = cimx - pct((iw/2), cw)
			}
			cy = im.Yp - (ih/2)/ch*100 - (capsize * 2)
			showtext(doc, cx, cy, im.Caption, capsize, gc.ColorLookup(im.Color), im.Font, im.Align)
		}
	}
	// every graphic on the slide
	const defaultColor = "rgb(127,127,127)"
	// rect
	for _, rect := range slide.Rect {
		if rect.Color == "" {
			rect.Color = defaultColor
		}
		if rect.Hr == 100 {
			c := gc.ColorLookup(rect.Color)
			c.A = setop(rect.Opacity)
			doc.CenterRect(float32(rect.Xp), float32(rect.Yp), float32(rect.Wp), float32((rect.Wp)*(cw/ch)), c)
		} else {
			dorect(doc, rect.Xp, rect.Yp, rect.Wp, rect.Hp, rect.Color, rect.Opacity)
		}
	}
	// ellipse
	for _, ellipse := range slide.Ellipse {
		if ellipse.Color == "" {
			ellipse.Color = defaultColor
		}
		if ellipse.Hr == 100 {
			c := gc.ColorLookup(ellipse.Color)
			c.A = setop(ellipse.Opacity)
			doc.Circle(float32(ellipse.Xp), float32(ellipse.Yp), float32(ellipse.Wp/2), c)
		} else {
			doellipse(doc, ellipse.Xp, ellipse.Yp, ellipse.Wp, ellipse.Hp, ellipse.Color, ellipse.Opacity)
		}
	}
	// curve
	for _, curve := range slide.Curve {
		if curve.Color == "" {
			curve.Color = defaultColor
		}
		if curve.Sp == 0 {
			curve.Sp = 0.2
		}
		docurve(doc, curve.Xp1, curve.Yp1, curve.Xp2, curve.Yp2, curve.Xp3, curve.Yp3, curve.Sp, curve.Color, curve.Opacity)
	}
	// arc
	for _, arc := range slide.Arc {
		if arc.Color == "" {
			arc.Color = defaultColor
		}
		w := arc.Wp
		h := arc.Hp
		if arc.Sp == 0 {
			arc.Sp = 0.2
		}
		doarc(doc, arc.Xp, arc.Yp, w/2, h/2, arc.A1, arc.A2, arc.Sp, arc.Color, arc.Opacity)
	}
	// line
	for _, line := range slide.Line {
		if line.Color == "" {
			line.Color = defaultColor
		}
		if line.Sp == 0 {
			line.Sp = 0.2
		}
		doline(doc, line.Xp1, line.Yp1, line.Xp2, line.Yp2, line.Sp, line.Color, line.Opacity)
	}
	// polygon
	for _, poly := range slide.Polygon {
		if poly.Color == "" {
			poly.Color = defaultColor
		}
		dopoly(doc, poly.XC, poly.YC, cw, ch, poly.Color, poly.Opacity)
	}

	// for every text element...
	var tdata string
	for _, t := range slide.Text {
		if t.Color == "" {
			t.Color = slide.Fg
		}
		if t.Font == "" {
			t.Font = "sans"
		}
		if t.File != "" {
			tdata = includefile(t.File)
		} else {
			tdata = t.Tdata
		}
		if t.Lp == 0 {
			t.Lp = linespacing
		}
		dotext(doc, t.Xp, t.Yp, t.Sp, t.Wp, t.Rotation, t.Lp*1.2, tdata, t.Font, t.Align, t.Type, t.Color, t.Opacity)
	}
	// for every list element...
	for _, l := range slide.List {
		if l.Color == "" {
			l.Color = slide.Fg
		}
		if l.Lp == 0 {
			l.Lp = listspacing
		}
		if l.Wp == 0 {
			l.Wp = listwrap
		}
		dolist(doc, cw, l.Xp, l.Yp, l.Sp, l.Wp, l.Rotation, l.Lp, l.Li, l.Font, l.Type, l.Align, l.Color, l.Opacity)
	}

}

// imageinfo returns the dimensions of an image
func imageInfo(s string) (int, int) {
	f, err := os.Open(s)
	defer f.Close()
	if err != nil {
		return 0, 0
	}
	im, _, err := image.DecodeConfig(f)
	if err != nil {
		return 0, 0
	}
	return im.Width, im.Height
}

// ReadDeck reads the deck file, rendering to the canvas
func readDeck(filename string, w, h float32) (deck.Deck, error) {
	d, err := deck.Read(filename, int(w), int(h))
	d.Canvas.Width = int(w)
	d.Canvas.Height = int(h)
	return d, err
}

// reload reloads the content and shows the first slide
func reload(filename string, c *gc.Canvas, w, h, n int) (deck.Deck, int) {
	d, err := readDeck(filename, float32(w), float32(h))
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return d, 0
	}
	showslide(c, &d, n)
	return d, len(d.Slide) - 1
}

// ngrid makes a numbered grid
func ngrid(c *gc.Canvas, interval, ts float32, color color.NRGBA) {
	color.A = 50
	c.Grid(0, 0, 100, 100, 0.1, interval, color)
	var x, y float32
	color.A = 220
	for x = interval; x < 100; x += interval {
		c.CText(x, ts, ts, fmt.Sprintf("%0.f", x), color)
	}
	for y = interval; y < 100; y += interval {
		c.CText(ts, y-(ts/2), ts, fmt.Sprintf("%0.f", y), color)
	}
}

func main() {
	var (
		title    = flag.String("title", "", "slide title")
		pagesize = flag.String("pagesize", "Letter", "pagesize: w,h, or one of: Letter, Legal, Tabloid, A3, A4, A5, ArchA, 4R, Index, Widescreen")
		initpage = flag.Int("page", 1, "initial page")
	)
	flag.Parse()

	// get the filename
	var filename string
	if len(flag.Args()) < 1 {
		filename = "-"
		*title = "Standard Input"
	} else {
		filename = flag.Args()[0]
	}
	if *title == "" {
		*title = filename
	}
	go slidedeck(*title, *initpage, filename, *pagesize)
	app.Main()
}

var pressed bool
var gridstate bool
var slidenumber int

func kbpointer(q event.Queue, ns int) {
	for _, ev := range q.Events(pressed) {
		if k, ok := ev.(key.Event); ok {
			switch k.State {
			case key.Press:
				switch k.Name {
				// emacs bindings
				case "A", "1": // first slide
					if k.Modifiers == 0 || k.Modifiers == key.ModCtrl {
						slidenumber = 0
					}
				case "E": // last slide
					if k.Modifiers == 0 || k.Modifiers == key.ModCtrl {
						slidenumber = ns
					}
				case "B": // back a slide
					if k.Modifiers == 0 || k.Modifiers == key.ModCtrl {
						slidenumber--
					}
				case "F": // forward a slide
					if k.Modifiers == 0 || k.Modifiers == key.ModCtrl {
						slidenumber++
					}
				case "P": // previous slide
					if k.Modifiers == 0 || k.Modifiers == key.ModCtrl {
						slidenumber--
					}
				case "N": // next slide
					if k.Modifiers == 0 || k.Modifiers == key.ModCtrl {
						slidenumber++
					}
				case "^", "⇱": // first slide
					slidenumber = 0
				case "$", "⇲": // last slide
					slidenumber = ns
				case "G":
					gridstate = !gridstate
				case key.NameSpace, "⏎":
					if k.Modifiers == 0 {
						slidenumber++
					} else {
						slidenumber--
					}
				case key.NameRightArrow, key.NamePageDown, key.NameDownArrow, "K":
					slidenumber++
				case key.NameLeftArrow, key.NamePageUp, key.NameUpArrow, "J":
					slidenumber--
				case key.NameEscape, "Q":
					os.Exit(0)
				}
			}
		}
		if p, ok := ev.(pointer.Event); ok {
			switch p.Type {
			case pointer.Press:
				switch p.Buttons {
				case pointer.ButtonPrimary:
					slidenumber++
				case pointer.ButtonSecondary:
					slidenumber--
				case pointer.ButtonTertiary:
					slidenumber = 0
				}
				pressed = true
			}
		}
	}

}

func slidedeck(s string, initpage int, filename, pagesize string) {
	width, height := pagedim(pagesize)
	deck, err := readDeck(filename, width, height)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	// set initial values
	nslides := len(deck.Slide) - 1
	if initpage > nslides+1 || initpage < 1 {
		initpage = 1
	}
	slidenumber = initpage - 1
	gridstate = false
	w := app.NewWindow(app.Title(s), app.Size(unit.Dp(width), unit.Dp(height)))
	for {
		ev := <-w.Events()
		switch e := ev.(type) {
		case system.DestroyEvent:
			os.Exit(0)
		case system.FrameEvent:
			canvas := gc.NewCanvas(float32(e.Size.X), float32(e.Size.Y), system.FrameEvent{})
			key.InputOp{Tag: pressed}.Add(canvas.Context.Ops)
			pointer.InputOp{Tag: pressed, Grab: false, Types: pointer.Press}.Add(canvas.Context.Ops)
			if slidenumber > nslides {
				slidenumber = 0
			}
			if slidenumber < 0 {
				slidenumber = nslides
			}
			showslide(canvas, &deck, slidenumber)
			if gridstate {
				ngrid(canvas, 5, 1, gc.ColorLookup(deck.Slide[slidenumber].Fg))
			}
			kbpointer(e.Queue, nslides)
			e.Frame(canvas.Context.Ops)
		}
	}
}
