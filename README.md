# PDF Library

A high-performance PDF generation library for Go that transforms XML documents into pixel-perfect PDF files. This library provides a declarative approach to PDF creation using familiar web-like semantics with widgets, layouts, and styling.

## Quick Start

### Advanced Example with Tables

```go
xml := `
<document fontFamily="roboto" fontSize="12">
    <page>
        <div fontSize="24" bold="true" align="center" marginBottom="20">
            Invoice #12345
        </div>
        
        <table cellPadding="5" cellBorder="1">
            <row isHeader="true" bold="true" backgroundColor="#f0f0f0">
                <cell width="200">Description</cell>
                <cell width="100" align="right">Quantity</cell>
                <cell width="100" align="right">Price</cell>
                <cell width="100" align="right">Total</cell>
            </row>
            <row>
                <cell>Premium Service</cell>
                <cell align="right">2</cell>
                <cell align="right">$50.00</cell>
                <cell align="right">$100.00</cell>
            </row>
            <row backgroundColor="#f9f9f9">
                <cell>Consultation</cell>
                <cell align="right">1</cell>
                <cell align="right">$75.00</cell>
                <cell align="right">$75.00</cell>
            </row>
        </table>
        
        <div marginTop="20" fontSize="16" bold="true" align="right">
            Total: $175.00
        </div>
    </page>
</document>`
```

## Core Widgets

### Document
Root container for the entire PDF document.

**Attributes:**
- `fontFamily` - Default font family ("roboto", "robotoBold")
- `fontSize` - Default font size in points
- `width` - Document width (default: A4)
- `height` - Document height (default: A4)
- `color` - Default text color (hex: "#ff0000" or rgb: "255,0,0")

### Page
Represents a single page in the document.

**Attributes:**
- `resetPageNumbers` - Reset page numbering from this page
- `header` - Page header content
- `footer` - Page footer content
- All styling attributes (colors, fonts, padding, etc.)

### Div
General-purpose container for content and layout.

**Attributes:**
- `value` - Text content
- `bold` - Bold text (true/false)
- `align` - Text alignment ("left", "center", "right")
- `direction` - Layout direction ("row", "column")
- `backgroundColor` - Background color
- `color` - Text color
- `fontSize` - Font size in points
- `fontFamily` - Font family
- `width`, `height` - Fixed dimensions
- `x`, `y` - Absolute positioning
- `gap` - Spacing between child elements
- `lineHeight` - Line height multiplier

### Table
Advanced table widget with automatic layout and pagination.

**Attributes:**
- `cellPadding` - Padding for all cells (object: `{top:5, right:5, bottom:5, left:5}`)
- `cellBorder` - Border for all cells
- `alternateColor` - Background color for alternate rows
- `breakMargin` - Minimum space before page break
- `carryColumn` - Column index for carry-over calculations (0-based)
- `carryHeader` - Header widget for carried values
- `carryFooter` - Footer widget for carry-over values

**Table Carry-over Example:**
```xml
<table carryColumn="2" cellPadding="3">
    <carryHeader>
        <div align="right">Carried forward: {carry}</div>
    </carryHeader>
    <carryFooter>
        <div align="right">To carry: {carry}</div>
    </carryFooter>
    
    <row isHeader="true">
        <cell>Item</cell>
        <cell>Quantity</cell>
        <cell align="right">Amount</cell>
    </row>
    <!-- Data rows -->
</table>
```

### Row & Cell
Table components for structured data.

**Row Attributes:**
- `isHeader` - Mark as header row (excluded from carry calculations)
- All styling attributes

**Cell Attributes:**
- `width` - Fixed cell width
- `align` - Cell content alignment
- All styling attributes

### Image
Display images and QR codes.

**Attributes:**
- `data` - Image data (base64 or URL)
- `imgWidth`, `imgHeight` - Image dimensions
- `imgMaxWidth`, `imgMaxHeight` - Maximum dimensions (maintains aspect ratio)

## Styling System

### Colors
Colors can be specified in multiple formats:
- Hex: `color="#ff0000"`
- RGB: `color="255,0,0"`
- Named: `color="red"` (limited set)

### Layout & Positioning
- **Box Model**: margin → border → padding → content
- **Flexbox-like**: `direction="row"` (horizontal) or `direction="column"` (vertical)
- **Alignment**: `align="left|center|right"` for text, `align="left|center|right"` for layout
- **Spacing**: `gap` for space between elements

### Typography
- **Fonts**: Built-in Roboto Regular and Bold
- **Sizes**: Specified in points (pt)
- **Line Height**: Automatic or custom multiplier

### Borders & Effects
```xml
<div border="1" borderColor="#000000" borderRadius="5">
    Content with rounded border
</div>
```

## Advanced Features

### Page Numbers
```xml
<page>
    <header>
        <div align="center">Page {page} of {pages}</div>
    </header>
    <footer>
        <div align="right">Document Footer</div>
    </footer>
</page>
```

### Template Interpolation
The library supports placeholder replacement:
- `{page}` - Current page number
- `{pages}` - Total page count
- `{carry}` - Carry-over values in tables
- Custom placeholders via template system

### Multi-page Tables
Tables automatically split across pages with:
- Header row repetition
- Carry-over value calculations
- Proper spacing and margins
- Page break optimization

## Architecture

The library follows a three-phase approach:

1. **Parse**: XML → Document AST
2. **Layout**: Calculate positions, sizes, and pagination  
3. **Render**: Generate PDF using gopdf library

```
XML Input → Parse → Layout → Render → PDF Output
           ↓       ↓       ↓
         AST    Calculated Final
               Positions   PDF
```

### Layout Engine
The layout engine implements:
- **Box model** calculations (margin, border, padding)
- **Flexbox-like** row/column layouts  
- **Text wrapping** and overflow handling
- **Table column** width distribution
- **Page breaking** with carry-over support
- **Absolute positioning** for headers/footers

## Examples

### Invoice Template
```xml
<document fontFamily="roboto" fontSize="10">
    <page>
        <!-- Header -->
        <div direction="row" marginBottom="20">
            <div width="300">
                <div fontSize="18" bold="true">INVOICE</div>
                <div>Invoice #: 12345</div>
                <div>Date: 2024-01-15</div>
            </div>
            <div width="200" align="right">
                <div bold="true">Company Name</div>
                <div>123 Business St</div>
                <div>City, State 12345</div>
            </div>
        </div>
        
        <!-- Items Table -->
        <table cellPadding="5" cellBorder="1" carryColumn="3">
            <carryHeader marginBottom="5">
                <div align="right" bold="true">Brought forward: {carry}</div>
            </carryHeader>
            
            <row isHeader="true" bold="true" backgroundColor="#e0e0e0">
                <cell width="250">Description</cell>
                <cell width="80" align="center">Qty</cell>
                <cell width="80" align="right">Price</cell>
                <cell width="90" align="right">Amount</cell>
            </row>
            
            <!-- Dynamic rows would be inserted here -->
            <row>
                <cell>Professional Services</cell>
                <cell align="center">10</cell>
                <cell align="right">$150.00</cell>
                <cell align="right">$1,500.00</cell>
            </row>
            
            <carryFooter marginTop="5">
                <div align="right" bold="true">To carry forward: {carry}</div>
            </carryFooter>
        </table>
        
        <!-- Totals -->
        <div marginTop="20" align="right">
            <div fontSize="14" bold="true">Total: $1,500.00</div>
        </div>
    </page>
</document>
```

### Multi-column Layout
```xml
<document>
    <page>
        <div direction="row" gap="20">
            <div width="250" backgroundColor="#f0f0f0" padding="10">
                <div fontSize="16" bold="true" marginBottom="10">Left Column</div>
                <div>This is the left column content with automatic text wrapping.</div>
            </div>
            
            <div width="250" backgroundColor="#e0f0ff" padding="10">
                <div fontSize="16" bold="true" marginBottom="10">Right Column</div>
                <div>This is the right column content with different background.</div>
            </div>
        </div>
    </page>
</document>
```
