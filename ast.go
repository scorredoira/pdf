package pdf

// Forward declaration for PdfLibDoc
type PdfLibDoc struct {
	FontSize float64
}

func (p *PdfLibDoc) MeasureTextWidth(text string) float64 {
	if text == "" {
		return 0
	}
	
	charWidthFactor := 0.45
	return float64(len(text)) * charWidthFactor * p.FontSize
}

// Alignment constants
const (
	LEFT        = 8
	TOP         = 4
	RIGHT       = 2
	BOTTOM      = 1
	CENTER      = 16
	MIDDLE      = 32
	ALL_BORDERS = 15
)

type Direction string

const (
	DirectionRow    Direction = "row"
	DirectionColumn Direction = "column"
)

// Document represents the root PDF document
type Document struct {
	Widget
	PDF      any        `json:"pdf,omitempty"`
	PdLibDoc *PdfLibDoc `json:"-"` // PDF library document for layout calculations
	Pages    []*Page    `json:"pages,omitempty"`
}

// DocumentJSON is used for JSON serialization
type DocumentJSON struct {
	// Root-level calculated field like TypeScript
	Calculated *CalculatedInfo `json:"calculated"`

	// All widget fields
	Type            string      `json:"type"`
	ID              *string     `json:"id"`
	X               float64     `json:"x"`
	Y               float64     `json:"y"`
	Width           float64     `json:"width"`
	Height          float64     `json:"height"`
	Right           *float64    `json:"right"`
	Bottom          *float64    `json:"bottom"`
	Padding         *Box        `json:"padding"`
	Margin          *Box        `json:"margin"`
	Border          *Border     `json:"border"`
	LineHeight      float64     `json:"lineHeight"`
	LineSpace       float64     `json:"lineSpace"`
	FontFamily      string      `json:"fontFamily"`
	FontSize        float64     `json:"fontSize"`
	Bold            *bool       `json:"bold"`
	Color           *Color      `json:"color"`
	BackgroundColor *Color      `json:"backgroundColor"`
	StrokeColor     *Color      `json:"strokeColor"`
	Gap             *float64    `json:"gap"`
	Direction       *Direction  `json:"direction"`
	Children        []*PageJSON `json:"children"`
	Hidden          *bool       `json:"hidden"`
	Value           *string     `json:"value"`
	ValueLines      []string    `json:"valueLines"`
	Wrap            *bool       `json:"wrap"`
	Align           *string     `json:"align"`
	Option          *CellOption `json:"option"`
	PageNumber      *int        `json:"pageNumber"`

	// Document-specific fields
	PdfLibDoc interface{} `json:"pdfLibDoc"`
}

// PageJSON represents the Page structure for JSON serialization
type PageJSON struct {
	Type            string          `json:"type"`
	ID              *string         `json:"id"`
	X               *float64        `json:"x"`
	Y               *float64        `json:"y"`
	Width           *float64        `json:"width"`
	Height          *float64        `json:"height"`
	Right           *float64        `json:"right"`
	Bottom          *float64        `json:"bottom"`
	Padding         *Box            `json:"padding"`
	Margin          *Box            `json:"margin"`
	Border          *Border         `json:"border"`
	LineHeight      *float64        `json:"lineHeight"`
	LineSpace       *float64        `json:"lineSpace"`
	FontFamily      *string         `json:"fontFamily"`
	FontSize        *float64        `json:"fontSize"`
	Bold            *bool           `json:"bold"`
	Color           *Color          `json:"color"`
	BackgroundColor *Color          `json:"backgroundColor"`
	StrokeColor     *Color          `json:"strokeColor"`
	Gap             *float64        `json:"gap"`
	Direction       *Direction      `json:"direction"`
	Children        []*WidgetJSON   `json:"children"`
	Hidden          *bool           `json:"hidden"`
	Value           *string         `json:"value"`
	ValueLines      []string        `json:"valueLines"`
	Wrap            *bool           `json:"wrap"`
	Align           *string         `json:"align"`
	Option          *CellOption     `json:"option"`
	Calculated      *CalculatedInfo `json:"calculated"`
	PageNumber      *int            `json:"pageNumber"`

	// Page-specific fields
	Header           *WidgetJSON `json:"header"`
	Footer           *WidgetJSON `json:"footer"`
	ResetPageNumbers *bool       `json:"resetPageNumbers"`
}

// WidgetJSON represents the Widget structure for JSON serialization
type WidgetJSON struct {
	Type            string          `json:"type"`
	ID              *string         `json:"id"`
	X               *float64        `json:"x"`
	Y               *float64        `json:"y"`
	Width           *float64        `json:"width"`
	Height          *float64        `json:"height"`
	Right           *float64        `json:"right"`
	Bottom          *float64        `json:"bottom"`
	Padding         *Box            `json:"padding"`
	Margin          *Box            `json:"margin"`
	Border          *Border         `json:"border"`
	LineHeight      *float64        `json:"lineHeight"`
	LineSpace       *float64        `json:"lineSpace"`
	FontFamily      *string         `json:"fontFamily"`
	FontSize        *float64        `json:"fontSize"`
	Bold            *bool           `json:"bold"`
	Color           *Color          `json:"color"`
	BackgroundColor *Color          `json:"backgroundColor"`
	StrokeColor     *Color          `json:"strokeColor"`
	Gap             *float64        `json:"gap"`
	Direction       *Direction      `json:"direction"`
	Children        []*WidgetJSON   `json:"children"`
	Hidden          *bool           `json:"hidden"`
	Value           *string         `json:"value"`
	ValueLines      []string        `json:"valueLines"`
	Wrap            *bool           `json:"wrap"`
	Align           *string         `json:"align"`
	Option          *CellOption     `json:"option"`
	Calculated      *CalculatedInfo `json:"calculated"`
	PageNumber      *int            `json:"pageNumber"`
}

// Page represents a page in the PDF
type Page struct {
	Widget
	Header           *Widget `json:"header,omitempty"`
	Footer           *Widget `json:"footer,omitempty"`
	ResetPageNumbers bool    `json:"resetPageNumbers,omitempty"`
}

// Div represents a container element
type Div struct {
	Widget
}

// Table represents a table element
type Table struct {
	Widget
	Columns        []*TableColumn `json:"columns,omitempty"`
	CellPadding    *Box           `json:"cellPadding,omitempty"`
	CellBorder     *Border        `json:"cellBorder,omitempty"`
	AlternateColor *Color         `json:"alternateColor,omitempty"`
	CarryColumn    int            `json:"carryColumn,omitempty"`
	CarryLast      *float64       `json:"carryLast,omitempty"`
	CarryNext      *float64       `json:"carryNext,omitempty"`
	CarryHeader    *Div           `json:"carryHeader,omitempty"`
	CarryFooter    *Div           `json:"carryFooter,omitempty"`
	Page           int            `json:"page,omitempty"`
	BreakMargin    float64        `json:"breakMargin,omitempty"`
}

// TableColumn represents a column definition in a table
type TableColumn struct {
	Widget
	Carry    bool `json:"carry,omitempty"`
	IsHeader bool `json:"isHeader,omitempty"`
}

// TableRow represents a row in a table
type TableRow struct {
	Widget
}

// TableCell represents a cell in a table
type TableCell struct {
	Widget
	IsHeader bool `json:"isHeader,omitempty"`
}

// Image represents an image element
type Image struct {
	Widget
	Bytes        []byte  `json:"bytes,omitempty"`
	Data         string  `json:"data,omitempty"`
	ImgWidth     float64 `json:"imgWidth,omitempty"`
	ImgHeight    float64 `json:"imgHeight,omitempty"`
	ImgMaxWidth  float64 `json:"imgMaxWidth,omitempty"`
	ImgMaxHeight float64 `json:"imgMaxHeight,omitempty"`
}

// QRCode represents a QR code element
type QRCode struct {
	Image
	Code    string `json:"code,omitempty"`
	Level   string `json:"level,omitempty"`
	Version int    `json:"version,omitempty"`
	Size    int    `json:"size,omitempty"`
}

// Widget is the base type for all PDF elements
type Widget struct {
	Rect
	Type            string          `json:"type"`
	ID              string          `json:"id,omitempty"`
	Padding         *Box            `json:"padding,omitempty"`
	Margin          *Box            `json:"margin,omitempty"`
	Border          *Border         `json:"border,omitempty"`
	LineHeight      float64         `json:"lineHeight,omitempty"`
	LineSpace       float64         `json:"lineSpace,omitempty"`
	FontFamily      string          `json:"fontFamily,omitempty"`
	FontSize        float64         `json:"fontSize,omitempty"`
	Bold            bool            `json:"bold,omitempty"`
	Color           *Color          `json:"color,omitempty"`
	BackgroundColor *Color          `json:"backgroundColor,omitempty"`
	StrokeColor     *Color          `json:"strokeColor,omitempty"`
	Gap             float64         `json:"gap,omitempty"`
	Direction       Direction       `json:"direction,omitempty"`
	Children        []*Widget       `json:"children,omitempty"`
	Hidden          bool            `json:"hidden,omitempty"`
	Value           string          `json:"value,omitempty"`
	ValueLines      []string        `json:"valueLines,omitempty"`
	Wrap            bool            `json:"wrap,omitempty"`
	Align           string          `json:"align,omitempty"`
	Option          *CellOption     `json:"option,omitempty"`
	Calculated      *CalculatedInfo `json:"calculated,omitempty"`
	PageNumber      int             `json:"pageNumber,omitempty"`

	// Table-specific fields added to Widget for carry functionality
	// This enables 1:1 translation with TypeScript without complex casting
	Columns        []*TableColumn `json:"columns,omitempty"`
	CarryColumn    int            `json:"carryColumn,omitempty"`
	CarryLast      *float64       `json:"carryLast,omitempty"`
	CarryNext      *float64       `json:"carryNext,omitempty"`
	CarryHeader    *Widget        `json:"carryHeader,omitempty"`
	CarryFooter    *Widget        `json:"carryFooter,omitempty"`
	AlternateColor *Color         `json:"alternateColor,omitempty"`
	BreakMargin    float64        `json:"breakMargin,omitempty"`
	CellBorder     *Border        `json:"cellBorder,omitempty"`
	CellPadding    *Box           `json:"cellPadding,omitempty"`
	IsHeader       bool           `json:"isHeader,omitempty"`

	// Image-specific fields for when widget.Type == "image" or "qr"
	Bytes        []byte  `json:"bytes,omitempty"`
	Data         string  `json:"data,omitempty"`
	ImgWidth     float64 `json:"imgWidth,omitempty"`
	ImgHeight    float64 `json:"imgHeight,omitempty"`
	ImgMaxWidth  float64 `json:"imgMaxWidth,omitempty"`
	ImgMaxHeight float64 `json:"imgMaxHeight,omitempty"`
}

// CellOption represents PDF cell options like in TypeScript
type CellOption struct {
	Align int `json:"align,omitempty"`
}

// CalculatedInfo contains calculated layout information
type CalculatedInfo struct {
	OuterX      float64   `json:"outerX,omitempty"`
	OuterY      float64   `json:"outerY,omitempty"`
	InnerX      float64   `json:"innerX,omitempty"`
	InnerY      float64   `json:"innerY,omitempty"`
	X           float64   `json:"x,omitempty"`
	Y           float64   `json:"y,omitempty"`
	Width       float64   `json:"width,omitempty"`
	Height      float64   `json:"height,omitempty"`
	OuterWidth  float64   `json:"outerWidth,omitempty"`
	OuterHeight float64   `json:"outerHeight,omitempty"`
	InnerWidth  float64   `json:"innerWidth,omitempty"`
	InnerHeight float64   `json:"innerHeight,omitempty"`
	LineHeight  float64   `json:"lineHeight,omitempty"`
	FontFamily  string    `json:"fontFamily,omitempty"`
	FontSize    float64   `json:"fontSize,omitempty"`
	Bold        bool      `json:"bold,omitempty"`
	Color       *Color    `json:"color,omitempty"`
	Direction   Direction `json:"direction,omitempty"`
}

// Rect represents a rectangle with position and size
type Rect struct {
	X      float64 `json:"x,omitempty"`
	Y      float64 `json:"y,omitempty"`
	Width  float64 `json:"width,omitempty"`
	Height float64 `json:"height,omitempty"`
	Right  float64 `json:"right,omitempty"`
	Bottom float64 `json:"bottom,omitempty"`
}

// Box represents padding or margin values
type Box struct {
	Bottom float64 `json:"bottom"`
	Left   float64 `json:"left"`
	Right  float64 `json:"right"`
	Top    float64 `json:"top"`
}

// Color represents an RGB color
type Color struct {
	R int `json:"r"`
	G int `json:"g"`
	B int `json:"b"`
}

// LineStyle represents the style of a line
type LineStyle struct {
	Width float64 `json:"width,omitempty"`
	Color *Color  `json:"color,omitempty"`
	Style string  `json:"style,omitempty"` // "dashed", "dotted", "solid", "none"
}

// Line represents a line to be drawn
type Line struct {
	LineStyle
	X  float64 `json:"x"`
	Y  float64 `json:"y"`
	X2 float64 `json:"x2"`
	Y2 float64 `json:"y2"`
}

// Border represents the border of a widget
type Border struct {
	Top    *LineStyle `json:"top,omitempty"`
	Right  *LineStyle `json:"right,omitempty"`
	Bottom *LineStyle `json:"bottom,omitempty"`
	Left   *LineStyle `json:"left,omitempty"`
	Radius float64    `json:"radius,omitempty"`
}

// Conversion functions for JSON serialization

// ToJSON converts Document to DocumentJSON with TypeScript-compatible structure
func (d *Document) ToJSON() *DocumentJSON {
	return &DocumentJSON{
		Calculated:      d.Calculated,
		Type:            d.Type,
		ID:              stringPtr(d.ID),
		X:               d.X,
		Y:               d.Y,
		Width:           d.Width,
		Height:          d.Height,
		Right:           floatPtr(d.Right),
		Bottom:          floatPtr(d.Bottom),
		Padding:         d.Padding,
		Margin:          d.Margin,
		Border:          d.Border,
		LineHeight:      d.LineHeight,
		LineSpace:       d.LineSpace,
		FontFamily:      d.FontFamily,
		FontSize:        d.FontSize,
		Bold:            boolPtr(d.Bold),
		Color:           d.Color,
		BackgroundColor: d.BackgroundColor,
		StrokeColor:     d.StrokeColor,
		Gap:             floatPtr(d.Gap),
		Direction:       directionPtr(d.Direction),
		Children:        convertPagesToJSON(d.Pages),
		Hidden:          boolPtr(d.Hidden),
		Value:           stringPtr(d.Value),
		ValueLines:      d.ValueLines,
		Wrap:            boolPtr(d.Wrap),
		Align:           stringPtr(d.Align),
		Option:          d.Option,
		PageNumber:      intPtr(d.PageNumber),
		PdfLibDoc:       map[string]interface{}{},
	}
}

func convertPagesToJSON(pages []*Page) []*PageJSON {
	result := make([]*PageJSON, len(pages))
	for i, page := range pages {
		result[i] = page.ToJSON()
	}
	return result
}

func (p *Page) ToJSON() *PageJSON {
	return &PageJSON{
		Type:             p.Type,
		ID:               stringPtr(p.ID),
		X:                floatPtr(p.X),
		Y:                floatPtr(p.Y),
		Width:            floatPtr(p.Width),
		Height:           floatPtr(p.Height),
		Right:            floatPtr(p.Right),
		Bottom:           floatPtr(p.Bottom),
		Padding:          p.Padding,
		Margin:           p.Margin,
		Border:           p.Border,
		LineHeight:       floatPtr(p.LineHeight),
		LineSpace:        floatPtr(p.LineSpace),
		FontFamily:       stringPtr(p.FontFamily),
		FontSize:         floatPtr(p.FontSize),
		Bold:             boolPtr(p.Bold),
		Color:            p.Color,
		BackgroundColor:  p.BackgroundColor,
		StrokeColor:      p.StrokeColor,
		Gap:              floatPtr(p.Gap),
		Direction:        directionPtr(p.Direction),
		Children:         convertWidgetsToJSON(p.Children),
		Hidden:           boolPtr(p.Hidden),
		Value:            stringPtr(p.Value),
		ValueLines:       p.ValueLines,
		Wrap:             boolPtr(p.Wrap),
		Align:            stringPtr(p.Align),
		Option:           p.Option,
		Calculated:       p.Calculated,
		PageNumber:       intPtr(p.PageNumber),
		Header:           widgetToJSON(p.Header),
		Footer:           widgetToJSON(p.Footer),
		ResetPageNumbers: boolPtr(p.ResetPageNumbers),
	}
}

func convertWidgetsToJSON(widgets []*Widget) []*WidgetJSON {
	result := make([]*WidgetJSON, len(widgets))
	for i, widget := range widgets {
		result[i] = widget.ToJSON()
	}
	return result
}

func (w *Widget) ToJSON() *WidgetJSON {
	return &WidgetJSON{
		Type:            w.Type,
		ID:              stringPtr(w.ID),
		X:               floatPtr(w.X),
		Y:               floatPtr(w.Y),
		Width:           floatPtr(w.Width),
		Height:          floatPtr(w.Height),
		Right:           floatPtr(w.Right),
		Bottom:          floatPtr(w.Bottom),
		Padding:         w.Padding,
		Margin:          w.Margin,
		Border:          w.Border,
		LineHeight:      floatPtr(w.LineHeight),
		LineSpace:       floatPtr(w.LineSpace),
		FontFamily:      stringPtr(w.FontFamily),
		FontSize:        floatPtr(w.FontSize),
		Bold:            boolPtr(w.Bold),
		Color:           w.Color,
		BackgroundColor: w.BackgroundColor,
		StrokeColor:     w.StrokeColor,
		Gap:             floatPtr(w.Gap),
		Direction:       directionPtrForWidget(w),
		Children:        convertWidgetsToJSON(w.Children),
		Hidden:          boolPtr(w.Hidden),
		Value:           stringPtr(w.Value),
		ValueLines:      w.ValueLines,
		Wrap:            boolPtr(w.Wrap),
		Align:           stringPtr(w.Align),
		Option:          w.Option,
		Calculated:      w.Calculated,
		PageNumber:      intPtr(w.PageNumber),
	}
}

func widgetToJSON(w *Widget) *WidgetJSON {
	if w == nil {
		return nil
	}
	return w.ToJSON()
}

// Helper functions to create pointers for null values
func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func floatPtr(f float64) *float64 {
	if f == 0 {
		return nil
	}
	return &f
}

func boolPtr(b bool) *bool {
	if !b {
		return nil
	}
	return &b
}

func intPtr(i int) *int {
	if i == 0 {
		return nil
	}
	return &i
}

func directionPtr(d Direction) *Direction {
	if d == "" {
		return nil
	}
	return &d
}

func directionPtrForWidget(w *Widget) *Direction {
	// Table elements use row direction by default
	if w.Type == "row" || w.Type == "cell" {
		row := DirectionRow
		return &row
	}
	return directionPtr(w.Direction)
}
