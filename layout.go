package pdf

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"sort"
	"strings"
)

// NumberFormatter interface for locale-aware number operations
type NumberFormatter interface {
	FormatNumber(format string, value float64) string
	ParseNumber(text string) (float64, error)
}

// SetLayout is the main entry point for document layout calculation
func SetLayout(document *Document, formatter NumberFormatter) *Document {
	pdLibDoc := &PdfLibDoc{
		FontSize: 7.7,
	}

	// Set the PDF document reference
	document.PdLibDoc = pdLibDoc

	layouter := &Layouter{
		doc:       document,
		formatter: formatter,
		pdLibDoc:  pdLibDoc,
	}
	layouter.setLayout()
	return document
}

// Layouter handles the PDF document layout calculations
type Layouter struct {
	pdLibDoc  *PdfLibDoc
	doc       *Document
	formatter NumberFormatter
}

// setLayout performs the main layout calculation steps
func (l *Layouter) setLayout() {
	l.initSizes(l.doc)
	l.setPositions(l.doc)
	l.splitPages(l.doc)
	l.setPageNumbers(l.doc)
	l.makeAbsolute(l.doc)
}

// setPageNumbers handles page number interpolation
func (l *Layouter) setPageNumbers(doc *Document) {
	pages := doc.Pages

	var groups [][]*Page
	var currentGroup []*Page

	for i := 0; i < len(pages); i++ {
		page := pages[i]
		if page.ResetPageNumbers || i == 0 {
			if i > 0 && len(currentGroup) > 0 {
				groups = append(groups, currentGroup)
			}
			currentGroup = []*Page{}
		}
		currentGroup = append(currentGroup, page)
	}
	// Add the last group
	if len(currentGroup) > 0 {
		groups = append(groups, currentGroup)
	}

	for _, group := range groups {
		for i := 0; i < len(group); i++ {
			page := group[i]

			current := fmt.Sprintf("%d", i+1)
			total := fmt.Sprintf("%d", len(group))

			if page.Header != nil {
				l.interpolatePageNumbers(page.Header, current, total)
			}

			if page.Footer != nil {
				l.interpolatePageNumbers(page.Footer, current, total)
			}
		}
	}
}

// interpolatePageNumbers replaces page number placeholders with actual values
func (l *Layouter) interpolatePageNumbers(w *Widget, page, pages string) {
	if w.ValueLines != nil {
		for i := 0; i < len(w.ValueLines); i++ {
			line := w.ValueLines[i]
			line = strings.ReplaceAll(line, "{page}", page)
			line = strings.ReplaceAll(line, "{pages}", pages)
			w.ValueLines[i] = line
		}
		return
	}

	for _, child := range w.Children {
		l.interpolatePageNumbers(child, page, pages)
	}
}

// makeAbsolute converts relative positions to absolute coordinates
func (l *Layouter) makeAbsolute(doc *Document) {
	for _, page := range doc.Pages {
		l.makePageAbsolute(page)
	}
}

// makePageAbsolute converts page positions to absolute coordinates
func (l *Layouter) makePageAbsolute(page *Page) {
	x := page.Calculated.X
	y := page.Calculated.Y

	if page.Header != nil {
		l.makeWidgetAbsolute(page.Header, 0, 0)
	}

	for _, w := range page.Children {
		l.makeWidgetAbsolute(w, x, y)
	}

	if page.Footer != nil {
		bottom := page.Calculated.OuterHeight - page.Footer.Calculated.OuterHeight - 1
		l.makeWidgetAbsolute(page.Footer, 0, bottom)
	}
}

// makeWidgetAbsolute converts widget positions to absolute coordinates
func (l *Layouter) makeWidgetAbsolute(w *Widget, parentX, parentY float64) {
	w.Calculated.OuterX += parentX
	w.Calculated.OuterY += parentY
	l.adjustCalculatedPosition(w)

	for _, child := range w.Children {
		l.makeWidgetAbsolute(child, w.Calculated.X, w.Calculated.Y)
	}
}

// initSizes initializes size calculations for all widgets
func (l *Layouter) initSizes(doc *Document) {
	// ensure that all items have line height etc..
	l.initCalculatedInfo(&doc.Widget, nil)

	for _, page := range doc.Pages {
		l.initPageSize(page)
	}
}

// initPageSize initializes size calculations for a page
func (l *Layouter) initPageSize(page *Page) {
	l.addjustCalculatedSize(&page.Widget)

	if page.Header != nil {
		l.initCalculatedInfo(page.Header, &page.Widget)
		l.initWidgetSize(page.Header, page.Calculated.OuterWidth)
	}

	for _, w := range page.Children {
		l.initWidgetSize(w, page.Calculated.InnerWidth)
	}

	if page.Footer != nil {
		l.initCalculatedInfo(page.Footer, &page.Widget)
		l.initWidgetSize(page.Footer, page.Calculated.OuterWidth)
	}
}

// initWidgetSize initializes size calculations for a widget
func (l *Layouter) initWidgetSize(w *Widget, innerWidth float64) {
	l.initFixedSizes(w, innerWidth)
	l.initWidgetsWidth(w, innerWidth)
	l.reflowTexts(w)
	l.initWidgetsHeight(w)
}

// initFixedSizes initializes fixed dimensions for widgets
func (l *Layouter) initFixedSizes(w *Widget, parentWidth float64) {
	switch w.Type {
	case "image", "qr":
		l.initImageSizeWidget(w)
		l.addjustCalculatedWidth(w)
		return

	case "table":
		if w.CarryHeader != nil {
			l.initCalculatedInfo(w.CarryHeader, w)
			l.initValueSize(w.CarryHeader)
		}
		if w.CarryFooter != nil {
			l.initCalculatedInfo(w.CarryFooter, w)
			l.initValueSize(w.CarryFooter)
		}

	default:
		if w.ValueLines != nil {
			l.initValueSize(w)
		}
	}

	for _, child := range w.Children {
		l.initFixedSizes(child, parentWidth)
	}
}

// initValueSize calculates size for text content
func (l *Layouter) initValueSize(w *Widget) {
	if w.Width == 0 {
		maxWidth := float64(0)
		for _, line := range w.ValueLines {
			width := l.pdLibDoc.MeasureTextWidth(line)
			if width > maxWidth {
				maxWidth = width
			}
		}
		w.Calculated.Width = maxWidth
	} else {
		w.Calculated.Width = w.Width
	}

	if w.Calculated.InnerHeight == 0 {
		lines := len(w.ValueLines)
		w.Calculated.InnerHeight = float64(lines) * w.Calculated.LineHeight
	}

	l.addjustCalculatedWidth(w)
	l.recalculateFromInnerHeight(w)
}

// splitPages handles page breaking logic
func (l *Layouter) splitPages(doc *Document) {
	var pages []*Page

	for _, page := range doc.Pages {
		splitted := l.splitPage(page)
		pages = append(pages, splitted...)
	}

	doc.Pages = pages

	// Also update doc.Children to match Pages
	doc.Children = []*Widget{}
	for _, page := range pages {
		doc.Children = append(doc.Children, &page.Widget)
	}
}

// splitPage splits a single page when content overflows
func (l *Layouter) splitPage(page *Page) []*Page {
	var pages []*Page

	children := page.Children
	sort.Slice(children, func(i, j int) bool {
		return children[i].Calculated.OuterY < children[j].Calculated.OuterY
	})

	pageInnerHeight := page.Calculated.InnerHeight
	pageBottom := pageInnerHeight

	var currentY float64
	var currentPage *Page

	for i := 0; i < len(children); i++ {
		if currentPage == nil || currentY >= pageBottom {
			currentY = 0
			currentPage = l.copyPage(page, true)
			pages = append(pages, currentPage)

			// reset Y to 0 of all childrens of the new page
			l.resetY(children[i:], 0, page.Gap)
		}

		w := children[i]

		bottom := currentY + w.Calculated.OuterHeight

		// if the widget does not fit in the space left and it is
		// shorter than a full page, move it to the next page.
		if len(currentPage.Children) > 0 && w.Type != "table" {
			if bottom > pageBottom {
				currentY = 0
				currentPage = l.copyPage(page, false)
				pages = append(pages, currentPage)

				// reset Y to 0 of all childrens of the new page
				l.resetY(children[i:], 0, page.Gap)
			}
		}

		currentPage.Children = append(currentPage.Children, w)

		var tablePageBreak bool

		// Only split tables at the page root level
		if w.Type == "table" {
			breakMargin := w.BreakMargin
			if bottom+breakMargin > pageBottom {
				result := l.splitTable(w, currentPage, currentY, page, &pages)
				currentPage = result.currentPage

				currentY = 0

				w = result.currentTable
				tablePageBreak = result.pageBreak
			}
		}

		currentY += w.Calculated.OuterHeight
		if page.Gap > 0 {
			currentY += page.Gap
		}

		if tablePageBreak && len(children) > i {
			l.resetY(children[i+1:], currentY, page.Gap)
		}
	}

	return pages
}

// resetY recalculates Y positions for widgets
func (l *Layouter) resetY(widgets []*Widget, currentY, gap float64) {
	y := currentY

	for _, w := range widgets {
		if w.Y == 0 {
			w.Calculated.OuterY = y
			l.adjustCalculatedY(w)
			y += w.Calculated.OuterHeight + gap
		}
	}
}

// splitTable handles table splitting across pages
func (l *Layouter) splitTable(w *Widget, currentPage *Page, currentY float64, page *Page, pages *[]*Page) *splitTableResult {
	margin := float64(0)
	if w.Margin != nil {
		margin = w.Margin.Top + w.Margin.Bottom
	}
	innerHeight := page.Calculated.InnerHeight - margin

	var headerRow *Widget
	if len(w.Columns) > 0 {
		headerRow = l.deepCloneWidget(w.Children[0])
	}

	// expand table
	rows := make([]*Widget, len(w.Children))
	copy(rows, w.Children)

	w.Children = nil
	table := l.deepCloneWidget(w)
	currentTable := w
	var currentRows []*Widget
	pageBreak := false

	pageIndex := 0

	carryLast := (*float64)(nil)

	// Main table splitting loop
	for {
		var index int
		found := false

		// Filter only rows that fit
		for i := 0; i < len(rows); i++ {
			row := rows[i]

			rowBottom := currentY + row.Calculated.OuterY + row.Calculated.OuterHeight

			if rowBottom > innerHeight {
				break
			}

			index = i + 1
			found = true
		}

		// No more pending rows
		if !found {
			break
		}

		currentRows = make([]*Widget, index)
		copy(currentRows, rows[:index])

		l.setAlternateColor(currentRows, table.AlternateColor)

		remainingRows := make([]*Widget, len(rows)-index)
		copy(remainingRows, rows[index:])
		rows = remainingRows

		// Readjust height
		currentTable.PageNumber = pageIndex
		currentTable.Children = currentRows

		totalHeight := float64(0)
		for _, row := range currentRows {
			totalHeight += row.Calculated.OuterHeight
		}
		currentTable.Calculated.InnerHeight = totalHeight
		l.recalculateFromInnerHeight(currentTable)

		// Implement carry column logic exactly like TypeScript
		// if (table.carryColumn) {
		//     currentTable.carryLast = carryLast
		//     let sum = this.getColumnSum(currentRows, currentTable.carryColumn)
		//     currentTable.carryNext = (carryLast ?? 0) + sum
		//     carryLast = currentTable.carryNext
		//     this.interpolateCarry(currentTable)
		// }
		if table.CarryColumn >= 0 { // Check if carry is enabled (-1 means no carry)
			currentTable.CarryLast = carryLast
			sum := l.getColumnSum(currentRows, table.CarryColumn) // Use carry column index directly
			nextValue := float64(0)
			if carryLast != nil {
				nextValue = *carryLast
			}
			nextValue += sum
			currentTable.CarryNext = &nextValue
			carryLast = &nextValue

			// DO NOT interpolate carry during layout - this should match TypeScript behavior
			// The TypeScript version leaves {carry} placeholders uninterpolated during layout
			// Interpolation happens during rendering phase in renderTableCarry
			// l.interpolateCarryWidget(currentTable)
		}

		if len(rows) == 0 {
			break
		}

		currentTable = l.deepCloneWidget(table)

		if headerRow != nil {
			newRows := make([]*Widget, len(rows)+1)
			newRows[0] = l.deepCloneWidget(headerRow)
			copy(newRows[1:], rows)
			rows = newRows
		}
		currentTable.Children = rows

		l.recalculateFromInnerHeight(currentTable)
		l.resetRowsY(currentTable)

		currentPage = l.copyPage(page, false)
		*pages = append(*pages, currentPage)
		currentPage.Children = append(currentPage.Children, currentTable)
		pageIndex++
		currentTable.PageNumber = pageIndex

		currentY = 0
		pageBreak = true
	}

	// the last one doesn't carry because there is no next
	currentTable.CarryNext = nil

	return &splitTableResult{
		currentTable: currentTable,
		currentPage:  currentPage,
		pageBreak:    pageBreak,
	}
}

type splitTableResult struct {
	currentTable *Widget
	currentPage  *Page
	pageBreak    bool
}

// setAlternateColor applies alternate row coloring
func (l *Layouter) setAlternateColor(rows []*Widget, alternateColor *Color) {
	if alternateColor == nil {
		return
	}

	for i := 1; i < len(rows); i++ {
		row := rows[i]
		if i%2 == 0 {
			for _, cell := range row.Children {
				if cell.BackgroundColor == nil {
					cell.BackgroundColor = alternateColor
				}
			}
		} else {
			for _, cell := range row.Children {
				cell.BackgroundColor = nil
			}
		}
	}
}

// getColumnSum calculates the sum of values in a specific column
func (l *Layouter) getColumnSum(rows []*Widget, column int) float64 {
	total := float64(0)

	for _, row := range rows {
		cell := row.Children[column]

		if cell.IsHeader {
			continue
		}

		if len(cell.ValueLines) == 1 {
			if l.formatter != nil {
				value, err := l.formatter.ParseNumber(cell.ValueLines[0])
				if err == nil {
					total += value
				}
			}
		}
	}

	return total
}

// resetRowsY recalculates Y positions for table rows
func (l *Layouter) resetRowsY(w *Widget) {
	y := float64(0)

	w.Calculated.OuterY = y
	l.adjustCalculatedY(w)

	for _, row := range w.Children {
		row.Calculated.OuterY = y
		l.adjustCalculatedY(row)

		y += row.Calculated.OuterHeight
	}
}

// copyPage creates a copy of a page for pagination
func (l *Layouter) copyPage(page *Page, copyReset bool) *Page {
	copy := &Page{}
	copy.Type = "page"

	if copyReset {
		copy.ResetPageNumbers = page.ResetPageNumbers
	}

	copy.FontFamily = page.FontFamily
	copy.FontSize = page.FontSize
	copy.LineHeight = page.LineHeight
	copy.LineSpace = page.LineSpace
	copy.Color = page.Color
	copy.BackgroundColor = page.BackgroundColor
	copy.Padding = page.Padding
	copy.Gap = page.Gap
	copy.Direction = page.Direction
	copy.Bold = page.Bold
	copy.Align = page.Align
	copy.StrokeColor = page.StrokeColor
	copy.Calculated = l.deepCloneCalculated(page.Calculated)
	copy.Header = l.deepCloneWidget(page.Header)
	copy.Footer = l.deepCloneWidget(page.Footer)
	copy.Children = []*Widget{}

	return copy
}

// setPositions calculates initial positions for all elements
func (l *Layouter) setPositions(doc *Document) {
	doc.Calculated.InnerX = 0
	doc.Calculated.InnerY = 0
	l.adjustCalculatedPositionFromInner(&doc.Widget)

	for _, page := range doc.Pages {
		l.setPagePositions(page)
	}
}

// setPagePositions calculates positions for page elements
func (l *Layouter) setPagePositions(page *Page) {
	page.Calculated.InnerX = 0
	page.Calculated.InnerY = 0
	l.adjustCalculatedPositionFromInner(&page.Widget)

	if page.Header != nil {
		l.setWidgetPosition(page.Header, 0, 0)
	}

	for _, w := range page.Children {
		l.setWidgetPosition(w, 0, 0)
	}

	if page.Footer != nil {
		// in makeAbsolute() is moved to the bottom
		l.setWidgetPosition(page.Footer, 0, 0)
	}
}

// setWidgetPosition calculates position for a widget and its children
func (l *Layouter) setWidgetPosition(w *Widget, parentX, parentY float64) {
	if w.X != 0 {
		w.Calculated.InnerX = w.X
	} else {
		w.Calculated.InnerX = parentX
	}

	if w.Y != 0 {
		w.Calculated.InnerY = w.Y
	} else {
		w.Calculated.InnerY = parentY
	}

	l.adjustCalculatedPositionFromInner(w)

	if len(w.Children) == 0 {
		return
	}

	x := float64(0)
	y := float64(0)

	gap := w.Gap

	direction := w.Calculated.Direction

	if w.Align == "right" && direction == "row" {
		width := float64(0)
		for _, child := range w.Children {
			width += child.Calculated.OuterWidth
		}
		if gap > 0 {
			width += gap * float64(len(w.Children)-1)
		}
		x = w.Calculated.InnerWidth - width
	}

	for _, child := range w.Children {
		if w.Align == "right" && direction == "column" {
			x = w.Calculated.InnerWidth - child.Calculated.OuterWidth
		}

		if child.X != 0 {
			child.Calculated.InnerX = child.X
		} else {
			child.Calculated.InnerX = x
		}

		if child.Y != 0 {
			child.Calculated.InnerY = child.Y
		} else {
			child.Calculated.InnerY = y
		}

		l.adjustCalculatedPositionFromInner(child)
		l.setWidgetPosition(child, child.Calculated.OuterX, child.Calculated.OuterY)

		if direction == "column" {
			y += child.Calculated.OuterHeight + gap
		} else {
			x += child.Calculated.OuterWidth + gap
		}
	}
}

// adjustCalculatedPositionFromInner converts inner positions to absolute positions
func (l *Layouter) adjustCalculatedPositionFromInner(w *Widget) {
	l.adjustCalculatedXFromInner(w)
	l.adjustCalculatedYFromInner(w)
}

// adjustCalculatedXFromInner converts inner X position to absolute X
func (l *Layouter) adjustCalculatedXFromInner(w *Widget) {
	w.Calculated.OuterX = w.Calculated.InnerX
	if w.Margin != nil {
		w.Calculated.InnerX += w.Margin.Left
	}

	w.Calculated.X = w.Calculated.InnerX
	if w.Padding != nil {
		w.Calculated.X += w.Padding.Left
	}
}

// adjustCalculatedYFromInner converts inner Y position to absolute Y
func (l *Layouter) adjustCalculatedYFromInner(w *Widget) {
	w.Calculated.OuterY = w.Calculated.InnerY
	if w.Margin != nil {
		w.Calculated.InnerY += w.Margin.Top
	}

	w.Calculated.Y = w.Calculated.InnerY
	if w.Padding != nil {
		w.Calculated.Y += w.Padding.Top
	}
}

// adjustCalculatedPosition adjusts calculated positions
func (l *Layouter) adjustCalculatedPosition(w *Widget) {
	l.adjustCalculatedX(w)
	l.adjustCalculatedY(w)
}

// adjustCalculatedX adjusts calculated X position
func (l *Layouter) adjustCalculatedX(w *Widget) {
	w.Calculated.InnerX = w.Calculated.OuterX
	if w.Margin != nil {
		w.Calculated.InnerX += w.Margin.Left
	}

	w.Calculated.X = w.Calculated.InnerX
	if w.Padding != nil {
		w.Calculated.X += w.Padding.Left
	}
}

// adjustCalculatedY adjusts calculated Y position
func (l *Layouter) adjustCalculatedY(w *Widget) {
	w.Calculated.InnerY = w.Calculated.OuterY
	if w.Margin != nil {
		w.Calculated.InnerY += w.Margin.Top
	}

	w.Calculated.Y = w.Calculated.InnerY
	if w.Padding != nil {
		w.Calculated.Y += w.Padding.Top
	}
}

// reflowTexts handles text wrapping for widgets
func (l *Layouter) reflowTexts(w *Widget) {
	if len(w.ValueLines) > 0 {
		l.wrapText(w)
		return
	}

	for _, child := range w.Children {
		l.reflowTexts(child)
	}
}

// wrapText wraps text content to fit within widget bounds
func (l *Layouter) wrapText(w *Widget) {
	var buf []string
	if w.Value != "" {
		buf = l.splitLines(w.Value, w.Calculated.FontSize, w.Calculated.InnerWidth)
	} else {
		buf = []string{}
	}

	if w.Wrap {
		if len(buf) > 0 {
			w.ValueLines = buf[:1] // slice(0, 1)
		} else {
			w.ValueLines = []string{}
		}
	} else {
		w.ValueLines = buf
	}

	if w.Height == 0 {
		lines := len(w.ValueLines)
		w.Calculated.InnerHeight = float64(lines) * w.Calculated.LineHeight
		l.recalculateFromInnerHeight(w)
	}
}

// initWidgetsHeight calculates heights for all child widgets
func (l *Layouter) initWidgetsHeight(w *Widget) {
	for _, child := range w.Children {
		l.initWidgetsHeight(child)
	}

	if w.Height == 0 {
		w.Calculated.OuterHeight = l.getHeight(w)
		l.recalculateFromOuterHeight(w)
	} else {
		l.addjustCalculatedHeight(w)
	}

	if w.Type == "table" {
		l.adjustRowsHeight(w)
	}
}

// initWidgetsWidth calculates widths for all child widgets
func (l *Layouter) initWidgetsWidth(w *Widget, parentWidth float64) {
	// If no width assigned, extend to container maximum
	if w.Width == 0 {
		w.Calculated.OuterWidth = parentWidth
		l.recalculateFromOuterWidth(w)
	} else {
		// If width exceeds maximum, adjust
		if w.Calculated.OuterWidth > parentWidth {
			w.Calculated.OuterWidth = parentWidth
			l.recalculateFromOuterWidth(w)
		} else {
			l.addjustCalculatedWidth(w)
		}
	}

	if w.Children == nil {
		return
	}

	innerWidth := w.Calculated.InnerWidth

	if w.Direction == "row" {
		sumWidth := float64(0)
		for _, child := range w.Children {
			sumWidth += child.Calculated.OuterWidth
		}

		var fixedItems []*Widget
		for _, child := range w.Children {
			if child.Width != 0 {
				fixedItems = append(fixedItems, child)
			}
		}

		gap := w.Gap * float64(len(w.Children)-1)
		if gap > 0 {
			sumWidth += gap
		}

		if sumWidth < innerWidth {
			var autoItems []*Widget
			for _, child := range w.Children {
				if child.Width == 0 {
					autoItems = append(autoItems, child)
				}
			}
			if len(autoItems) > 0 {
				remaining := innerWidth
				for _, item := range fixedItems {
					remaining -= item.Calculated.OuterWidth
				}
				remaining -= gap
				itemWidth := remaining / float64(len(autoItems))
				for _, child := range autoItems {
					l.initWidgetsWidth(child, itemWidth)
				}
			}
		} else {
			remaining := innerWidth - gap
			itemWidth := remaining / float64(len(w.Children))
			for _, child := range w.Children {
				l.initWidgetsWidth(child, itemWidth)
			}
		}
	} else {
		for _, child := range w.Children {
			l.initWidgetsWidth(child, innerWidth)
		}
	}

	if w.Type == "table" {
		l.adjustColumns(w)
	}
}

// getHeight calculates the total height of a widget
func (l *Layouter) getHeight(w *Widget) float64 {
	if len(w.Children) == 0 {
		if w.Calculated.OuterHeight != 0 {
			return w.Calculated.OuterHeight
		}
		return 0
	}

	if w.Calculated.Direction == "column" {
		height := float64(0)
		for _, child := range w.Children {
			height += l.getHeight(child)
		}
		if w.Gap > 0 {
			height += w.Gap * float64(len(w.Children)-1)
		}
		result := l.getHeightFromInnerHeight(w, height)
		return result
	}

	maxHeight := float64(0)
	for _, child := range w.Children {
		h := l.getHeight(child)
		if h > maxHeight {
			maxHeight = h
		}
	}
	result := l.getHeightFromInnerHeight(w, maxHeight)
	return result
}

// getHeightFromInnerHeight calculates outer height from inner height
func (l *Layouter) getHeightFromInnerHeight(w *Widget, innerHeight float64) float64 {
	h := innerHeight

	if w.Padding != nil {
		h += w.Padding.Top + w.Padding.Bottom
	}

	if w.Margin != nil {
		h += w.Margin.Top + w.Margin.Bottom
	}

	return h
}

// adjustColumns adjusts table column widths to fit table width
func (l *Layouter) adjustColumns(table *Widget) {
	if len(table.Children) == 0 {
		return
	}

	row := table.Children[0]
	columnCount := len(row.Children)

	columnSizes := make([]float64, columnCount)

	for _, row := range table.Children {
		for i := 0; i < columnCount; i++ {
			if len(row.Children) <= i {
				panic(fmt.Sprintf("invalid number of row cells, expected %d, got %d", i+1, len(row.Children)))
			}

			cell := row.Children[i]
			var rowMax float64
			if cell.Width != 0 {
				rowMax = cell.Width
			} else {
				rowMax = l.getOuterWidth(cell)
			}
			if rowMax > columnSizes[i] {
				columnSizes[i] = rowMax
			}
		}
	}

	totalWidth := float64(0)
	for _, size := range columnSizes {
		totalWidth += size
	}

	tableWidth := table.Calculated.InnerWidth

	ratio := tableWidth / totalWidth
	if ratio == 1 {
		return
	}

	for i := 0; i < columnCount; i++ {
		columnSizes[i] *= ratio
	}

	for _, row := range table.Children {
		for i := 0; i < columnCount; i++ {
			if len(row.Children) <= i {
				panic(fmt.Sprintf("invalid number of row cells, expected %d, got %d", i+1, len(row.Children)))
			}
			cell := row.Children[i]
			cell.Calculated.OuterWidth = columnSizes[i]
			l.recalculateFromOuterWidth(cell)
			l.reflowTexts(cell)

			if cell.Align != "" {
				for _, item := range cell.Children {
					item.Align = cell.Align
				}
			}
		}
	}
}

// adjustRowsHeight adjusts table row heights for uniform appearance
func (l *Layouter) adjustRowsHeight(table *Widget) {
	height := float64(0)

	for _, row := range table.Children {
		maxCellHeight := l.getItemsInnerHeight(row)

		for _, cell := range row.Children {
			cell.Calculated.InnerHeight = maxCellHeight
			l.recalculateFromInnerHeight(cell)
		}

		rowHeight := float64(0)
		for _, cell := range row.Children {
			if cell.Calculated.OuterHeight > rowHeight {
				rowHeight = cell.Calculated.OuterHeight
			}
		}
		row.Calculated.InnerHeight = rowHeight
		l.recalculateFromInnerHeight(row)

		height += rowHeight
	}

	table.Calculated.InnerHeight = height
	l.recalculateFromInnerHeight(table)
}

// getItemsInnerHeight gets the maximum inner height of child widgets
func (l *Layouter) getItemsInnerHeight(w *Widget) float64 {
	height := float64(0)

	for _, child := range w.Children {
		v := child.Calculated.InnerHeight
		if v > height {
			height = v
		}
	}

	if w.Calculated.Direction == "column" {
		height = float64(0)
		for _, child := range w.Children {
			height += child.Calculated.InnerHeight
		}
		if w.Gap > 0 {
			height += w.Gap * float64(len(w.Children)-1)
		}
	} else {
		// TypeScript: height = w.children.max(t => t.calculated.innerHeight)
		maxHeight := float64(0)
		for _, child := range w.Children {
			if child.Calculated.InnerHeight > maxHeight {
				maxHeight = child.Calculated.InnerHeight
			}
		}
		height = maxHeight
	}

	return height
}

// getOuterWidth calculates the total outer width of a widget
func (l *Layouter) getOuterWidth(w *Widget) float64 {
	if len(w.Children) == 0 {
		return w.Calculated.OuterWidth
	}

	if w.Calculated.Direction == "row" {
		width := float64(0)
		for _, child := range w.Children {
			width += l.getOuterWidth(child)
		}
		if w.Gap > 0 {
			width += float64(len(w.Children)-1) * w.Gap
		}
		return width
	} else {
		maxWidth := float64(0)
		for _, child := range w.Children {
			w := l.getOuterWidth(child)
			if w > maxWidth {
				maxWidth = w
			}
		}
		return maxWidth
	}
}

// addjustCalculatedSize adjusts calculated width and height
func (l *Layouter) addjustCalculatedSize(w *Widget) {
	l.addjustCalculatedWidth(w)
	l.addjustCalculatedHeight(w)
}

// addjustCalculatedWidth adjusts calculated width values
func (l *Layouter) addjustCalculatedWidth(w *Widget) {
	if w.Calculated.Width == 0 {
		w.Calculated.Width = w.Width
	}

	w.Calculated.InnerWidth = w.Calculated.Width
	if w.Padding != nil {
		w.Calculated.InnerWidth -= w.Padding.Left + w.Padding.Right
		if w.Calculated.InnerWidth < 0 {
			w.Calculated.InnerWidth = 0
		}
	}

	w.Calculated.OuterWidth = w.Calculated.Width
	if w.Margin != nil {
		w.Calculated.OuterWidth += w.Margin.Left + w.Margin.Right
	}
}

// recalculateFromInnerHeight recalculates outer dimensions from inner height
func (l *Layouter) recalculateFromInnerHeight(w *Widget) {
	w.Calculated.Height = w.Calculated.InnerHeight
	if w.Padding != nil {
		w.Calculated.Height += w.Padding.Top + w.Padding.Bottom
	}

	w.Calculated.OuterHeight = w.Calculated.Height
	if w.Margin != nil {
		w.Calculated.OuterHeight += w.Margin.Top + w.Margin.Bottom
	}
}

// addjustCalculatedHeight adjusts calculated height values
func (l *Layouter) addjustCalculatedHeight(w *Widget) {
	if w.Calculated.Height == 0 {
		w.Calculated.Height = w.Height
	}

	w.Calculated.InnerHeight = w.Calculated.Height

	if w.Padding != nil {
		w.Calculated.InnerHeight -= w.Padding.Top + w.Padding.Bottom
		if w.Calculated.InnerHeight < 0 {
			w.Calculated.InnerHeight = 0
		}
	}

	w.Calculated.OuterHeight = w.Calculated.Height
	if w.Margin != nil {
		w.Calculated.OuterHeight += w.Margin.Top + w.Margin.Bottom
	}
}

// recalculateFromOuterHeight recalculates inner dimensions from outer height
func (l *Layouter) recalculateFromOuterHeight(w *Widget) {
	w.Calculated.Height = w.Calculated.OuterHeight
	if w.Margin != nil {
		w.Calculated.Height -= w.Margin.Top + w.Margin.Bottom
	}

	w.Calculated.InnerHeight = w.Calculated.Height
	if w.Padding != nil {
		w.Calculated.InnerHeight -= w.Padding.Top + w.Padding.Bottom
		if w.Calculated.InnerHeight < 0 {
			w.Calculated.InnerHeight = 0
		}
	}
}

// recalculateFromOuterWidth recalculates inner dimensions from outer width
func (l *Layouter) recalculateFromOuterWidth(w *Widget) {
	w.Calculated.Width = w.Calculated.OuterWidth
	if w.Margin != nil {
		w.Calculated.Width -= w.Margin.Left + w.Margin.Right
	}

	w.Calculated.InnerWidth = w.Calculated.Width
	if w.Padding != nil {
		w.Calculated.InnerWidth -= w.Padding.Left + w.Padding.Right
		if w.Calculated.InnerWidth < 0 {
			w.Calculated.InnerWidth = 0
		}
	}
}

// initCalculatedInfo initializes calculated information for a widget
func (l *Layouter) initCalculatedInfo(w *Widget, parent *Widget) {
	w.Calculated = &CalculatedInfo{}

	if parent != nil {
		// Exactly like TypeScript: w.calculated.fontFamily = w.fontFamily || parent.calculated.fontFamily
		if w.FontFamily != "" {
			w.Calculated.FontFamily = w.FontFamily
		} else {
			w.Calculated.FontFamily = parent.Calculated.FontFamily
		}

		// Exactly like TypeScript: w.calculated.fontSize = w.fontSize || parent.calculated.fontSize
		if w.FontSize != 0 {
			w.Calculated.FontSize = w.FontSize
		} else {
			w.Calculated.FontSize = parent.Calculated.FontSize
		}

		// Exactly like TypeScript: w.calculated.lineHeight = w.lineHeight || parent.calculated.lineHeight
		if w.LineHeight != 0 {
			w.Calculated.LineHeight = w.LineHeight
		} else {
			w.Calculated.LineHeight = parent.Calculated.LineHeight
		}

		// Exactly like TypeScript: w.calculated.color = w.color || parent.calculated.color
		if w.Color != nil {
			w.Calculated.Color = w.Color
		} else {
			w.Calculated.Color = parent.Calculated.Color
		}

		// Exactly like TypeScript: w.calculated.bold = w.bold || parent.calculated.bold
		w.Calculated.Bold = w.Bold || parent.Calculated.Bold
	} else {
		w.Calculated.FontFamily = w.FontFamily
		w.Calculated.FontSize = w.FontSize
		w.Calculated.LineHeight = w.LineHeight
		w.Calculated.Color = w.Color
		w.Calculated.Bold = w.Bold
	}

	if w.Width != 0 {
		w.Calculated.Width = w.Width
		l.addjustCalculatedWidth(w)
	}

	if w.Height != 0 {
		w.Calculated.Height = w.Height
		l.addjustCalculatedHeight(w)
	}

	w.Calculated.Direction = w.Direction
	if w.Calculated.Direction == "" {
		w.Calculated.Direction = "column"
	}

	if w.Calculated.Bold && !strings.HasSuffix(w.Calculated.FontFamily, "Bold") {
		w.Calculated.FontFamily += "Bold"
	}

	for _, child := range w.Children {
		l.initCalculatedInfo(child, w)
	}
}

// initImageSizeWidget handles image size calculation for widgets
func (l *Layouter) initImageSizeWidget(w *Widget) {
	if w.ImgWidth == 0 && w.Width != 0 {
		w.ImgWidth = w.Width
	}

	if w.ImgHeight == 0 && w.Calculated != nil && w.Calculated.Height != 0 {
		w.ImgHeight = w.Calculated.Height
	}

	if w.ImgWidth == 0 && w.ImgHeight == 0 {
		if len(w.Bytes) > 0 {
			img, _, err := image.Decode(bytes.NewReader(w.Bytes))
			if err == nil {
				bounds := img.Bounds()
				w.ImgWidth = float64(bounds.Dx())
				w.ImgHeight = float64(bounds.Dy())
			}
		}
	} else if w.ImgWidth != 0 && w.ImgHeight == 0 {
		if len(w.Bytes) > 0 {
			img, _, err := image.Decode(bytes.NewReader(w.Bytes))
			if err == nil {
				bounds := img.Bounds()
				originalWidth := float64(bounds.Dx())
				originalHeight := float64(bounds.Dy())
				if originalWidth > 0 {
					w.ImgHeight = (w.ImgWidth / originalWidth) * originalHeight
				}
			}
		}
	} else if w.ImgHeight != 0 && w.ImgWidth == 0 {
		if len(w.Bytes) > 0 {
			img, _, err := image.Decode(bytes.NewReader(w.Bytes))
			if err == nil {
				bounds := img.Bounds()
				originalWidth := float64(bounds.Dx())
				originalHeight := float64(bounds.Dy())
				if originalHeight > 0 {
					w.ImgWidth = (w.ImgHeight / originalHeight) * originalWidth
				}
			}
		}
	}

	if w.ImgWidth == 0 {
		w.ImgWidth = 100
	}

	if w.ImgHeight == 0 {
		w.ImgHeight = 100
	}

	if w.ImgMaxHeight > 0 && w.ImgHeight > w.ImgMaxHeight {
		ratio := w.ImgMaxHeight / w.ImgHeight
		w.ImgWidth *= ratio
		w.ImgHeight = w.ImgMaxHeight
	}

	if w.ImgMaxWidth > 0 && w.ImgWidth > w.ImgMaxWidth {
		ratio := w.ImgMaxWidth / w.ImgWidth
		w.ImgHeight *= ratio
		w.ImgWidth = w.ImgMaxWidth
	}

	w.Width = w.ImgWidth
	w.Height = w.ImgHeight

	l.addjustCalculatedSize(w)
}

// measureTextWidth measures text width with specified font size
func (l *Layouter) measureTextWidth(fontSize float64, text string) float64 {
	current := l.pdLibDoc.FontSize
	if current != fontSize {
		l.pdLibDoc.FontSize = fontSize
		width := l.pdLibDoc.MeasureTextWidth(text)
		l.pdLibDoc.FontSize = current
		return width
	}

	return l.pdLibDoc.MeasureTextWidth(text)
}

// splitLines splits text into lines that fit within available width
func (l *Layouter) splitLines(text string, fontSize, availableWidth float64) []string {
	var lines []string

	spaceWidth := l.measureTextWidth(fontSize, " ")

	textLines := strings.Split(text, "\n")
	for _, textLine := range textLines {
		textLine = strings.TrimSpace(textLine)
		if textLine == "" {
			continue
		}

		words := strings.Fields(textLine)
		var line []string
		lineWidth := float64(0)

		for _, word := range words {
			wordWidth := l.measureTextWidth(fontSize, word)

			if wordWidth > availableWidth {
				if len(line) > 0 {
					lines = append(lines, strings.Join(line, " "))
				}

				// Split long word into parts that fit
				remainingWord := word
				for len(remainingWord) > 0 {
					var wordBuff []string
					wordWidth := float64(0)

					runes := []rune(remainingWord)
					for _, r := range runes {
						s := string(r)
						charWidth := l.measureTextWidth(fontSize, s)
						if wordWidth+charWidth > availableWidth && len(wordBuff) > 0 {
							break
						}
						wordWidth += charWidth
						wordBuff = append(wordBuff, s)
					}

					if len(wordBuff) > 0 {
						lines = append(lines, strings.Join(wordBuff, ""))
						remainingWord = string(runes[len(wordBuff):])
					} else {
						// If not even one character fits, force at least one to avoid infinite loop
						lines = append(lines, string(runes[0]))
						remainingWord = string(runes[1:])
					}
				}

				line = []string{}
				lineWidth = 0
				continue
			}

			// Check if this word fits on the current line
			if lineWidth+wordWidth > availableWidth {
				if len(line) > 0 {
					lines = append(lines, strings.Join(line, " "))
				}

				line = []string{word}
				lineWidth = wordWidth
				continue
			}

			line = append(line, word)
			lineWidth += wordWidth
			lineWidth += spaceWidth
		}

		if len(line) > 0 {
			lines = append(lines, strings.Join(line, " "))
		}
	}

	return lines
}

// Helper functions

func (l *Layouter) deepCloneWidget(w *Widget) *Widget {
	if w == nil {
		return nil
	}

	// Start with a copy of the original widget
	clone := *w

	if w.Calculated != nil {
		newCalc := CalculatedInfo{}
		newCalc.X = w.Calculated.X
		newCalc.Y = w.Calculated.Y
		newCalc.InnerX = w.Calculated.InnerX
		newCalc.InnerY = w.Calculated.InnerY
		newCalc.OuterX = w.Calculated.OuterX
		newCalc.OuterY = w.Calculated.OuterY
		newCalc.Width = w.Calculated.Width
		newCalc.Height = w.Calculated.Height
		newCalc.InnerWidth = w.Calculated.InnerWidth
		newCalc.InnerHeight = w.Calculated.InnerHeight
		newCalc.OuterWidth = w.Calculated.OuterWidth
		newCalc.OuterHeight = w.Calculated.OuterHeight
		newCalc.FontFamily = w.Calculated.FontFamily
		newCalc.FontSize = w.Calculated.FontSize
		newCalc.LineHeight = w.Calculated.LineHeight
		newCalc.Color = w.Calculated.Color
		newCalc.Bold = w.Calculated.Bold
		newCalc.Direction = w.Calculated.Direction

		clone.Calculated = &newCalc
	}
	if w.Padding != nil {
		pad := *w.Padding
		clone.Padding = &pad
	}
	if w.Margin != nil {
		mar := *w.Margin
		clone.Margin = &mar
	}
	if w.Border != nil {
		bor := *w.Border
		if w.Border.Top != nil {
			top := *w.Border.Top
			bor.Top = &top
			if w.Border.Top.Color != nil {
				topColor := *w.Border.Top.Color
				bor.Top.Color = &topColor
			}
		}
		if w.Border.Bottom != nil {
			bottom := *w.Border.Bottom
			bor.Bottom = &bottom
			if w.Border.Bottom.Color != nil {
				bottomColor := *w.Border.Bottom.Color
				bor.Bottom.Color = &bottomColor
			}
		}
		if w.Border.Left != nil {
			left := *w.Border.Left
			bor.Left = &left
			if w.Border.Left.Color != nil {
				leftColor := *w.Border.Left.Color
				bor.Left.Color = &leftColor
			}
		}
		if w.Border.Right != nil {
			right := *w.Border.Right
			bor.Right = &right
			if w.Border.Right.Color != nil {
				rightColor := *w.Border.Right.Color
				bor.Right.Color = &rightColor
			}
		}
		clone.Border = &bor
	}
	if w.Color != nil {
		col := *w.Color
		clone.Color = &col
	}
	if w.BackgroundColor != nil {
		bg := *w.BackgroundColor
		clone.BackgroundColor = &bg
	}
	if w.StrokeColor != nil {
		sc := *w.StrokeColor
		clone.StrokeColor = &sc
	}
	if w.Option != nil {
		opt := *w.Option
		clone.Option = &opt
	}
	if w.CarryLast != nil {
		cl := *w.CarryLast
		clone.CarryLast = &cl
	}
	if w.CarryNext != nil {
		cn := *w.CarryNext
		clone.CarryNext = &cn
	}
	if w.CarryHeader != nil {
		clone.CarryHeader = l.deepCloneWidget(w.CarryHeader)
	}
	if w.CarryFooter != nil {
		clone.CarryFooter = l.deepCloneWidget(w.CarryFooter)
	}
	if w.AlternateColor != nil {
		ac := *w.AlternateColor
		clone.AlternateColor = &ac
	}

	// Clone slices
	if w.Children != nil {
		clone.Children = make([]*Widget, len(w.Children))
		for i, child := range w.Children {
			clone.Children[i] = l.deepCloneWidget(child)
		}
	}
	if w.ValueLines != nil {
		clone.ValueLines = make([]string, len(w.ValueLines))
		copy(clone.ValueLines, w.ValueLines)
	}

	return &clone
}

func (l *Layouter) deepCloneCalculated(c *CalculatedInfo) *CalculatedInfo {
	if c == nil {
		return nil
	}
	clone := *c
	return &clone
}
