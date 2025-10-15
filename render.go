package pdf

import (
	_ "embed"
	"fmt"
	"io"
	"strings"

	"github.com/beevik/etree"
	"github.com/signintech/gopdf"
)

//go:embed assets/fonts/Roboto-Regular.ttf
var RobotoRegular []byte

//go:embed assets/fonts/Roboto-Medium.ttf
var RobotoBold []byte

// NewRendererFromXML creates a new PDF renderer from XML string
func NewRendererFromXML(str string) (*Renderer, error) {
	return newRenderer(str)
}

// newRenderer creates a new PDF renderer from XML string
func newRenderer(str string) (*Renderer, error) {

	xmlDoc := etree.NewDocument()
	if err := xmlDoc.ReadFromString(str); err != nil {
		return nil, err
	}

	// Parse to PDF document
	document, err := Parse(xmlDoc)
	if err != nil {
		return nil, err
	}

	doc := SetLayout(document, nil)
	if doc == nil {
		return nil, fmt.Errorf("failed to set layout")
	}

	return NewRenderer(doc, str)
}

// NewRenderer creates a new PDF renderer from a parsed document
func NewRenderer(doc *Document, source string) (*Renderer, error) {
	// Create PDF configuration
	config := gopdf.Config{
		PageSize: *gopdf.PageSizeA4,
	}

	pdf := &gopdf.GoPdf{}

	pdf.Start(config)

	// Load embedded fonts
	if err := pdf.AddTTFFontData("roboto", RobotoRegular); err != nil {
		return nil, fmt.Errorf("failed to load roboto font (required for text rendering): %w", err)
	}

	if err := pdf.AddTTFFontData("robotoBold", RobotoBold); err != nil {
		// Bold font is optional, continue with regular font only
	}

	return &Renderer{
		pdf:      pdf,
		doc:      doc,
		rendered: false,
		source:   source,
	}, nil
}

type Renderer struct {
	pdf      *gopdf.GoPdf
	doc      *Document
	rendered bool
	source   string
}

func (r *Renderer) GetDocument() *Document {
	return r.doc
}

func (r *Renderer) GetSource() string {
	return r.source
}

func (r *Renderer) Render() error {
	if r.rendered {
		return nil
	}

	for _, page := range r.doc.Pages {
		if err := r.renderPage(page); err != nil {
			return err
		}
	}

	r.rendered = true
	return nil
}

func (r *Renderer) renderPage(page *Page) error {
	r.pdf.AddPage()

	// Render background color
	r.renderColors(&page.Widget)

	// Render header if exists
	if page.Header != nil {
		if err := r.renderWidget(page.Header); err != nil {
			return err
		}
	}

	// Render all child widgets
	for _, child := range page.Children {
		if err := r.renderWidget(child); err != nil {
			return err
		}
	}

	// Render footer if exists
	if page.Footer != nil {
		if err := r.renderWidget(page.Footer); err != nil {
			return err
		}
	}

	return nil
}

func (r *Renderer) renderWidget(w *Widget) error {
	switch w.Type {
	case "div":
		return r.renderDiv(w)
	case "table":
		return r.renderTable(w)
	case "image", "qr":
		return r.renderImage(w)
	default:
		return fmt.Errorf("unknown widget type: %s", w.Type)
	}
}

func (r *Renderer) renderDiv(w *Widget) error {
	r.renderColors(w)
	r.renderValue(w)

	// Render children
	for _, child := range w.Children {
		if err := r.renderWidget(child); err != nil {
			return err
		}
	}

	r.renderBorder(w)
	return nil
}

func (r *Renderer) renderTable(w *Widget) error {
	r.renderColors(w)

	// Render carryHeader
	if w.CarryLast != nil && w.CarryHeader != nil {
		y := w.Calculated.InnerY - w.Calculated.LineHeight
		if w.CarryHeader.Margin != nil {
			y -= w.CarryHeader.Margin.Bottom
		}
		if err := r.renderTableCarry(w.CarryHeader, w, y); err != nil {
			return err
		}
	}

	// Render table rows
	for _, child := range w.Children {
		if err := r.renderTableRow(child); err != nil {
			return err
		}
	}

	// Render carryFooter
	if w.CarryNext != nil && w.CarryFooter != nil {
		y := w.Calculated.InnerY + w.Calculated.InnerHeight
		if w.CarryFooter.Margin != nil {
			y += w.CarryFooter.Margin.Top
		}
		if err := r.renderTableCarry(w.CarryFooter, w, y); err != nil {
			return err
		}
	}

	return nil
}

// renderTableCarry renders table carry headers and footers
func (r *Renderer) renderTableCarry(w *Widget, table *Widget, y float64) error {
	if w.Option == nil {
		w.Option = &CellOption{
			Align: RIGHT,
		}
	}

	if w.Calculated == nil {
		w.Calculated = &CalculatedInfo{}
	}
	w.Calculated.X = table.Calculated.InnerX
	w.Calculated.InnerX = table.Calculated.InnerX
	w.Calculated.Y = y
	w.Calculated.InnerY = w.Calculated.Y
	w.Calculated.InnerWidth = table.Calculated.InnerWidth
	w.Calculated.Width = table.Calculated.Width
	w.Calculated.OuterWidth = table.Calculated.OuterWidth

	if w.Padding != nil {
		w.Calculated.InnerX += w.Padding.Left
		w.Calculated.InnerY += w.Padding.Top
		w.Calculated.InnerWidth -= w.Padding.Left + w.Padding.Right
		w.Calculated.Height += w.Padding.Top + w.Padding.Bottom
	}

	if w == table.CarryHeader && table.CarryLast != nil {
		carryValue := fmt.Sprintf("%.2f", *table.CarryLast)
		for i := range w.ValueLines {
			w.ValueLines[i] = strings.ReplaceAll(w.ValueLines[i], "{carry}", carryValue)
		}
	}
	if w == table.CarryFooter && table.CarryNext != nil {
		carryValue := fmt.Sprintf("%.2f", *table.CarryNext)
		for i := range w.ValueLines {
			w.ValueLines[i] = strings.ReplaceAll(w.ValueLines[i], "{carry}", carryValue)
		}
	}

	return r.renderDiv(w)
}

func (r *Renderer) renderTableRow(w *Widget) error {
	// Render table cells
	for _, child := range w.Children {
		if err := r.renderTableCell(child); err != nil {
			return err
		}
	}
	return nil
}

func (r *Renderer) renderTableCell(w *Widget) error {
	r.renderColors(w)
	r.renderValue(w)

	// Render children
	for _, child := range w.Children {
		if err := r.renderWidget(child); err != nil {
			return err
		}
	}

	r.renderBorder(w)
	return nil
}

func (r *Renderer) renderImage(w *Widget) error {
	r.renderColors(w)

	// Handle image/qr rendering
	var rect *gopdf.Rect
	if w.ImgWidth != 0 || w.ImgHeight != 0 {
		if w.ImgWidth == 0 || w.ImgHeight == 0 {
			return fmt.Errorf("image: if width or height is specified then both are required")
		}
		rect = &gopdf.Rect{
			W: w.ImgWidth,
			H: w.ImgHeight,
		}
	}

	if len(w.Bytes) > 0 {
		// Create image holder from bytes
		imgHolder, err := gopdf.ImageHolderByBytes(w.Bytes)
		if err != nil {
			return fmt.Errorf("failed to create image holder: %v", err)
		}

		// Render the image at the calculated position
		err = r.pdf.ImageByHolder(imgHolder, w.Calculated.X, w.Calculated.Y, rect)
		if err != nil {
			return fmt.Errorf("failed to render image: %v", err)
		}
	}

	r.renderBorder(w)
	return nil
}

func (r *Renderer) renderValue(w *Widget) {
	if len(w.ValueLines) == 0 {
		return
	}

	// Create cell options
	option := &CellOption{}
	if w.Option != nil {
		option.Align = w.Option.Align
	}

	// Add middle alignment for single lines
	if len(w.ValueLines) == 1 {
		option.Align |= MIDDLE
	}

	r.renderWidgetText(w, w.ValueLines, option)
}

func (r *Renderer) renderWidgetText(w *Widget, lines []string, option *CellOption) {
	// Handle text color
	var textColor *Color
	if w.Color != nil {
		textColor = w.Color
	} else if w.Calculated != nil && w.Calculated.Color != nil {
		textColor = w.Calculated.Color
	}

	if textColor != nil {
		r.pdf.SetTextColor(uint8(textColor.R), uint8(textColor.G), uint8(textColor.B))
	} else {
		// Set default black color if no color specified
		r.pdf.SetTextColor(0, 0, 0)
	}

	// Set font properties
	fontFamily := w.Calculated.FontFamily
	if fontFamily == "" {
		fontFamily = "roboto"
	}

	if w.Calculated.Bold {
		fontFamily = "robotoBold"
	}

	if err := r.pdf.SetFont(fontFamily, "", w.Calculated.FontSize); err != nil {
		r.pdf.SetFont("roboto", "", w.Calculated.FontSize)
	}

	// Get positioning values
	y := w.Calculated.Y
	width := w.Calculated.InnerWidth
	height := w.Calculated.LineHeight

	// Render each line
	for _, line := range lines {
		r.pdf.SetXY(w.Calculated.X, y)

		// Handle text width overflow
		textWidth, _ := r.pdf.MeasureTextWidth(line)
		if width < textWidth {
			// Truncate text to fit within width
			bufWidth := 0.0
			var buf []string

			for _, runeChar := range line {
				s := string(runeChar)
				charWidth, _ := r.pdf.MeasureTextWidth(s)
				bufWidth += charWidth
				if bufWidth > width {
					break
				}
				buf = append(buf, s)
			}

			if len(buf) > 0 {
				line = ""
				for _, s := range buf {
					line += s
				}
			}
		}

		// Render cell
		rect := &gopdf.Rect{W: w.Calculated.InnerWidth, H: height}

		// Convert our CellOption to gopdf.CellOption
		goCellOption := gopdf.CellOption{}
		if option != nil {
				if option.Align&LEFT != 0 {
				goCellOption.Align |= gopdf.Left
			}
			if option.Align&CENTER != 0 {
				goCellOption.Align |= gopdf.Center
			}
			if option.Align&RIGHT != 0 {
				goCellOption.Align |= gopdf.Right
			}
			if option.Align&MIDDLE != 0 {
				goCellOption.Align |= gopdf.Middle
			}
		}

		r.pdf.CellWithOption(rect, line, goCellOption)

		// Move to next line
		y += height
	}
}

func (r *Renderer) renderColors(w *Widget) {
	if w.BackgroundColor != nil {
		r.pdf.SetFillColor(uint8(w.BackgroundColor.R), uint8(w.BackgroundColor.G), uint8(w.BackgroundColor.B))

		borderRadius := float64(0)
		if w.Border != nil {
			borderRadius = w.Border.Radius
		}

		r.pdf.Rectangle(
			w.Calculated.InnerX,
			w.Calculated.InnerY,
			w.Calculated.InnerX+w.Calculated.Width,
			w.Calculated.InnerY+w.Calculated.Height,
			"F",
			borderRadius,
			10,
		)
	}
}

func (r *Renderer) renderBorder(w *Widget) {
	if w.Border == nil {
		return
	}

	x := w.Calculated.InnerX
	y := w.Calculated.InnerY
	width := w.Calculated.Width
	height := w.Calculated.Height

	// Check if all borders are present and the same
	if r.hasAllBorders(w.Border) && r.allBordersSame(w.Border) {
		// Draw full rectangle border
		if w.Border.Top.Color != nil {
			r.pdf.SetStrokeColor(uint8(w.Border.Top.Color.R), uint8(w.Border.Top.Color.G), uint8(w.Border.Top.Color.B))
		}

		if w.Border.Top.Width > 0 {
			r.pdf.SetLineWidth(w.Border.Top.Width)
		}

		r.pdf.RectFromUpperLeftWithStyle(x, y, width, height, "D")
		return
	}

	// Draw individual borders
	if w.Border.Left != nil && w.Border.Left.Style != "none" {
		r.drawLine(x, y, x, y+height, w.Border.Left)
	}

	if w.Border.Right != nil && w.Border.Right.Style != "none" {
		r.drawLine(x+width, y, x+width, y+height, w.Border.Right)
	}

	if w.Border.Top != nil && w.Border.Top.Style != "none" {
		r.drawLine(x, y, x+width, y, w.Border.Top)
	}

	if w.Border.Bottom != nil && w.Border.Bottom.Style != "none" {
		r.drawLine(x, y+height, x+width, y+height, w.Border.Bottom)
	}
}

func (r *Renderer) hasAllBorders(border *Border) bool {
	return border.Left != nil && border.Right != nil && border.Top != nil && border.Bottom != nil &&
		border.Left.Style != "none" && border.Right.Style != "none" &&
		border.Top.Style != "none" && border.Bottom.Style != "none"
}

func (r *Renderer) allBordersSame(border *Border) bool {
	if !r.hasAllBorders(border) {
		return false
	}

	return border.Left.Width == border.Right.Width &&
		border.Right.Width == border.Top.Width &&
		border.Top.Width == border.Bottom.Width &&
		r.colorsEqual(border.Left.Color, border.Right.Color) &&
		r.colorsEqual(border.Right.Color, border.Top.Color) &&
		r.colorsEqual(border.Top.Color, border.Bottom.Color)
}

func (r *Renderer) colorsEqual(c1, c2 *Color) bool {
	if c1 == nil && c2 == nil {
		return true
	}
	if c1 == nil || c2 == nil {
		return false
	}
	return c1.R == c2.R && c1.G == c2.G && c1.B == c2.B
}

func (r *Renderer) drawLine(x1, y1, x2, y2 float64, style *LineStyle) {
	if style.Color != nil {
		r.pdf.SetStrokeColor(uint8(style.Color.R), uint8(style.Color.G), uint8(style.Color.B))
	}

	if style.Width > 0 {
		r.pdf.SetLineWidth(style.Width)
	}

	r.pdf.Line(x1, y1, x2, y2)
}

// Convert functions for JSON serialization

func (r *Renderer) WriteFile(path string) error {
	if err := r.Render(); err != nil {
		return err
	}
	return r.pdf.WritePdf(path)
}

func (r *Renderer) Write(w io.Writer) error {
	if err := r.Render(); err != nil {
		return err
	}

	_, err := r.pdf.WriteTo(w)
	return err
}

func (r *Renderer) GetById(id string) *Widget {
	return r.doGetById(id, &r.doc.Widget)
}

func (r *Renderer) GetByType(typ string) *Widget {
	return r.doGetByType(typ, &r.doc.Widget)
}

func (r *Renderer) doGetById(id string, w *Widget) *Widget {
	if w.ID == id {
		return w
	}

	for _, child := range w.Children {
		if result := r.doGetById(id, child); result != nil {
			return result
		}
	}

	return nil
}

func (r *Renderer) doGetByType(typ string, w *Widget) *Widget {
	if w.Type == typ {
		return w
	}

	for _, child := range w.Children {
		if result := r.doGetByType(typ, child); result != nil {
			return result
		}
	}

	return nil
}

// Helper functions for creating built-in functions

// WriteFileFromXML renders XML to PDF file
func WriteFileFromXML(xmlStr string, path string) error {
	renderer, err := newRenderer(xmlStr)
	if err != nil {
		return err
	}

	return renderer.WriteFile(path)
}

// WriteFromXML renders XML to writer
func WriteFromXML(xmlStr string, w io.Writer) error {
	renderer, err := newRenderer(xmlStr)
	if err != nil {
		return err
	}

	return renderer.Write(w)
}
