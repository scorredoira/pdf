package pdf

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image/png"
	"strconv"
	"strings"

	"github.com/beevik/etree"
	"github.com/skip2/go-qrcode"
)

const (
	A4_WIDTH  = 595
	A4_HEIGHT = 842
)

// Parse parses an XML document into a PDF AST
func Parse(docElement *etree.Document) (*Document, error) {
	doc := &Document{
		Widget: Widget{
			Type:     "document",
			Rect:     Rect{X: 0, Y: 0},
			Children: []*Widget{},
		},
	}

	root := docElement.Root()
	if root == nil {
		return nil, fmt.Errorf("document has no root element")
	}

	// Allow implicit document and page
	if root.Tag != "document" {
		el := etree.NewElement("document")
		if root.Tag != "page" {
			page := el.CreateElement("page")
			for _, child := range root.ChildElements() {
				page.AddChild(child)
			}
		} else {
			for _, child := range root.ChildElements() {
				el.AddChild(child)
			}
		}
		root = el
	}

	// Parse document attributes
	color := getAttrValue(root, "color", "#222")
	doc.Color = parseColor(color)

	doc.FontFamily = getAttrValue(root, "fontFamily", "roboto")
	doc.FontSize = parseFloatAttr(root, "fontSize", 14)
	doc.LineHeight = parseFloatAttr(root, "lineHeight", doc.FontSize)
	doc.LineSpace = parseFloatAttr(root, "lineSpace", doc.FontSize/5)
	doc.Width = parseFloatAttr(root, "width", A4_WIDTH)
	doc.Height = parseFloatAttr(root, "height", A4_HEIGHT)

	// Parse pages
	for _, child := range root.ChildElements() {
		page, err := parsePage(child, doc)
		if err != nil {
			return nil, err
		}
		doc.Pages = append(doc.Pages, page)
		doc.Children = append(doc.Children, &page.Widget)
	}

	return doc, nil
}

func parsePage(el *etree.Element, doc *Document) (*Page, error) {
	if el.Tag != "page" {
		return nil, fmt.Errorf("expected page element, got %s", el.Tag)
	}

	widget, err := parseWidget(el)
	if err != nil {
		return nil, err
	}

	page := &Page{
		Widget: *widget,
	}

	if page.Width == 0 {
		page.Width = doc.Width
	}
	if page.Height == 0 {
		page.Height = doc.Height
	}

	page.ResetPageNumbers = parseBoolAttr(el, "resetPageNumbers", false)
	page.Children = []*Widget{}

	for _, child := range el.Child {
		w, err := parseToken(child, page, true)
		if err != nil {
			return nil, err
		}
		if w != nil {
			page.Children = append(page.Children, w)
		}
	}

	return page, nil
}

func parseToken(tk etree.Token, page *Page, textAsWidget bool) (*Widget, error) {
	switch t := tk.(type) {
	case *etree.CharData:
		text := t.Data
		if strings.TrimSpace(text) == "" {
			return nil, nil
		}
		if textAsWidget {
			lines := splitClean(text, "\n")
			return &Widget{
				Type:       "div",
				ValueLines: lines,
				Option:     page.Option,
			}, nil
		}
		return nil, nil

	case *etree.Element:
		return parseElement(t, page)

	default:
		return nil, nil
	}
}

func parseElement(el *etree.Element, page *Page) (*Widget, error) {
	switch el.Tag {
	case "header":
		header, err := parseDiv(el)
		if err != nil {
			return nil, err
		}
		header.Type = "div"
		page.Header = &header.Widget
		return nil, nil

	case "footer":
		footer, err := parseDiv(el)
		if err != nil {
			return nil, err
		}
		footer.Type = "div"
		page.Footer = &footer.Widget
		return nil, nil

	case "div":
		div, err := parseDiv(el)
		if err != nil {
			return nil, err
		}
		// Copy div-specific fields to Widget for 1:1 TypeScript compatibility
		div.Widget.Direction = div.Direction
		return &div.Widget, nil

	case "image":
		img, err := parseImage(el)
		if err != nil {
			return nil, err
		}
		// Copy image-specific fields to Widget for 1:1 TypeScript compatibility
		img.Widget.Data = img.Data
		img.Widget.ImgWidth = img.ImgWidth
		img.Widget.ImgHeight = img.ImgHeight
		img.Widget.ImgMaxWidth = img.ImgMaxWidth
		img.Widget.ImgMaxHeight = img.ImgMaxHeight
		return &img.Widget, nil

	case "qr":
		qr, err := parseQR(el)
		if err != nil {
			return nil, err
		}
		// Copy image-specific fields to Widget for 1:1 TypeScript compatibility
		qr.Image.Widget.Data = qr.Data
		qr.Image.Widget.ImgWidth = qr.ImgWidth
		qr.Image.Widget.ImgHeight = qr.ImgHeight
		qr.Image.Widget.ImgMaxWidth = qr.ImgMaxWidth
		qr.Image.Widget.ImgMaxHeight = qr.ImgMaxHeight
		return &qr.Image.Widget, nil

	case "table":
		table, err := parseTable(el)
		if err != nil {
			return nil, err
		}
		// Copy table-specific fields to Widget for 1:1 TypeScript compatibility
		table.Widget.Columns = table.Columns
		table.Widget.CarryColumn = table.CarryColumn
		table.Widget.CarryLast = table.CarryLast
		table.Widget.CarryNext = table.CarryNext
		if table.CarryHeader != nil {
			table.Widget.CarryHeader = &table.CarryHeader.Widget
		}
		if table.CarryFooter != nil {
			table.Widget.CarryFooter = &table.CarryFooter.Widget
		}
		table.Widget.AlternateColor = table.AlternateColor
		table.Widget.BreakMargin = table.BreakMargin
		table.Widget.CellBorder = table.CellBorder
		table.Widget.CellPadding = table.CellPadding
		return &table.Widget, nil

	default:
		return nil, fmt.Errorf("unknown widget type: %s", el.Tag)
	}
}

func parseDiv(el *etree.Element) (*Div, error) {
	widget, err := parseWidget(el)
	if err != nil {
		return nil, err
	}

	div := &Div{
		Widget: *widget,
	}

	if dir := getAttrValue(el, "direction", ""); dir != "" {
		div.Direction = Direction(dir)
	}

	div.Children = []*Widget{}

	// Handle text content and children
	for _, child := range el.Child {
		switch c := child.(type) {
		case *etree.CharData:
			text := c.Data
			if strings.TrimSpace(text) != "" {
				div.Value = text
				div.ValueLines = splitClean(text, "\n")
			}
		case *etree.Element:
			w, err := parseElement(c, nil)
			if err != nil {
				return nil, err
			}
			if w != nil {
				div.Children = append(div.Children, w)
			}
		}
	}

	return div, nil
}

func parseTable(el *etree.Element) (*Table, error) {
	widget, err := parseWidget(el)
	if err != nil {
		return nil, err
	}

	if widget.Padding != nil {
		return nil, fmt.Errorf("tables cannot have padding, use cellPadding instead")
	}

	table := &Table{
		Widget:      *widget,
		CarryColumn: -1, // Initialize to -1 to indicate no carry column
	}

	table.Border = parseBorder(el, "border")
	table.CellBorder = parseBorder(el, "cellBorder")

	table.CellPadding = parsePadding(el, "cellPadding")

	table.BreakMargin = parseFloatAttr(el, "breakMargin", 0)

	if alternateColor := getAttrValue(el, "alternateColor", ""); alternateColor != "" {
		table.AlternateColor = parseColor(alternateColor)
	}

	table.Columns = []*TableColumn{}
	table.Children = []*Widget{}

	anyRow := false
	for _, child := range el.ChildElements() {
		switch child.Tag {
		case "carryHeader":
			header, err := parseDiv(child)
			if err != nil {
				return nil, err
			}
			table.CarryHeader = header

		case "carryFooter":
			footer, err := parseDiv(child)
			if err != nil {
				return nil, err
			}
			table.CarryFooter = footer

		case "columns":
			if err := parseTableColumns(child, table); err != nil {
				return nil, err
			}

		case "row":
			if !anyRow {
				anyRow = true
				addTableHeaderColumns(table)
			}
			row, err := parseTableRow(child, table)
			if err != nil {
				return nil, err
			}
			// Copy row-specific fields to Widget for 1:1 TypeScript compatibility
			row.Widget.Direction = row.Direction
			table.Children = append(table.Children, &row.Widget)
		}
	}

	if table.AlternateColor != nil {
		setAlternateColor(table)
	}

	return table, nil
}

func parseTableColumns(el *etree.Element, table *Table) error {
	index := 0
	for _, child := range el.ChildElements() {
		if child.Tag != "column" {
			continue
		}
		col, err := parseTableColumn(child, table)
		if err != nil {
			return err
		}
		table.Columns = append(table.Columns, col)

		if col.Carry {
			table.CarryColumn = index
		}
		index++
	}
	return nil
}

func parseTableColumn(el *etree.Element, table *Table) (*TableColumn, error) {
	widget, err := parseWidget(el)
	if err != nil {
		return nil, err
	}

	col := &TableColumn{
		Widget: *widget,
	}
	col.Direction = DirectionRow

	if col.Padding == nil {
		col.Padding = table.CellPadding
	}

	if col.Border == nil {
		col.Border = table.CellBorder
	}

	col.Carry = parseBoolAttr(el, "carry", false)
	col.Children = []*Widget{}

	// Handle text content
	text := el.Text()
	if strings.TrimSpace(text) != "" {
		col.Value = text
		col.ValueLines = splitClean(text, "\n")
	}

	return col, nil
}

func parseTableRow(el *etree.Element, table *Table) (*TableRow, error) {
	widget, err := parseWidget(el)
	if err != nil {
		return nil, err
	}

	row := &TableRow{
		Widget: *widget,
	}
	row.Direction = DirectionRow
	row.Children = []*Widget{}

	index := 0
	for _, child := range el.ChildElements() {
		if child.Tag != "cell" {
			continue
		}
		cell, err := parseTableCell(child, table, index)
		if err != nil {
			return nil, err
		}
		row.Children = append(row.Children, &cell.Widget)
		index++
	}

	return row, nil
}

func parseTableCell(el *etree.Element, table *Table, index int) (*TableCell, error) {
	widget, err := parseWidget(el)
	if err != nil {
		return nil, err
	}

	cell := &TableCell{
		Widget: *widget,
	}
	cell.Direction = DirectionRow

	if index < len(table.Columns) {
		column := table.Columns[index]
		cell.Align = column.Align
		if cell.Option == nil {
			cell.Option = column.Option
		}
	}

	if cell.Padding == nil {
		cell.Padding = table.CellPadding
	}

	if cell.Border == nil {
		cell.Border = table.CellBorder
	}

	cell.Children = []*Widget{}

	// Handle text content and children
	for _, child := range el.Child {
		switch c := child.(type) {
		case *etree.CharData:
			text := c.Data
			if strings.TrimSpace(text) != "" {
				cell.Value = text
				cell.ValueLines = splitClean(text, "\n")
			}
		case *etree.Element:
			w, err := parseElement(c, nil)
			if err != nil {
				return nil, err
			}
			if w != nil {
				cell.Children = append(cell.Children, w)
			}
		}
	}

	// Copy cell-specific fields to Widget
	cell.Widget.IsHeader = cell.IsHeader
	cell.Widget.Padding = cell.Padding
	cell.Widget.Border = cell.Border
	cell.Widget.Direction = cell.Direction

	return cell, nil
}

func parseImage(el *etree.Element) (*Image, error) {
	widget, err := parseWidget(el)
	if err != nil {
		return nil, err
	}

	img := &Image{
		Widget: *widget,
	}

	img.Data = getAttrValue(el, "data", "")
	img.ImgWidth = parseFloatAttr(el, "imgWidth", 0)
	img.ImgHeight = parseFloatAttr(el, "imgHeight", 0)
	img.ImgMaxWidth = parseFloatAttr(el, "imgMaxWidth", 0)
	img.ImgMaxHeight = parseFloatAttr(el, "imgMaxHeight", 0)

	// Also set these in the Widget fields
	img.Widget.Data = img.Data
	img.Widget.ImgWidth = img.ImgWidth
	img.Widget.ImgHeight = img.ImgHeight
	img.Widget.ImgMaxWidth = img.ImgMaxWidth
	img.Widget.ImgMaxHeight = img.ImgMaxHeight

	if img.Width == 0 && img.ImgWidth > 0 {
		img.Width = img.ImgWidth
	}

	if img.Height == 0 && img.ImgHeight > 0 {
		img.Height = img.ImgHeight
	}

	if img.Data != "" {
		encoder := base64.RawStdEncoding
		decoded, err := encoder.DecodeString(img.Data)
		if err != nil {
			return nil, fmt.Errorf("failed to decode image data: %w", err)
		}
		img.Bytes = decoded
		// Also set in Widget
		img.Widget.Bytes = decoded
	}

	return img, nil
}

func parseQR(el *etree.Element) (*QRCode, error) {
	imgWidget, err := parseImage(el)
	if err != nil {
		return nil, err
	}

	qr := &QRCode{
		Image: *imgWidget,
	}

	qr.Code = getAttrValue(el, "code", "")
	qr.Level = getAttrValue(el, "level", "high")
	qr.Version = int(parseFloatAttr(el, "version", 9))
	qr.Size = int(parseFloatAttr(el, "size", 150))

	if qr.Size == 0 {
		qr.Size = 150
	}

	qr.Width = float64(qr.Size)
	qr.Height = float64(qr.Size)

	// Generate QR code bytes if code is provided
	if qr.Code != "" {
		// Map level string to qrcode.RecoveryLevel
		var level qrcode.RecoveryLevel
		switch strings.ToLower(qr.Level) {
		case "low":
			level = qrcode.Low
		case "medium":
			level = qrcode.Medium
		case "high":
			level = qrcode.High
		case "highest":
			level = qrcode.Highest
		default:
			level = qrcode.High
		}

		// Generate QR code
		qrCode, err := qrcode.New(qr.Code, level)
		if err != nil {
			return nil, fmt.Errorf("failed to generate QR code: %w", err)
		}

		// Convert to PNG bytes
		qrCode.DisableBorder = true
		img := qrCode.Image(qr.Size)

		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			return nil, fmt.Errorf("failed to encode QR code as PNG: %w", err)
		}

		qr.Bytes = buf.Bytes()
		qr.Image.Bytes = buf.Bytes()
		qr.Image.Widget.Bytes = buf.Bytes()

		// Set image dimensions
		qr.ImgWidth = float64(qr.Size)
		qr.ImgHeight = float64(qr.Size)
		qr.Image.ImgWidth = float64(qr.Size)
		qr.Image.ImgHeight = float64(qr.Size)
		qr.Image.Widget.ImgWidth = float64(qr.Size)
		qr.Image.Widget.ImgHeight = float64(qr.Size)
	}

	return qr, nil
}

func parseWidget(el *etree.Element) (*Widget, error) {
	w := &Widget{
		Type: el.Tag,
	}

	w.ID = getAttrValue(el, "id", "")
	w.Rect.X = parseFloatAttr(el, "x", 0)
	w.Rect.Y = parseFloatAttr(el, "y", 0)
	w.Width = parseFloatAttr(el, "width", 0)
	w.Height = parseFloatAttr(el, "height", 0)
	w.Right = parseFloatAttr(el, "right", 0)
	w.Bottom = parseFloatAttr(el, "bottom", 0)
	w.LineHeight = parseFloatAttr(el, "lineHeight", 0)
	w.Gap = parseFloatAttr(el, "gap", 0)

	if dir := getAttrValue(el, "direction", ""); dir != "" {
		w.Direction = Direction(dir)
	}

	w.Hidden = parseBoolAttr(el, "hidden", false)
	w.Wrap = parseBoolAttr(el, "wrap", false)

	w.Padding = parsePadding(el, "padding")
	w.Margin = parseMargin(el)

	// Parse alignment exactly like TypeScript: parseAlign(el, w)
	parseAlign(el, w)

	parseFont(el, w)

	w.Border = parseBorder(el, "border")

	if bgColor := getAttrValue(el, "backgroundColor", ""); bgColor != "" {
		w.BackgroundColor = parseColor(bgColor)
	}

	if color := getAttrValue(el, "color", ""); color != "" {
		w.Color = parseColor(color)
	}

	if strokeColor := getAttrValue(el, "strokeColor", ""); strokeColor != "" {
		w.StrokeColor = parseColor(strokeColor)
	}

	// Parse option (align) - this is handled by parseAlign function above
	// The Option field is set in parseAlign function

	return w, nil
}

func parsePadding(el *etree.Element, typ string) *Box {
	if typ == "" {
		typ = "padding"
	}

	v := getAttrValue(el, typ, "")
	if v == "" {
		return nil
	}

	box := parseBox(v)

	// Override with specific attributes
	if v := getAttrValue(el, typ+"Top", ""); v != "" {
		box.Top = parseFloat(v)
	}
	if v := getAttrValue(el, typ+"Right", ""); v != "" {
		box.Right = parseFloat(v)
	}
	if v := getAttrValue(el, typ+"Bottom", ""); v != "" {
		box.Bottom = parseFloat(v)
	}
	if v := getAttrValue(el, typ+"Left", ""); v != "" {
		box.Left = parseFloat(v)
	}

	return box
}

func parseMargin(el *etree.Element) *Box {
	var margin *Box

	if v := getAttrValue(el, "margin", ""); v != "" {
		margin = parseBox(v)
	}

	// Override with specific attributes
	if v := getAttrValue(el, "marginTop", ""); v != "" {
		if margin == nil {
			margin = &Box{}
		}
		margin.Top = parseFloat(v)
	}
	if v := getAttrValue(el, "marginRight", ""); v != "" {
		if margin == nil {
			margin = &Box{}
		}
		margin.Right = parseFloat(v)
	}
	if v := getAttrValue(el, "marginBottom", ""); v != "" {
		if margin == nil {
			margin = &Box{}
		}
		margin.Bottom = parseFloat(v)
	}
	if v := getAttrValue(el, "marginLeft", ""); v != "" {
		if margin == nil {
			margin = &Box{}
		}
		margin.Left = parseFloat(v)
	}

	return margin
}

func parseFont(el *etree.Element, w *Widget) {
	if v := getAttrValue(el, "fontFamily", ""); v != "" {
		w.FontFamily = v
	}
	if v := getAttrValue(el, "fontSize", ""); v != "" {
		w.FontSize = parseFloat(v)
	}
	if v := getAttrValue(el, "bold", ""); v != "" {
		w.Bold = parseBool(v)
	}
}

func parseBox(v string) *Box {
	box := &Box{}
	parts := strings.Fields(v)

	switch len(parts) {
	case 1:
		all := parseFloat(parts[0])
		box.Top = all
		box.Right = all
		box.Bottom = all
		box.Left = all
	case 2:
		h := parseFloat(parts[0])
		w := parseFloat(parts[1])
		box.Top = h
		box.Right = w
		box.Bottom = h
		box.Left = w
	case 4:
		box.Top = parseFloat(parts[0])
		box.Right = parseFloat(parts[1])
		box.Bottom = parseFloat(parts[2])
		box.Left = parseFloat(parts[3])
	}

	return box
}

func parseBorder(el *etree.Element, typ string) *Border {
	if typ == "" {
		typ = "border"
	}

	var border *Border
	borderRadius := parseFloatAttr(el, typ+"Radius", 0)

	if v := getAttrValue(el, typ, ""); v != "" {
		style := parseLineStyle(v)
		border = &Border{
			Top:    style,
			Right:  style,
			Bottom: style,
			Left:   style,
			Radius: borderRadius,
		}
	}

	// Override with specific sides
	if v := getAttrValue(el, typ+"Top", ""); v != "" {
		if border == nil {
			border = &Border{Radius: borderRadius}
		}
		border.Top = parseLineStyle(v)
	}
	if v := getAttrValue(el, typ+"Right", ""); v != "" {
		if border == nil {
			border = &Border{Radius: borderRadius}
		}
		border.Right = parseLineStyle(v)
	}
	if v := getAttrValue(el, typ+"Bottom", ""); v != "" {
		if border == nil {
			border = &Border{Radius: borderRadius}
		}
		border.Bottom = parseLineStyle(v)
	}
	if v := getAttrValue(el, typ+"Left", ""); v != "" {
		if border == nil {
			border = &Border{Radius: borderRadius}
		}
		border.Left = parseLineStyle(v)
	}

	if borderRadius > 0 && border == nil {
		border = &Border{Radius: borderRadius}
	}

	return border
}

func parseLineStyle(v string) *LineStyle {
	style := &LineStyle{}
	parts := strings.Fields(v)

	switch len(parts) {
	case 1:
		style.Style = parseLineStyleValue(parts[0])
	case 2:
		style.Style = parseLineStyleValue(parts[0])
		style.Width = parseFloat(parts[1])
	case 3:
		style.Style = parseLineStyleValue(parts[0])
		style.Width = parseFloat(parts[1])
		style.Color = parseColor(parts[2])
	}

	return style
}

func parseLineStyleValue(v string) string {
	switch v {
	case "0":
		return "none"
	case "1":
		return "solid"
	default:
		return v
	}
}

func parseColor(v string) *Color {
	if !strings.HasPrefix(v, "#") {
		return nil
	}

	v = v[1:]

	var r, g, b int

	switch len(v) {
	case 3:
		r = parseHex(string(v[0]) + string(v[0]))
		g = parseHex(string(v[1]) + string(v[1]))
		b = parseHex(string(v[2]) + string(v[2]))
	case 6:
		r = parseHex(v[0:2])
		g = parseHex(v[2:4])
		b = parseHex(v[4:6])
	default:
		return nil
	}

	return &Color{R: r, G: g, B: b}
}

// parseAlign - exactly like TypeScript parseAlign function
func parseAlign(el *etree.Element, w *Widget) {
	w.Align = getAttrValue(el, "align", "")
	if w.Align == "" {
		return
	}

	var opt *CellOption
	if w.Option != nil {
		opt = w.Option
	} else {
		opt = &CellOption{} // equivalent to pdf.newCellOption()
		w.Option = opt
	}

	opt.Align = parseAlignOption(w.Align, 0)
}

// parseAlignOption - exactly like TypeScript parseAlignOption function
func parseAlignOption(v string, defaultValue int) int {
	align := defaultValue

	if v != "" {
		items := strings.Fields(v) // equivalent to v.splitClean(" ")
		for _, item := range items {
			switch item {
			case "left":
				align = align | LEFT
			case "center":
				align = align | CENTER
			case "right":
				align = align | RIGHT
			case "top":
				align = align | TOP
			case "middle":
				align = align | MIDDLE
			case "bottom":
				align = align | BOTTOM
			}
		}
	}

	return align
}

func parseHex(v string) int {
	n, _ := strconv.ParseInt(v, 16, 32)
	return int(n)
}

// Helper functions

func addTableHeaderColumns(table *Table) {
	if len(table.Columns) == 0 {
		return
	}

	row := &TableRow{
		Widget: Widget{
			Type:      "row",
			Direction: DirectionRow,
			Children:  []*Widget{},
		},
	}

	for _, col := range table.Columns {
		cell := &TableCell{
			Widget: col.Widget,
		}
		cell.Type = "cell"
		cell.IsHeader = true
		if !cell.Bold {
			cell.Bold = true
		}
		// Copy cell-specific fields to Widget
		cell.Widget.IsHeader = cell.IsHeader
		cell.Widget.Bold = cell.Bold
		cell.Widget.Direction = cell.Direction
		row.Children = append(row.Children, &cell.Widget)
	}

	// Copy row-specific fields to Widget for header row
	row.Widget.Direction = row.Direction
	table.Children = append([]*Widget{&row.Widget}, table.Children...)
}

func setAlternateColor(table *Table) {
	for i := 1; i < len(table.Children); i++ {
		if i%2 == 0 {
			row := table.Children[i]
			for _, cell := range row.Children {
				if cell.BackgroundColor == nil {
					cell.BackgroundColor = table.AlternateColor
				}
			}
		}
	}
}

func splitClean(s, sep string) []string {
	parts := strings.Split(s, sep)
	result := []string{}
	for _, part := range parts {
		if cleaned := strings.TrimSpace(part); cleaned != "" {
			result = append(result, cleaned)
		}
	}
	return result
}

func getAttrValue(el *etree.Element, name, defaultValue string) string {
	v := el.SelectAttrValue(name, defaultValue)
	return v
}

func parseFloatAttr(el *etree.Element, name string, defaultValue float64) float64 {
	v := getAttrValue(el, name, "")
	if v == "" {
		return defaultValue
	}
	return parseFloat(v)
}

func parseBoolAttr(el *etree.Element, name string, defaultValue bool) bool {
	v := getAttrValue(el, name, "")
	if v == "" {
		return defaultValue
	}
	return parseBool(v)
}

func parseFloat(v string) float64 {
	f, _ := strconv.ParseFloat(v, 64)
	return f
}

func parseBool(v string) bool {
	return v == "true" || v == "1"
}
