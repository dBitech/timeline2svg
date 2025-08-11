/*
Package main implements a sophisticated SVG timeline generator that converts CSV data
into temporal visualizations with intelligent positioning algorithms.

This application features advanced temporal clustering analysis, constraint-based
positioning, and collision avoidance systems designed to balance time proportionality
with visual clarity.

Note: Its documentation comments were generated and maintained with
the assistance of AI (GitHub Copilot) to ensure comprehensive documentation
following Go conventions and best practices.
*/
package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Temporal clustering and positioning algorithm constants
const (
	// DefaultClusterThreshold defines the time window for automatic temporal clustering.
	// Events occurring within this duration are considered part of the same cluster
	// and receive specialized positioning treatment to preserve temporal relationships.
	DefaultClusterThreshold = 2 * time.Hour

	// UltraAggressiveBuffer is the buffer value used for temporal cluster events.
	// Negative values allow controlled text overlap to maintain tight clustering.
	UltraAggressiveBuffer = -50

	// TemporalClusterMinSeparation is the minimum pixel separation enforced
	// between events in a temporal cluster, ensuring basic readability.
	TemporalClusterMinSeparation = 20

	// StandardCollisionBuffer is the default buffer for non-cluster event collisions.
	StandardCollisionBuffer = 15

	// MixedClusterBuffer is used when one event is in a cluster and one is outside.
	MixedClusterBuffer = 5
)

// Global debug flag
var debugMode bool

// Global variable to store optimized callout lengths
var globalOptimizedCallouts []int

// debugPrint prints debug messages when debug mode is enabled
func debugPrint(format string, args ...interface{}) {
	if debugMode {
		fmt.Fprintf(os.Stderr, "[DEBUG] "+format+"\n", args...)
	}
}

// TimelineEvent represents a single event on the timeline with flexible data
type TimelineEvent struct {
	Timestamp time.Time
	Data      map[string]string // Flexible data storage for any columns
}

// GetDisplayText returns the text for a given display element
func (e TimelineEvent) GetDisplayText(elementName string) string {
	if elementName == "timestamp" {
		return e.Timestamp.Format("2006-01-02 15:04")
	}
	return e.Data[strings.ToLower(elementName)]
}

// ColumnStyle defines the styling for a specific column when using detailed column configuration
type ColumnStyle struct {
	Name       string `yaml:"name"`        // Column name from CSV header (case-insensitive matching)
	FontFamily string `yaml:"font_family"` // Font family for this column (e.g., "Arial, sans-serif", overrides global font.family)
	FontSize   int    `yaml:"font_size"`   // Font size in pixels for this column (overrides global font.size)
	FontWeight string `yaml:"font_weight"` // Font weight: "normal", "bold", "bolder", "lighter", or numeric values
	Color      string `yaml:"color"`       // Text color for this column (hex color code, overrides global colors)
	CSSClass   string `yaml:"css_class"`   // Custom CSS class name for advanced styling (optional)
}

// Config represents the complete configuration for SVG timeline generation.
// This structure maps directly to YAML configuration files and controls all aspects
// of timeline appearance and behavior, including:
//   - Font and color settings
//   - Layout dimensions and margins
//   - Timeline positioning and collision detection
//   - Event marker styling
//   - Column display and styling options
//
// Key configuration patterns:
//   - For time-proportional layouts: Set timeline.min_text_spacing to low values (10-20)
//   - For temporal clustering: Use timeline.callout_levels = 8 for more positioning options
//   - For detailed styling: Set columns.use_detailed_styling = true and define detailed_columns
type Config struct {
	Font struct {
		Family string `yaml:"family"` // Font family for all text elements (e.g., "Arial, sans-serif")
		Size   int    `yaml:"size"`   // Base font size in pixels for text elements
	} `yaml:"font"`
	Colors struct {
		Background string `yaml:"background"` // SVG background color (hex color code, e.g., "#ffffff")
		Timeline   string `yaml:"timeline"`   // Color of the main timeline line (hex color code)
		Events     string `yaml:"events"`     // Color of event markers (hex color code)
		Text       string `yaml:"text"`       // Color of title and main text (hex color code)
		Notes      string `yaml:"notes"`      // Color of notes text (hex color code)
	} `yaml:"colors"`
	Layout struct {
		Width        int `yaml:"width"`         // Total SVG width in pixels
		Height       int `yaml:"height"`        // Total SVG height in pixels
		MarginTop    int `yaml:"margin_top"`    // Top margin in pixels
		MarginBottom int `yaml:"margin_bottom"` // Bottom margin in pixels
		MarginLeft   int `yaml:"margin_left"`   // Left margin in pixels
		MarginRight  int `yaml:"margin_right"`  // Right margin in pixels
		EventRadius  int `yaml:"event_radius"`  // Radius of event markers in pixels (deprecated, use EventMarker.Size)
		EventSpacing int `yaml:"event_spacing"` // Vertical spacing from timeline to text in pixels
	} `yaml:"layout"`
	Timeline struct {
		LineWidth          int  `yaml:"line_width"`           // Width of the main timeline line in pixels
		ShowDates          bool `yaml:"show_dates"`           // Whether to display dates below/above event titles
		ShowTimes          bool `yaml:"show_times"`           // Whether to show times along with dates when available
		HorizontalBuffer   int  `yaml:"horizontal_buffer"`    // Horizontal buffer space before first and after last event in pixels
		AvoidTextOverlap   bool `yaml:"avoid_text_overlap"`   // Enable collision avoidance for overlapping text
		MinTextSpacing     int  `yaml:"min_text_spacing"`     // Minimum horizontal spacing in pixels to trigger overlap avoidance (lower values = more time-proportional)
		MinCalloutLength   int  `yaml:"min_callout_length"`   // Minimum length of vertical callout lines in pixels
		MaxCalloutLength   int  `yaml:"max_callout_length"`   // Maximum length of vertical callout lines in pixels
		CalloutLevels      int  `yaml:"callout_levels"`       // Number of different callout levels for vertical text stacking (higher = more positioning options)
		TextElementPadding int  `yaml:"text_element_padding"` // Vertical padding between text elements (title, timestamp, notes) in pixels
		CalloutTextGap     int  `yaml:"callout_text_gap"`     // Gap between callout line endpoint and text start in pixels
	} `yaml:"timeline"`
	Columns struct {
		DisplayOrder       []string      `yaml:"display_order"`        // Simple format: ordered list of column names to display (e.g., ["title", "timestamp", "notes"])
		DetailedColumns    []ColumnStyle `yaml:"detailed_columns"`     // Detailed format: full styling configuration per column (overrides simple format when UseDetailedStyling=true)
		TimestampColumn    string        `yaml:"timestamp_column"`     // Name of the CSV column containing timestamp data (required, case-insensitive)
		UseDetailedStyling bool          `yaml:"use_detailed_styling"` // Whether to use detailed column styling (true) or simple display order (false)
	} `yaml:"columns"`
	EventMarker struct {
		Shape       string `yaml:"shape"`        // Marker shape: "circle", "triangle", "square", or "diamond"
		Size        int    `yaml:"size"`         // Size of the marker in pixels (radius for circle, side length for others)
		FillColor   string `yaml:"fill_color"`   // Fill color of the marker (hex color code, e.g., "#4285f4")
		StrokeColor string `yaml:"stroke_color"` // Border/stroke color of the marker (hex color code)
		StrokeWidth int    `yaml:"stroke_width"` // Width of the marker border in pixels
	} `yaml:"event_marker"`
}

// getDefaultConfig returns the default configuration with sensible defaults for all parameters.
// These defaults provide a good starting point for most timeline visualizations:
//   - 1200x800px canvas with 100px margins
//   - 12px Arial font with standard colors
//   - 80px min_text_spacing (triggers collision avoidance easily)
//   - 4 callout levels for basic vertical separation
//   - Circle markers with blue fill
//
// For time-proportional layouts, consider lowering min_text_spacing to 10-20.
// For temporal clustering, consider increasing callout_levels to 6-8.
func getDefaultConfig() Config {
	return Config{
		Font: struct {
			Family string `yaml:"family"`
			Size   int    `yaml:"size"`
		}{
			Family: "Arial, sans-serif",
			Size:   12,
		},
		Colors: struct {
			Background string `yaml:"background"`
			Timeline   string `yaml:"timeline"`
			Events     string `yaml:"events"`
			Text       string `yaml:"text"`
			Notes      string `yaml:"notes"`
		}{
			Background: "#ffffff",
			Timeline:   "#333333",
			Events:     "#4285f4",
			Text:       "#333333",
			Notes:      "#666666",
		},
		Layout: struct {
			Width        int `yaml:"width"`
			Height       int `yaml:"height"`
			MarginTop    int `yaml:"margin_top"`
			MarginBottom int `yaml:"margin_bottom"`
			MarginLeft   int `yaml:"margin_left"`
			MarginRight  int `yaml:"margin_right"`
			EventRadius  int `yaml:"event_radius"`
			EventSpacing int `yaml:"event_spacing"`
		}{
			Width:        1200,
			Height:       800,
			MarginTop:    50,
			MarginBottom: 50,
			MarginLeft:   100,
			MarginRight:  100,
			EventRadius:  8,
			EventSpacing: 120,
		},
		Timeline: struct {
			LineWidth          int  `yaml:"line_width"`
			ShowDates          bool `yaml:"show_dates"`
			ShowTimes          bool `yaml:"show_times"`
			HorizontalBuffer   int  `yaml:"horizontal_buffer"`
			AvoidTextOverlap   bool `yaml:"avoid_text_overlap"`
			MinTextSpacing     int  `yaml:"min_text_spacing"`
			MinCalloutLength   int  `yaml:"min_callout_length"`
			MaxCalloutLength   int  `yaml:"max_callout_length"`
			CalloutLevels      int  `yaml:"callout_levels"`
			TextElementPadding int  `yaml:"text_element_padding"`
			CalloutTextGap     int  `yaml:"callout_text_gap"`
		}{
			LineWidth:          2,
			ShowDates:          true,
			ShowTimes:          true,
			HorizontalBuffer:   50,
			AvoidTextOverlap:   true,
			MinTextSpacing:     80,
			MinCalloutLength:   60,
			MaxCalloutLength:   180,
			CalloutLevels:      4,
			TextElementPadding: 2,
			CalloutTextGap:     5, // 5-pixel gap between callout lines and text
		},
		Columns: struct {
			DisplayOrder       []string      `yaml:"display_order"`
			DetailedColumns    []ColumnStyle `yaml:"detailed_columns"`
			TimestampColumn    string        `yaml:"timestamp_column"`
			UseDetailedStyling bool          `yaml:"use_detailed_styling"`
		}{
			DisplayOrder:       []string{"title", "timestamp", "notes"}, // Default order
			DetailedColumns:    []ColumnStyle{},                         // Empty by default
			TimestampColumn:    "timestamp",                             // Default timestamp column name
			UseDetailedStyling: false,                                   // Use simple format by default
		},
		EventMarker: struct {
			Shape       string `yaml:"shape"`
			Size        int    `yaml:"size"`
			FillColor   string `yaml:"fill_color"`
			StrokeColor string `yaml:"stroke_color"`
			StrokeWidth int    `yaml:"stroke_width"`
		}{
			Shape:       "circle",
			Size:        8,
			FillColor:   "#4285f4",
			StrokeColor: "#333333",
			StrokeWidth: 2,
		},
	}
}

// loadConfig loads configuration from a YAML file or returns default config if no file specified.
// The configuration system supports both simple and detailed column styling modes:
//   - Simple mode: Use columns.display_order to specify column order
//   - Detailed mode: Set columns.use_detailed_styling=true and define columns.detailed_columns
//
// Key configuration tips:
//   - Lower timeline.min_text_spacing (10-20) for more time-proportional positioning
//   - Higher timeline.callout_levels (6-8) provides more positioning options for clustering
//   - Set timeline.avoid_text_overlap=false to disable collision detection entirely
func loadConfig(configPath string) (Config, error) {
	if configPath == "" {
		return getDefaultConfig(), nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return Config{}, fmt.Errorf("error reading config file: %w", err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return Config{}, fmt.Errorf("error parsing config file: %w", err)
	}

	return config, nil
}

// parseCSV reads and parses the CSV file containing timeline events with configurable columns
func parseCSV(filename string, config Config) ([]TimelineEvent, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("error opening CSV file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	var events []TimelineEvent

	// Read header to get column mapping
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("error reading CSV header: %w", err)
	}

	// Create case-insensitive column mapping
	columnMap := make(map[string]int)
	for i, col := range header {
		columnMap[strings.ToLower(strings.TrimSpace(col))] = i
	}

	// Find the timestamp column
	timestampCol := -1
	timestampColumnName := strings.ToLower(config.Columns.TimestampColumn)
	if col, exists := columnMap[timestampColumnName]; exists {
		timestampCol = col
	} else {
		return nil, fmt.Errorf("timestamp column '%s' not found in CSV. Available columns: %v", config.Columns.TimestampColumn, header)
	}

	// Read data rows
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error reading CSV: %w", err)
		}

		event, err := parseCSVRowConfigurable(record, columnMap, timestampCol, config)
		if err != nil {
			return nil, fmt.Errorf("error parsing CSV row: %w", err)
		}

		events = append(events, event)
	}

	// Sort events by timestamp
	sort.Slice(events, func(i, j int) bool {
		return events[i].Timestamp.Before(events[j].Timestamp)
	})

	return events, nil
}

// parseCSVRowConfigurable parses a single CSV row into a TimelineEvent with configurable columns
func parseCSVRowConfigurable(record []string, columnMap map[string]int, timestampCol int, config Config) (TimelineEvent, error) {
	if timestampCol < 0 || timestampCol >= len(record) {
		return TimelineEvent{}, fmt.Errorf("timestamp column index %d out of range", timestampCol)
	}

	// Parse timestamp
	timestampFormats := []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
		"01/02/2006 15:04:05",
		"01/02/2006 15:04",
		"01/02/2006",
		"02/01/2006 15:04:05",
		"02/01/2006 15:04",
		"02/01/2006",
	}

	var timestamp time.Time
	var err error
	timestampStr := strings.TrimSpace(record[timestampCol])

	for _, format := range timestampFormats {
		timestamp, err = time.Parse(format, timestampStr)
		if err == nil {
			break
		}
	}

	if err != nil {
		return TimelineEvent{}, fmt.Errorf("unable to parse timestamp '%s': %w", timestampStr, err)
	}

	// Create data map for all columns
	data := make(map[string]string)
	for colName, colIndex := range columnMap {
		if colIndex < len(record) && colName != strings.ToLower(config.Columns.TimestampColumn) {
			data[colName] = strings.TrimSpace(record[colIndex])
		}
	}

	return TimelineEvent{
		Timestamp: timestamp,
		Data:      data,
	}, nil
}

// getColumnOrder returns the display order based on configuration format.
// Two modes are supported:
//   - Simple mode (default): Uses columns.display_order array
//   - Detailed mode: When columns.use_detailed_styling=true, extracts order from columns.detailed_columns
//
// The returned order determines the vertical stacking of text elements for each event.
func getColumnOrder(config Config) []string {
	if config.Columns.UseDetailedStyling && len(config.Columns.DetailedColumns) > 0 {
		order := make([]string, len(config.Columns.DetailedColumns))
		for i, col := range config.Columns.DetailedColumns {
			order[i] = col.Name
		}
		return order
	}
	return config.Columns.DisplayOrder
}

// getColumnStyle returns the styling information for a column with intelligent defaults.
// In detailed styling mode, returns the specific configuration from columns.detailed_columns.
// In simple mode or when detailed config is missing, provides sensible fallbacks:
//   - Uses global font.family and font.size as defaults
//   - Applies appropriate colors based on column type (timestamp vs. other columns)
//   - Generates CSS class names automatically
//
// Column names are matched case-insensitively for maximum compatibility.
func getColumnStyle(columnName string, config Config) ColumnStyle {
	columnName = strings.ToLower(columnName)

	if config.Columns.UseDetailedStyling {
		for _, col := range config.Columns.DetailedColumns {
			if strings.ToLower(col.Name) == columnName {
				// Fill in defaults if not specified
				style := col
				if style.FontFamily == "" {
					style.FontFamily = config.Font.Family
				}
				if style.FontSize == 0 {
					style.FontSize = config.Font.Size
				}
				if style.FontWeight == "" {
					style.FontWeight = "normal"
				}
				if style.Color == "" {
					// Use default colors based on column type
					switch columnName {
					case "timestamp":
						style.Color = config.Colors.Text
					default:
						style.Color = config.Colors.Text
					}
				}
				if style.CSSClass == "" {
					style.CSSClass = getElementClassName(columnName)
				}
				return style
			}
		}
	}

	// Fallback to default styling
	return ColumnStyle{
		Name:       columnName,
		FontFamily: config.Font.Family,
		FontSize:   config.Font.Size,
		FontWeight: "normal",
		Color:      config.Colors.Text,
		CSSClass:   getElementClassName(columnName),
	}
}

// getElementText returns the text for a display element
func getElementText(event TimelineEvent, elementName string, config Config) string {
	switch strings.ToLower(elementName) {
	case "timestamp":
		if config.Timeline.ShowTimes && (event.Timestamp.Hour() != 0 || event.Timestamp.Minute() != 0 || event.Timestamp.Second() != 0) {
			return event.Timestamp.Format("2006-01-02 15:04")
		}
		return event.Timestamp.Format("2006-01-02")
	default:
		return event.Data[strings.ToLower(elementName)]
	}
}

// getElementClassName returns the CSS class for a display element
func getElementClassName(elementName string) string {
	switch strings.ToLower(elementName) {
	case "timestamp":
		return "date-text"
	case "title":
		return "title-text"
	default:
		return "notes-text"
	}
}

// calculateConfigurableTextPositions calculates positions for all display elements
func calculateConfigurableTextPositions(event TimelineEvent, eventY int, above bool, config Config) map[string]int {
	positions := make(map[string]int)
	columnOrder := getColumnOrder(config)
	padding := config.Timeline.TextElementPadding

	currentY := eventY

	for i, elementName := range columnOrder {
		text := getElementText(event, elementName, config)
		if text != "" {
			style := getColumnStyle(elementName, config)
			bounds := estimateTextBounds(text, style.FontSize)

			if i == 0 {
				// First element positioning
				positions[elementName] = currentY
			} else {
				// Subsequent elements are offset by text height + padding
				if above {
					currentY += bounds.Height + padding
				} else {
					currentY -= bounds.Height + padding
				}
				positions[elementName] = currentY
			}
		}
	}

	return positions
} // generateSVG creates an SVG timeline from the events and config
func generateSVG(events []TimelineEvent, config Config) string {
	if len(events) == 0 {
		return ""
	}

	// Calculate timeline dimensions
	timelineWidth := config.Layout.Width - config.Layout.MarginLeft - config.Layout.MarginRight
	timelineHeight := config.Layout.Height - config.Layout.MarginTop - config.Layout.MarginBottom

	// Calculate usable timeline width after accounting for horizontal buffers
	usableTimelineWidth := timelineWidth - (2 * config.Timeline.HorizontalBuffer)
	timelineStartX := config.Layout.MarginLeft + config.Timeline.HorizontalBuffer

	// Start building SVG
	var svg strings.Builder
	svg.WriteString(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<svg width="%d" height="%d" xmlns="http://www.w3.org/2000/svg">
<rect width="100%%" height="100%%" fill="%s"/>
<defs>
<style>
.title-text { font-family: %s; font-size: %dpx; font-weight: bold; fill: %s; }
.notes-text { font-family: %s; font-size: %dpx; fill: %s; }
.date-text { font-family: %s; font-size: %dpx; fill: %s; }
</style>
</defs>
`, config.Layout.Width, config.Layout.Height, config.Colors.Background,
		config.Font.Family, config.Font.Size+2, config.Colors.Text,
		config.Font.Family, config.Font.Size-2, config.Colors.Notes,
		config.Font.Family, config.Font.Size-1, config.Colors.Text))

	// Draw main timeline line
	timelineY := config.Layout.MarginTop + timelineHeight/2
	svg.WriteString(fmt.Sprintf(`<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="%s" stroke-width="%d"/>`,
		config.Layout.MarginLeft, timelineY,
		config.Layout.MarginLeft+timelineWidth, timelineY,
		config.Colors.Timeline, config.Timeline.LineWidth))

	// Calculate positions for events based on actual timestamps
	if len(events) == 1 {
		// Single event goes in the middle of the usable timeline area
		x := timelineStartX + usableTimelineWidth/2
		drawEvent(&svg, events[0], x, timelineY, config, 0, []int{x})
	} else {
		// First calculate ideal callout lengths based on time-proportional positions
		// This preserves the sophisticated vertical level distribution logic
		timeProportionalPositions := make([]int, len(events))
		for i, event := range events {
			timeRange := events[len(events)-1].Timestamp.Sub(events[0].Timestamp)
			timeFromStart := event.Timestamp.Sub(events[0].Timestamp)
			proportion := float64(timeFromStart) / float64(timeRange)
			timeProportionalPositions[i] = timelineStartX + int(proportion*float64(usableTimelineWidth))
		}

		// Position events with constraint-based approach that includes callout optimization
		eventPositions := calculateSmartPositions(events, timelineStartX, usableTimelineWidth, config.Timeline.MinTextSpacing, config)

		// Use the globally optimized callout lengths from the smart positioning algorithm
		var calloutLengths []int
		if len(globalOptimizedCallouts) == len(events) {
			calloutLengths = make([]int, len(events))
			copy(calloutLengths, globalOptimizedCallouts)
			debugPrint("Using optimized callout lengths: %v", calloutLengths)
		} else {
			// Fallback to original calculation if optimization didn't work
			calloutLengths = make([]int, len(events))
			for i := range events {
				above := i%2 == 0
				calloutLengths[i] = calculateCalloutLength(timeProportionalPositions[i], i, timeProportionalPositions, above, config, timelineY)
			}
			debugPrint("Fallback to calculated callout lengths: %v", calloutLengths)
		}

		// Draw events with collision-free positioning
		for i, event := range events {
			drawEventWithCallout(&svg, event, eventPositions[i], timelineY, config, i, eventPositions, calloutLengths[i])
		}
	}

	svg.WriteString("</svg>")
	return svg.String()
}

// estimateTextWidth estimates the width of text in pixels based on character count
func estimateTextWidth(text string, fontSize int) int {
	// Rough estimation: average character width is about 0.6 * font size
	avgCharWidth := float64(fontSize) * 0.6
	return int(float64(len(text)) * avgCharWidth)
}

// estimateEventTextWidth calculates the maximum width needed for an event's text
func estimateEventTextWidth(event TimelineEvent, config Config) int {
	// Estimate text width for the first display element (usually title)
	var titleText string
	columnOrder := getColumnOrder(config)
	if len(columnOrder) > 0 {
		titleText = getElementText(event, columnOrder[0], config)
	}
	titleWidth := estimateTextWidth(titleText, config.Font.Size)

	// Check date width if dates are shown
	dateWidth := 0
	if config.Timeline.ShowDates {
		dateText := event.Timestamp.Format("2006-01-02")
		if config.Timeline.ShowTimes && (event.Timestamp.Hour() != 0 || event.Timestamp.Minute() != 0 || event.Timestamp.Second() != 0) {
			dateText = event.Timestamp.Format("2006-01-02 15:04")
		}
		dateWidth = estimateTextWidth(dateText, config.Font.Size)
	}

	// Check width of other display elements
	otherElementsWidth := 0
	for _, elementName := range getColumnOrder(config) {
		if elementName != "timestamp" {
			text := getElementText(event, elementName, config)
			if text != "" {
				style := getColumnStyle(elementName, config)
				// Account for text wrapping - find longest line
				words := strings.Fields(text)
				maxWidth := 20 // Default wrap width
				lines := wrapText(words, maxWidth)
				for _, line := range lines {
					lineWidth := estimateTextWidth(line, style.FontSize)
					if lineWidth > otherElementsWidth {
						otherElementsWidth = lineWidth
					}
				}
			}
		}
	}

	// Return the maximum width plus some padding
	maxWidth := titleWidth
	if dateWidth > maxWidth {
		maxWidth = dateWidth
	}
	if otherElementsWidth > maxWidth {
		maxWidth = otherElementsWidth
	}

	return maxWidth + 20 // Add padding
}

// calculateSmartPositions calculates event positions using a constraint-based approach
func calculateSmartPositions(events []TimelineEvent, startX, width, minSpacing int, config Config) []int {
	debugPrint("=== Constraint-Based Smart Positioning ===")
	debugPrint("StartX: %d, Width: %d, MinSpacing: %d", startX, width, minSpacing)

	if len(events) <= 1 {
		return []int{startX + width/2}
	}

	firstTime := events[0].Timestamp
	lastTime := events[len(events)-1].Timestamp
	totalDuration := lastTime.Sub(firstTime)

	debugPrint("Time range: %s to %s (duration: %s)", firstTime.Format("2006-01-02 15:04"), lastTime.Format("2006-01-02 15:04"), totalDuration)

	if totalDuration == 0 {
		// All events have the same timestamp, distribute evenly
		debugPrint("All events have same timestamp, using even distribution")
		positions := make([]int, len(events))
		for i := range events {
			x := startX + (i * width / (len(events) - 1))
			positions[i] = x
		}
		return positions
	}

	// Step 1: Calculate ideal proportional positions
	debugPrint("Step 1: Calculating ideal time-proportional positions...")
	idealPositions := make([]int, len(events))
	for i, event := range events {
		eventDuration := event.Timestamp.Sub(firstTime)
		proportion := float64(eventDuration) / float64(totalDuration)
		x := startX + int(float64(width)*proportion)
		idealPositions[i] = x
		debugPrint("Event %d: %s -> proportion %.3f -> ideal x=%d", i, event.Timestamp.Format("15:04"), proportion, x)
	}

	// Step 2: Optimize callout heights to minimize temporal distortion
	debugPrint("Step 2: Optimizing callout heights for temporal positioning...")

	// Timeline boundaries for collision detection
	timelineY := config.Layout.MarginTop + (config.Layout.Height-config.Layout.MarginTop-config.Layout.MarginBottom)/2

	// Try different callout height combinations to find best temporal fit
	optimizedCallouts, optimizedPositions := optimizeCalloutHeightsForTempo(events, idealPositions, startX, width, timelineY, config)

	debugPrint("Optimized callout heights: %v", optimizedCallouts)
	debugPrint("Optimized positions for temporal accuracy: %v", optimizedPositions)

	// Step 3: Apply constraint-based refinement if needed
	debugPrint("Step 3: Final constraint-based refinement...")
	minSpacingConstraints := make([][]int, len(events))
	for i := range minSpacingConstraints {
		minSpacingConstraints[i] = make([]int, len(events))
	}

	// Identify temporal cluster for constraint relaxation
	clusterThreshold := DefaultClusterThreshold
	eventFirstTime := events[0].Timestamp
	clusterSize := 1

	for i := 1; i < len(events); i++ {
		timeDiff := events[i].Timestamp.Sub(eventFirstTime)
		if timeDiff <= clusterThreshold {
			clusterSize = i + 1
		} else {
			break
		}
	}
	debugPrint("Final refinement: Using temporal cluster of %d events for relaxed constraints", clusterSize)

	// Check for remaining collisions with optimized setup
	for i := 0; i < len(events); i++ {
		for j := i + 1; j < len(events); j++ {
			// Calculate bounding boxes for optimized positions and callouts
			bbox1 := calculateEventBoundingBox(events[i], optimizedPositions[i], timelineY, optimizedCallouts[i], i, config)
			bbox2 := calculateEventBoundingBox(events[j], optimizedPositions[j], timelineY, optimizedCallouts[j], j, config)

			if detectBoundingBoxOverlap(bbox1, bbox2) {
				// Use extremely aggressive constraints for temporal cluster events
				var buffer int
				if i < clusterSize && j < clusterSize {
					// Both events in temporal cluster - allow massive overlap for tight clustering
					buffer = UltraAggressiveBuffer // Very negative buffer allows significant text overlap
					debugPrint("Using ultra-aggressive temporal clustering constraint for events %d and %d: buffer=%d", i, j, buffer)
				} else if i < clusterSize || j < clusterSize {
					// One event in cluster, one outside - use moderate relaxation
					buffer = MixedClusterBuffer
				} else {
					// Both events outside cluster - use normal buffer
					buffer = StandardCollisionBuffer
				}

				requiredSeparation := (bbox1.Width+bbox2.Width)/2 + buffer

				// For temporal cluster events, ensure minimum separation is very small
				if i < clusterSize && j < clusterSize {
					requiredSeparation = max(requiredSeparation, TemporalClusterMinSeparation) // Minimum separation for cluster events
				}

				// Store constraint: j must be at least this far from i
				minSpacingConstraints[i][j] = requiredSeparation
				minSpacingConstraints[j][i] = requiredSeparation

				debugPrint("Remaining constraint: Events %d and %d need minimum %d pixels separation", i, j, requiredSeparation)
			} else {
				// No collision, allow events to maintain current spacing
				currentSeparation := absInt(optimizedPositions[j] - optimizedPositions[i])
				minSpacingConstraints[i][j] = min(currentSeparation, config.EventMarker.Size)
				minSpacingConstraints[j][i] = minSpacingConstraints[i][j]
			}
		}
	}

	// Apply final constraint solving if there are any remaining issues
	finalPositions := solveConstraintBasedPositioning(events, optimizedPositions, minSpacingConstraints, startX, width, config)

	debugPrint("Final constraint-satisfied positions: %v", finalPositions)
	debugPrint("=== End Constraint-Based Smart Positioning ===")

	// Store optimized callouts globally so they can be used in later processing
	globalOptimizedCallouts = optimizedCallouts

	return finalPositions
}

// optimizeCalloutHeightsForTempo uses backward optimization from constraint solver results
func optimizeCalloutHeightsForTempo(events []TimelineEvent, idealPositions []int, startX, width, timelineY int, config Config) ([]int, []int) {
	debugPrint("--- Backward-Working Callout Height Optimization ---")

	n := len(events)

	// Step 1: Analyze temporal clustering to determine optimization scope
	debugPrint("Step 1: Analyzing temporal clustering...")

	// Find the actual temporal cluster - events within a reasonable time window
	clusterThreshold := DefaultClusterThreshold // Time window for tight clustering
	firstTime := events[0].Timestamp
	clusterSize := 1

	for i := 1; i < n; i++ {
		timeDiff := events[i].Timestamp.Sub(firstTime)
		if timeDiff <= clusterThreshold {
			clusterSize = i + 1
		} else {
			break // Found the end of the tight cluster
		}
	}

	debugPrint("Detected temporal cluster: first %d events within %v", clusterSize, clusterThreshold)
	if clusterSize > 1 {
		clusterDuration := events[clusterSize-1].Timestamp.Sub(events[0].Timestamp)
		debugPrint("Cluster spans: %s to %s (duration: %v)",
			events[0].Timestamp.Format("15:04"),
			events[clusterSize-1].Timestamp.Format("15:04"),
			clusterDuration)
	}

	// Step 2: Get baseline constraint-imposed positions with uniform callouts
	debugPrint("Step 2: Getting constraint-imposed baseline positions...")
	uniformCallouts := make([]int, n)
	minCallout := config.Timeline.MinCalloutLength
	for i := range uniformCallouts {
		uniformCallouts[i] = minCallout
	}

	// Get what the constraint solver would do with uniform callouts
	baselinePositions := simulateConstraintSolverResults(events, idealPositions, uniformCallouts, startX, width, timelineY, config)
	debugPrint("Baseline constraint-imposed positions: %v", baselinePositions)

	// Calculate initial temporal distortion
	baselineError := calculateTemporalDistortion(events, baselinePositions, idealPositions)
	debugPrint("Baseline temporal distortion: %.1f", baselineError)

	// Step 3: Test callout adjustments to allow movement back toward temporal positions
	debugPrint("Step 3: Testing callout adjustments to reduce temporal distortion...")

	bestCallouts := make([]int, n)
	bestPositions := make([]int, n)
	copy(bestCallouts, uniformCallouts)
	copy(bestPositions, baselinePositions)
	bestDistortion := baselineError

	// Generate callout height options with wider range for better vertical separation
	minCallout = config.Timeline.MinCalloutLength
	maxCallout := config.Timeline.MaxCalloutLength
	if maxCallout > minCallout+100 {
		maxCallout = minCallout + 100 // Reasonable limit
	}

	debugPrint("Using actual temporal cluster size: %d events", clusterSize)

	// Test systematic callout variations that create vertical separation for the ENTIRE cluster
	calloutOptions := []int{minCallout, minCallout + 25, minCallout + 50, minCallout + 75}
	if maxCallout > minCallout+75 {
		calloutOptions = append(calloutOptions, maxCallout)
	}

	debugPrint("Available callout heights: %v", calloutOptions)

	// Test combinations that create significant vertical separation
	testCombinations := generateVerticalSeparationCombinations(calloutOptions, clusterSize)

	for i, combo := range testCombinations {
		debugPrint("Testing combination %d: %v", i+1, combo)

		// Create test callout configuration
		testCallouts := make([]int, n)
		copy(testCallouts, uniformCallouts)

		// Apply combination to clustered events
		for j := 0; j < len(combo) && j < clusterSize; j++ {
			testCallouts[j] = combo[j]
		}

		// Simulate what positions would result from this callout configuration
		testPositions := simulateConstraintSolverResults(events, idealPositions, testCallouts, startX, width, timelineY, config)

		// Calculate temporal distortion
		distortion := calculateTemporalDistortion(events, testPositions, idealPositions)
		debugPrint("  Resulting positions: %v", testPositions)
		debugPrint("  Temporal distortion: %.1f (baseline: %.1f)", distortion, baselineError)

		// Check if this is an improvement
		if distortion < bestDistortion {
			bestDistortion = distortion
			copy(bestCallouts, testCallouts)
			copy(bestPositions, testPositions)
			debugPrint("  NEW BEST! Distortion reduced by %.1f", baselineError-distortion)
		}
	}

	debugPrint("Final optimized callouts: %v", bestCallouts)
	debugPrint("Final optimized positions: %v", bestPositions)
	debugPrint("Temporal distortion improvement: %.1f -> %.1f (%.1f%% better)",
		baselineError, bestDistortion, (baselineError-bestDistortion)/baselineError*100)

	return bestCallouts, bestPositions
}

// calculateBestPositionsForCallouts finds the best horizontal positions given fixed callout heights
func calculateBestPositionsForCallouts(events []TimelineEvent, callouts, idealPositions []int, timelineY int, config Config) []int {
	positions := make([]int, len(events))
	copy(positions, idealPositions)

	// Use a greedy approach: try to move each event as close as possible to its ideal position
	// while avoiding collisions, starting with the events that are furthest from ideal
	maxIterations := 20

	for iteration := 0; iteration < maxIterations; iteration++ {
		improved := false

		// Calculate how far each event is from its ideal position
		errors := make([]struct {
			index int
			error float64
		}, len(events))
		for i := range events {
			errors[i] = struct {
				index int
				error float64
			}{i, float64(absInt(positions[i] - idealPositions[i]))}
		}

		// Sort by error descending - work on worst cases first
		sort.Slice(errors, func(i, j int) bool {
			return errors[i].error > errors[j].error
		})

		// Try to improve position of each event
		for _, err := range errors {
			i := err.index
			if err.error < 5 { // Skip if already close enough
				continue
			}

			ideal := idealPositions[i]
			current := positions[i]

			// Try to move toward ideal in steps
			stepSize := 10
			targetPos := current

			if current < ideal {
				targetPos = min(ideal, current+stepSize)
			} else if current > ideal {
				targetPos = max(ideal, current-stepSize)
			}

			if targetPos == current {
				continue
			}

			// Test if this position would cause collisions
			testPositions := make([]int, len(positions))
			copy(testPositions, positions)
			testPositions[i] = targetPos

			if !hasCollisionsWithCallouts(events, testPositions, callouts, timelineY, config) {
				positions[i] = targetPos
				improved = true
			}
		}

		if !improved {
			break // No more improvements possible
		}
	}

	return positions
}

// hasCollisionsWithCallouts checks if given positions and callouts would create text collisions
func hasCollisionsWithCallouts(events []TimelineEvent, positions, callouts []int, timelineY int, config Config) bool {
	for i := 0; i < len(events); i++ {
		for j := i + 1; j < len(events); j++ {
			bbox1 := calculateEventBoundingBox(events[i], positions[i], timelineY, callouts[i], i, config)
			bbox2 := calculateEventBoundingBox(events[j], positions[j], timelineY, callouts[j], j, config)

			if detectBoundingBoxOverlap(bbox1, bbox2) {
				return true
			}
		}
	}
	return false
}

// calculateTemporalDistortion measures temporal distortion with dynamic clustering analysis
func calculateTemporalDistortion(events []TimelineEvent, actualPositions, idealPositions []int) float64 {
	if len(events) <= 1 {
		return 0.0
	}

	// Dynamic cluster detection - find events within the default threshold of first event
	clusterThreshold := DefaultClusterThreshold
	firstTime := events[0].Timestamp
	clusterSize := 1

	for i := 1; i < len(events); i++ {
		timeDiff := events[i].Timestamp.Sub(firstTime)
		if timeDiff <= clusterThreshold {
			clusterSize = i + 1
		} else {
			break
		}
	}

	totalDistortion := 0.0

	// Weight clustered events heavily, with decreasing weight by proximity to cluster
	for i := range events {
		distortion := float64(absInt(actualPositions[i] - idealPositions[i]))

		// Dynamic weighting based on actual cluster analysis
		weight := 1.0
		if i < clusterSize {
			// Events within the temporal cluster get high weights
			// Earlier events in cluster get slightly higher weights
			weight = 4.0 - (float64(i) * 0.3) // 4.0, 3.7, 3.4, 3.1, 2.8, etc.
		} else if i == clusterSize {
			// First event after cluster gets medium weight
			weight = 1.5
		}
		// Events far from cluster keep weight = 1.0

		totalDistortion += distortion * weight
	}

	return totalDistortion
}

// simulateConstraintSolverResults predicts what positions would result from constraint solving
func simulateConstraintSolverResults(events []TimelineEvent, idealPositions, callouts []int, startX, width, timelineY int, config Config) []int {
	// This simulates the constraint-based positioning process with temporal clustering awareness

	// Step 1: Identify temporal cluster
	clusterThreshold := DefaultClusterThreshold
	firstTime := events[0].Timestamp
	clusterSize := 1

	for i := 1; i < len(events); i++ {
		timeDiff := events[i].Timestamp.Sub(firstTime)
		if timeDiff <= clusterThreshold {
			clusterSize = i + 1
		} else {
			break
		}
	}

	// Step 2: Start with ideal positions
	positions := make([]int, len(events))
	copy(positions, idealPositions)

	// Step 3: Calculate constraint requirements based on callout configuration
	constraints := make([][]int, len(events))
	for i := range constraints {
		constraints[i] = make([]int, len(events))
	}

	// Calculate pairwise collision requirements with temporal clustering preference
	for i := 0; i < len(events); i++ {
		for j := i + 1; j < len(events); j++ {
			// Calculate bounding boxes for these callout heights
			bbox1 := calculateEventBoundingBox(events[i], idealPositions[i], timelineY, callouts[i], i, config)
			bbox2 := calculateEventBoundingBox(events[j], idealPositions[j], timelineY, callouts[j], j, config)

			if detectBoundingBoxOverlap(bbox1, bbox2) {
				// Both events in temporal cluster - use more relaxed constraints
				if i < clusterSize && j < clusterSize {
					// For temporal cluster events, allow more overlap - prioritize clustering
					requiredSeparation := (bbox1.Width+bbox2.Width)/3 + MixedClusterBuffer // Reduced separation
					constraints[i][j] = requiredSeparation
					constraints[j][i] = requiredSeparation
				} else {
					// Normal collision constraints for non-cluster events
					requiredSeparation := (bbox1.Width+bbox2.Width)/2 + StandardCollisionBuffer // Buffer
					constraints[i][j] = requiredSeparation
					constraints[j][i] = requiredSeparation
				}
			} else {
				// No collision, allow tight spacing
				constraints[i][j] = config.EventMarker.Size
				constraints[j][i] = constraints[i][j]
			}
		}
	}

	// Step 3: Apply simplified constraint solving (similar to solveConstraintBasedPositioning)
	maxIterations := 10
	for iteration := 0; iteration < maxIterations; iteration++ {
		violations := 0

		// Check all pairwise constraints
		for i := 0; i < len(events)-1; i++ {
			for j := i + 1; j < len(events); j++ {
				currentSeparation := positions[j] - positions[i]
				requiredSeparation := constraints[i][j]

				if currentSeparation < requiredSeparation {
					violations++
					deficit := requiredSeparation - currentSeparation

					// Distribute the adjustment
					halfDeficit := deficit / 2
					positions[i] -= halfDeficit
					positions[j] += halfDeficit
				}
			}
		}

		if violations == 0 {
			break
		}
	}

	// Step 4: Ensure chronological order and bounds
	for i := 0; i < len(events)-1; i++ {
		if positions[i] >= positions[i+1] {
			positions[i+1] = positions[i] + config.EventMarker.Size
		}
	}

	// Ensure bounds
	for i := range positions {
		if positions[i] < startX {
			positions[i] = startX
		}
		if positions[i] > startX+width {
			positions[i] = startX + width
		}
	}

	return positions
}

// generateVerticalSeparationCombinations creates callout combinations that maximize vertical separation
func generateVerticalSeparationCombinations(calloutOptions []int, clusterSize int) [][]int {
	combinations := [][]int{}

	// Start with baseline: all minimum
	baseline := make([]int, clusterSize)
	for i := range baseline {
		baseline[i] = calloutOptions[0]
	}
	combinations = append(combinations, baseline)

	if len(calloutOptions) >= 2 {
		min_val := calloutOptions[0]
		max_val := calloutOptions[len(calloutOptions)-1]

		// For 5-event clusters, create more sophisticated patterns
		if clusterSize == 5 {
			// Pattern 1: Maximum separation - extreme alternating
			pattern1 := []int{min_val, max_val, min_val, max_val, min_val}
			combinations = append(combinations, pattern1)

			// Pattern 2: Reverse extreme alternating
			pattern2 := []int{max_val, min_val, max_val, min_val, max_val}
			combinations = append(combinations, pattern2)

			// Pattern 3: Progressive staircase up
			if len(calloutOptions) >= 4 {
				pattern3 := []int{
					calloutOptions[0], // 40
					calloutOptions[1], // 65
					calloutOptions[2], // 90
					calloutOptions[3], // 115
					calloutOptions[4], // 140
				}
				combinations = append(combinations, pattern3)
			}

			// Pattern 4: Progressive staircase down
			if len(calloutOptions) >= 4 {
				pattern4 := []int{
					calloutOptions[4], // 140
					calloutOptions[3], // 115
					calloutOptions[2], // 90
					calloutOptions[1], // 65
					calloutOptions[0], // 40
				}
				combinations = append(combinations, pattern4)
			}

			// Pattern 5: V-shape - tall on ends, short in middle
			if len(calloutOptions) >= 3 {
				mid_val := calloutOptions[len(calloutOptions)/2]
				pattern5 := []int{max_val, mid_val, min_val, mid_val, max_val}
				combinations = append(combinations, pattern5)
			}

			// Pattern 6: Inverted V - short on ends, tall in middle
			if len(calloutOptions) >= 3 {
				mid_val := calloutOptions[len(calloutOptions)/2]
				pattern6 := []int{min_val, mid_val, max_val, mid_val, min_val}
				combinations = append(combinations, pattern6)
			}

			// Pattern 7: Maximum vertical spread for tight clustering
			// This should create the most vertical separation
			if len(calloutOptions) >= 5 {
				pattern7 := []int{
					min_val,           // Event 0: Morning Meeting (above, short)
					max_val,           // Event 1: Quick Check-in (below, tall)
					calloutOptions[1], // Event 2: Code Review (above, medium-short)
					calloutOptions[3], // Event 3: Architecture Discussion (below, medium-tall)
					calloutOptions[2], // Event 4: Sprint Planning (above, medium)
				}
				combinations = append(combinations, pattern7)
			}

		} else {
			// Fallback patterns for other cluster sizes

			// Pattern 1: Alternating min/max
			alt1 := make([]int, clusterSize)
			for i := range alt1 {
				if i%2 == 0 {
					alt1[i] = min_val
				} else {
					alt1[i] = max_val
				}
			}
			combinations = append(combinations, alt1)

			// Pattern 2: Alternating max/min
			alt2 := make([]int, clusterSize)
			for i := range alt2 {
				if i%2 == 0 {
					alt2[i] = max_val
				} else {
					alt2[i] = min_val
				}
			}
			combinations = append(combinations, alt2)

			// Pattern 3: Ascending
			ascending := make([]int, clusterSize)
			for i := range ascending {
				optionIndex := (i * len(calloutOptions)) / clusterSize
				if optionIndex >= len(calloutOptions) {
					optionIndex = len(calloutOptions) - 1
				}
				ascending[i] = calloutOptions[optionIndex]
			}
			combinations = append(combinations, ascending)

			// Pattern 4: Descending
			descending := make([]int, clusterSize)
			for i := range descending {
				optionIndex := ((clusterSize - 1 - i) * len(calloutOptions)) / clusterSize
				if optionIndex >= len(calloutOptions) {
					optionIndex = len(calloutOptions) - 1
				}
				descending[i] = calloutOptions[optionIndex]
			}
			combinations = append(combinations, descending)
		}
	}

	return combinations
}

// calculateTemporalError measures how far events are from their ideal time-proportional positions
func calculateTemporalError(events []TimelineEvent, actualPositions, idealPositions []int) float64 {
	totalError := 0.0

	for i := range events {
		error := float64(absInt(actualPositions[i] - idealPositions[i]))
		// Weight earlier events more heavily since they're more clustered
		weight := 1.0
		if i < 5 { // First 5 events are clustered
			weight = 2.0
		}
		totalError += error * weight
	}

	return totalError
}

// solveConstraintBasedPositioning redistributes events globally while satisfying spacing constraints
func solveConstraintBasedPositioning(events []TimelineEvent, idealPositions []int, constraints [][]int, startX, width int, config Config) []int {
	debugPrint("--- Constraint Solver ---")

	n := len(events)
	positions := make([]int, n)
	copy(positions, idealPositions)

	// Calculate the total constraint "pressure" - how much extra space we need
	totalConstraintSpace := 0

	// Find maximum constraint requirements
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			requiredSpace := constraints[i][j]
			idealSpace := absInt(idealPositions[j] - idealPositions[i])
			if requiredSpace > idealSpace {
				totalConstraintSpace += (requiredSpace - idealSpace)
			}
		}
	}

	debugPrint("Constraint pressure: need %d extra pixels beyond ideal spacing", totalConstraintSpace)

	if totalConstraintSpace <= 0 {
		// No constraints violated, use ideal positions
		debugPrint("No constraint violations, using ideal positions")
		return positions
	}

	// Strategy: Use iterative constraint relaxation with proportional scaling
	maxIterations := 20
	for iteration := 0; iteration < maxIterations; iteration++ {
		debugPrint("Constraint solver iteration %d", iteration+1)

		violations := 0

		// Check all pairwise constraints
		for i := 0; i < n-1; i++ {
			for j := i + 1; j < n; j++ {
				currentSeparation := positions[j] - positions[i]
				requiredSeparation := constraints[i][j]

				if currentSeparation < requiredSeparation {
					violations++
					deficit := requiredSeparation - currentSeparation

					// Distribute the adjustment proportionally based on ideal positions
					totalIdealRange := idealPositions[n-1] - idealPositions[0]
					if totalIdealRange > 0 {
						// Calculate adjustment weights based on time proportions
						leftWeight := float64(idealPositions[i]-idealPositions[0]) / float64(totalIdealRange)
						rightWeight := float64(idealPositions[n-1]-idealPositions[j]) / float64(totalIdealRange)

						leftAdjustment := int(float64(deficit) * leftWeight / (leftWeight + rightWeight + 0.1))
						rightAdjustment := deficit - leftAdjustment

						// Apply adjustments while preserving chronological order
						newPosI := positions[i] - leftAdjustment
						newPosJ := positions[j] + rightAdjustment

						// Ensure we don't violate bounds or chronological order
						if newPosI >= startX && newPosJ <= startX+width && newPosI < newPosJ {
							positions[i] = newPosI
							positions[j] = newPosJ
							debugPrint("  Adjusted events %d,%d: moved %d left by %d, %d right by %d",
								i, j, i, leftAdjustment, j, rightAdjustment)
						}
					}
				}
			}
		}

		if violations == 0 {
			debugPrint("All constraints satisfied after %d iterations", iteration+1)
			break
		}

		debugPrint("Iteration %d: %d constraint violations remaining", iteration+1, violations)
	}

	// Final pass: ensure chronological order and bounds
	for i := 0; i < n-1; i++ {
		if positions[i] >= positions[i+1] {
			// Force minimum separation while maintaining order
			positions[i+1] = positions[i] + config.EventMarker.Size
		}
	}

	// Ensure all positions are within bounds
	for i := range positions {
		if positions[i] < startX {
			positions[i] = startX
		}
		if positions[i] > startX+width {
			positions[i] = startX + width
		}
	}

	debugPrint("Final constraint-solved positions: %v", positions)
	return positions
}

// adjustForTextCollisions detects and resolves horizontal text collisions between events
func adjustForTextCollisions(events []TimelineEvent, positions []int, config Config) []int {
	debugPrint("=== Text Collision Detection ===")
	if len(events) <= 1 {
		return positions
	}

	// Calculate timeline boundaries (add some buffer from margins)
	minX := config.Layout.MarginLeft + 20                        // 20px buffer from left edge
	maxX := config.Layout.Width - config.Layout.MarginRight - 20 // 20px buffer from right edge
	debugPrint("Timeline boundaries: minX=%d, maxX=%d", minX, maxX)

	// Create text bounding boxes for each event
	type TextBounds struct {
		left, right int
		above       bool
	}

	bounds := make([]TextBounds, len(events))
	adjustedPositions := make([]int, len(positions))
	copy(adjustedPositions, positions)

	// Calculate initial text bounds for each event
	for i, event := range events {
		above := i%2 == 0
		textWidth := estimateEventTextWidth(event, config)
		halfWidth := textWidth / 2

		bounds[i] = TextBounds{
			left:  adjustedPositions[i] - halfWidth,
			right: adjustedPositions[i] + halfWidth,
			above: above,
		}

		debugPrint("Event %d: x=%d, textWidth=%d, bounds=[%d,%d], above=%v",
			i, adjustedPositions[i], textWidth, bounds[i].left, bounds[i].right, above)
	}

	// Detect and resolve collisions iteratively
	maxIterations := 10
	for iteration := 0; iteration < maxIterations; iteration++ {
		debugPrint("--- Collision Detection Iteration %d ---", iteration+1)
		hasCollisions := false

		for i := 0; i < len(events); i++ {
			for j := i + 1; j < len(events); j++ {
				// Only check collisions between events on the same side of timeline
				if bounds[i].above != bounds[j].above {
					continue
				}

				// Check for horizontal overlap
				if bounds[i].right > bounds[j].left && bounds[i].left < bounds[j].right {
					debugPrint("Collision detected between event %d [%d,%d] and event %d [%d,%d]",
						i, bounds[i].left, bounds[i].right, j, bounds[j].left, bounds[j].right)

					hasCollisions = true

					// Calculate overlap and required adjustment
					overlap := min(bounds[i].right, bounds[j].right) - max(bounds[i].left, bounds[j].left)
					adjustment := (overlap / 2) + 10 // Add 10px buffer between texts

					debugPrint("Overlap: %d pixels, adjustment: %d", overlap, adjustment)

					// Move events apart, but respect boundaries
					if adjustedPositions[i] < adjustedPositions[j] {
						// Move event i left and event j right
						newPosI := adjustedPositions[i] - adjustment
						newPosJ := adjustedPositions[j] + adjustment

						// Ensure positions stay within boundaries
						textWidthI := estimateEventTextWidth(events[i], config)
						textWidthJ := estimateEventTextWidth(events[j], config)

						if newPosI-textWidthI/2 < minX {
							newPosI = minX + textWidthI/2
						}
						if newPosJ+textWidthJ/2 > maxX {
							newPosJ = maxX - textWidthJ/2
						}

						adjustedPositions[i] = newPosI
						adjustedPositions[j] = newPosJ
						debugPrint("Moving event %d left to %d, event %d right to %d",
							i, adjustedPositions[i], j, adjustedPositions[j])
					} else {
						// Move event j left and event i right
						newPosJ := adjustedPositions[j] - adjustment
						newPosI := adjustedPositions[i] + adjustment

						// Ensure positions stay within boundaries
						textWidthI := estimateEventTextWidth(events[i], config)
						textWidthJ := estimateEventTextWidth(events[j], config)

						if newPosJ-textWidthJ/2 < minX {
							newPosJ = minX + textWidthJ/2
						}
						if newPosI+textWidthI/2 > maxX {
							newPosI = maxX - textWidthI/2
						}

						adjustedPositions[j] = newPosJ
						adjustedPositions[i] = newPosI
						debugPrint("Moving event %d left to %d, event %d right to %d",
							j, adjustedPositions[j], i, adjustedPositions[i])
					}

					// Update bounds after position changes
					for k := 0; k < len(events); k++ {
						textWidth := estimateEventTextWidth(events[k], config)
						halfWidth := textWidth / 2
						bounds[k].left = adjustedPositions[k] - halfWidth
						bounds[k].right = adjustedPositions[k] + halfWidth
					}
				}
			}
		}

		if !hasCollisions {
			debugPrint("No more collisions detected after %d iterations", iteration+1)
			break
		}

		if iteration == maxIterations-1 {
			debugPrint("Maximum iterations reached, some collisions may remain")
		}
	}

	debugPrint("Final adjusted positions: %v", adjustedPositions)
	debugPrint("=== End Text Collision Detection ===")
	return adjustedPositions
}

// TextBoundingBox represents the complete bounding box of an event's text
type TextBoundingBox struct {
	X, Y          int  // Center position
	Width, Height int  // Total dimensions
	Left, Right   int  // Calculated bounds
	Top, Bottom   int  // Calculated bounds
	EventIndex    int  // Which event this belongs to
	Above         bool // Whether this is above or below timeline
}

// calculateEventBoundingBox calculates the complete 2D bounding box for an event's text
func calculateEventBoundingBox(event TimelineEvent, x, y int, calloutLength int, index int, config Config) TextBoundingBox {
	above := index%2 == 0

	// Calculate vertical offset from timeline
	adjustedCalloutLength := calloutLength
	if !above {
		adjustedCalloutLength = -calloutLength
	}
	eventY := y + adjustedCalloutLength

	// For below-timeline events, adjust eventY to provide clearance above the first text element
	if !above {
		// Get the first text element to determine its height
		columnOrder := getColumnOrder(config)
		for _, elementName := range columnOrder {
			text := getElementText(event, elementName, config)
			if text != "" {
				style := getColumnStyle(elementName, config)
				bounds := estimateTextBounds(text, style.FontSize)
				// Move the callout endpoint up to provide clearance above the text
				// Use configurable gap between callout line end and text start
				eventY -= bounds.Height + config.Timeline.TextElementPadding + config.Timeline.CalloutTextGap
				break
			}
		}
	}

	// Calculate text positioning for this event
	positions := calculateConfigurableTextPositions(event, eventY, above, config)

	// Find the bounds of all text elements
	minY, maxY := eventY, eventY
	maxWidth := 0

	columnOrder := getColumnOrder(config)
	for _, elementName := range columnOrder {
		if position, exists := positions[elementName]; exists {
			text := getElementText(event, elementName, config)
			if text != "" {
				style := getColumnStyle(elementName, config)

				// Calculate realistic text width with wrapping for longer text
				var textWidth int
				if strings.ToLower(elementName) == "notes" && len(text) > 30 {
					// For notes, assume reasonable wrapping at about 25-30 characters per line
					maxLineLength := 30
					lines := len(text) / maxLineLength
					if len(text)%maxLineLength > 0 {
						lines++
					}
					// Use the shorter of wrapped width or a reasonable maximum
					wrappedWidth := estimateTextWidth(strings.Repeat("A", maxLineLength), style.FontSize)
					singleLineWidth := estimateTextWidth(text, style.FontSize)
					textWidth = min(wrappedWidth, singleLineWidth)
					debugPrint("Event %d, element '%s': text='%s', fontSize=%d, singleLine=%d, wrapped=%d, using=%d",
						index, elementName, text[:min(30, len(text))], style.FontSize, singleLineWidth, wrappedWidth, textWidth)
				} else {
					textWidth = estimateTextWidth(text, style.FontSize)
					debugPrint("Event %d, element '%s': text='%s', fontSize=%d, textWidth=%d",
						index, elementName, text, style.FontSize, textWidth)
				}

				if textWidth > maxWidth {
					maxWidth = textWidth
				}

				// Update vertical bounds
				if position < minY {
					minY = position
				}
				if position+style.FontSize > maxY {
					maxY = position + style.FontSize
				}
			}
		}
	}

	// Add some padding
	padding := 5
	width := maxWidth + (padding * 2)
	height := (maxY - minY) + (padding * 2)

	bbox := TextBoundingBox{
		X:          x,
		Y:          (minY + maxY) / 2, // Center Y
		Width:      width,
		Height:     height,
		Left:       x - width/2,
		Right:      x + width/2,
		Top:        minY - padding,
		Bottom:     maxY + padding,
		EventIndex: index,
		Above:      above,
	}

	debugPrint("Event %d bounding box: [%d,%d] to [%d,%d] (w=%d, h=%d)",
		index, bbox.Left, bbox.Top, bbox.Right, bbox.Bottom, bbox.Width, bbox.Height)

	return bbox
}

// detectBoundingBoxOverlap checks if two bounding boxes overlap in 2D space.
// It returns true if the boxes intersect in any way, false if they are completely separate.
// Uses the standard rectangle overlap detection algorithm: boxes don't overlap only if
// one box is completely to the left, right, above, or below the other box.
func detectBoundingBoxOverlap(box1, box2 TextBoundingBox) bool {
	// No overlap if one box is completely to the left, right, above, or below the other
	if box1.Right <= box2.Left || box1.Left >= box2.Right ||
		box1.Bottom <= box2.Top || box1.Top >= box2.Bottom {
		return false
	}
	return true
}

// resolve2DCollisions implements comprehensive 2D bounding box collision detection and resolution
func resolve2DCollisions(events []TimelineEvent, positions []int, calloutLengths []int, timelineY int, config Config) ([]int, []int) {
	debugPrint("=== 2D Collision Detection ===")

	if len(events) <= 1 {
		return positions, calloutLengths
	}

	// Timeline boundaries
	minX := config.Layout.MarginLeft + 20
	maxX := config.Layout.Width - config.Layout.MarginRight - 20
	debugPrint("Timeline boundaries: minX=%d, maxX=%d", minX, maxX)

	adjustedPositions := make([]int, len(positions))
	adjustedCallouts := make([]int, len(calloutLengths))
	copy(adjustedPositions, positions)
	copy(adjustedCallouts, calloutLengths)

	// Collision resolution strategy: prioritize horizontal separation when min_text_spacing is too small
	maxIterations := 10
	for iteration := 0; iteration < maxIterations; iteration++ {
		debugPrint("--- 2D Collision Iteration %d ---", iteration+1)

		// Calculate current bounding boxes
		boundingBoxes := make([]TextBoundingBox, len(events))
		for i, event := range events {
			boundingBoxes[i] = calculateEventBoundingBox(event, adjustedPositions[i], timelineY, adjustedCallouts[i], i, config)
		}

		hasCollisions := false

		// Check all pairs for collisions
		for i := 0; i < len(boundingBoxes); i++ {
			for j := i + 1; j < len(boundingBoxes); j++ {
				if detectBoundingBoxOverlap(boundingBoxes[i], boundingBoxes[j]) {
					debugPrint("2D Collision detected between event %d and event %d", i, j)
					hasCollisions = true

					// Calculate overlap dimensions
					overlapWidth := min(boundingBoxes[i].Right, boundingBoxes[j].Right) - max(boundingBoxes[i].Left, boundingBoxes[j].Left)
					overlapHeight := min(boundingBoxes[i].Bottom, boundingBoxes[j].Bottom) - max(boundingBoxes[i].Top, boundingBoxes[j].Top)

					debugPrint("Overlap: %dx%d pixels", overlapWidth, overlapHeight)

					// Calculate time gap between events to inform collision resolution strategy
					timeDiff := absTimeDuration(events[i].Timestamp.Sub(events[j].Timestamp))

					// Strategy: If events are very close horizontally (less than half the text width),
					// prioritize horizontal separation to preserve readability
					averageTextWidth := (boundingBoxes[i].Width + boundingBoxes[j].Width) / 2
					horizontalDistance := absInt(adjustedPositions[i] - adjustedPositions[j])

					// Also consider if they already have good vertical separation from dynamic callouts
					verticalDistance := absInt(adjustedCallouts[i] - adjustedCallouts[j])

					// For events with large time gaps (>1 hour), prefer vertical separation to preserve time proportionality
					if timeDiff > time.Hour && horizontalDistance > 30 {
						// These events should be temporally spaced - use vertical separation
						resolveVerticalCollisionGentle(i, j, &adjustedCallouts, overlapHeight, config)
						debugPrint("Resolved with vertical separation (preserving time gap of %v): callouts now [%d, %d]", timeDiff, adjustedCallouts[i], adjustedCallouts[j])
					} else if horizontalDistance < averageTextWidth/2 {
						// Events are too close horizontally - check if we can use existing vertical separation
						if verticalDistance > 30 && boundingBoxes[i].Above == boundingBoxes[j].Above {
							// Same side with good vertical separation - enhance it slightly
							resolveVerticalCollisionGentle(i, j, &adjustedCallouts, overlapHeight, config)
							debugPrint("Resolved with enhanced vertical separation: callouts now [%d, %d]", adjustedCallouts[i], adjustedCallouts[j])
						} else {
							// Use minimal horizontal separation to preserve time relationships
							resolveHorizontalCollisionMinimal(i, j, &adjustedPositions, overlapWidth, events, config, minX, maxX)
							debugPrint("Resolved with minimal horizontal separation (events too close): positions now [%d, %d]", adjustedPositions[i], adjustedPositions[j])
						}
					} else if boundingBoxes[i].Above != boundingBoxes[j].Above {
						// Different sides - use gentle horizontal separation
						resolveHorizontalCollisionMinimal(i, j, &adjustedPositions, overlapWidth, events, config, minX, maxX)
						debugPrint("Resolved with minimal horizontal separation (different sides): positions now [%d, %d]", adjustedPositions[i], adjustedPositions[j])
					} else {
						// Same side and reasonable horizontal distance - prefer vertical separation
						resolveVerticalCollisionGentle(i, j, &adjustedCallouts, overlapHeight, config)
						debugPrint("Resolved with gentle vertical separation: callouts now [%d, %d]", adjustedCallouts[i], adjustedCallouts[j])
					}
				}
			}
		}

		if !hasCollisions {
			debugPrint("No 2D collisions detected after %d iterations", iteration+1)
			break
		}

		if iteration == maxIterations-1 {
			debugPrint("Maximum iterations reached, some collisions may remain")
		}
	}

	debugPrint("Final adjusted positions: %v", adjustedPositions)
	debugPrint("Final adjusted callouts: %v", adjustedCallouts)

	// Enforce minimum marker separation for ALL events (critical constraint)
	debugPrint("=== Enforcing Marker Separation ===")
	baseMinSpacing := config.EventMarker.Size
	if baseMinSpacing < 6 {
		baseMinSpacing = 6
	}

	// Sort positions by value to ensure we check them in left-to-right order
	positionIndices := make([]int, len(adjustedPositions))
	for i := range positionIndices {
		positionIndices[i] = i
	}

	// Sort indices by their corresponding positions
	for i := 0; i < len(positionIndices)-1; i++ {
		for j := i + 1; j < len(positionIndices); j++ {
			if adjustedPositions[positionIndices[i]] > adjustedPositions[positionIndices[j]] {
				positionIndices[i], positionIndices[j] = positionIndices[j], positionIndices[i]
			}
		}
	}

	// Enforce minimum spacing between adjacent markers
	for i := 1; i < len(positionIndices); i++ {
		currentIdx := positionIndices[i]
		prevIdx := positionIndices[i-1]

		if adjustedPositions[currentIdx]-adjustedPositions[prevIdx] < baseMinSpacing {
			adjustment := baseMinSpacing - (adjustedPositions[currentIdx] - adjustedPositions[prevIdx])
			adjustedPositions[currentIdx] += adjustment
			debugPrint("Enforced marker separation: moved event %d from %d to %d",
				currentIdx, adjustedPositions[currentIdx]-adjustment, adjustedPositions[currentIdx])
		}
	}

	debugPrint("=== End Marker Separation Enforcement ===")

	// Ensure chronological order is preserved by adjusting positions if necessary
	debugPrint("=== Enforcing Chronological Order ===")
	for i := 0; i < len(events)-1; i++ {
		for j := i + 1; j < len(events); j++ {
			// Check if chronologically earlier event is positioned after a later event
			if events[i].Timestamp.Before(events[j].Timestamp) && adjustedPositions[i] > adjustedPositions[j] {
				debugPrint("Chronological order violation: Event %d (%s) at position %d should be before Event %d (%s) at position %d",
					i, events[i].Timestamp.Format("15:04"), adjustedPositions[i],
					j, events[j].Timestamp.Format("15:04"), adjustedPositions[j])

				// Swap positions to maintain chronological order
				adjustedPositions[i], adjustedPositions[j] = adjustedPositions[j], adjustedPositions[i]

				debugPrint("Corrected positions: Event %d now at %d, Event %d now at %d", i, adjustedPositions[i], j, adjustedPositions[j])
			}
		}
	}
	debugPrint("Final chronologically ordered positions: %v", adjustedPositions)
	debugPrint("=== End Chronological Order Enforcement ===")

	debugPrint("=== End 2D Collision Detection ===")

	return adjustedPositions, adjustedCallouts
}

// resolveVerticalCollision adjusts callout lengths to separate events vertically
func resolveVerticalCollision(i, j int, calloutLengths *[]int, overlapHeight int, config Config) {
	// Increase the difference in callout lengths
	adjustment := (overlapHeight / 2) + 10 // Add buffer

	if (*calloutLengths)[i] <= (*calloutLengths)[j] {
		// Decrease i's callout, increase j's callout
		newI := (*calloutLengths)[i] - adjustment
		newJ := (*calloutLengths)[j] + adjustment

		// Ensure we stay within bounds
		if newI < config.Timeline.MinCalloutLength {
			newI = config.Timeline.MinCalloutLength
		}
		if newJ > config.Timeline.MaxCalloutLength {
			newJ = config.Timeline.MaxCalloutLength
		}

		(*calloutLengths)[i] = newI
		(*calloutLengths)[j] = newJ
	} else {
		// Decrease j's callout, increase i's callout
		newI := (*calloutLengths)[i] + adjustment
		newJ := (*calloutLengths)[j] - adjustment

		// Ensure we stay within bounds
		if newJ < config.Timeline.MinCalloutLength {
			newJ = config.Timeline.MinCalloutLength
		}
		if newI > config.Timeline.MaxCalloutLength {
			newI = config.Timeline.MaxCalloutLength
		}

		(*calloutLengths)[i] = newI
		(*calloutLengths)[j] = newJ
	}
}

// resolveVerticalCollisionGentle makes smaller adjustments for better visual coherence
// This works with the existing dynamic callout heights rather than overriding them
func resolveVerticalCollisionGentle(i, j int, calloutLengths *[]int, overlapHeight int, config Config) {
	// Use smaller adjustment for better visual coherence
	adjustment := (overlapHeight / 3) + 15 // More conservative adjustment

	if (*calloutLengths)[i] <= (*calloutLengths)[j] {
		// Smaller adjustments to maintain visual grouping
		newI := (*calloutLengths)[i] - adjustment/2
		newJ := (*calloutLengths)[j] + adjustment/2

		// Ensure we stay within configured bounds (respect the dynamic range)
		if newI < config.Timeline.MinCalloutLength {
			newI = config.Timeline.MinCalloutLength
		}
		// Allow full range up to MaxCalloutLength instead of artificial cap
		if newJ > config.Timeline.MaxCalloutLength {
			newJ = config.Timeline.MaxCalloutLength
		}

		(*calloutLengths)[i] = newI
		(*calloutLengths)[j] = newJ
	} else {
		newI := (*calloutLengths)[i] + adjustment/2
		newJ := (*calloutLengths)[j] - adjustment/2

		if newJ < config.Timeline.MinCalloutLength {
			newJ = config.Timeline.MinCalloutLength
		}
		// Allow full range up to MaxCalloutLength instead of artificial cap
		if newI > config.Timeline.MaxCalloutLength {
			newI = config.Timeline.MaxCalloutLength
		}

		(*calloutLengths)[i] = newI
		(*calloutLengths)[j] = newJ
	}
}

// resolveHorizontalCollision adjusts horizontal positions to separate events
func resolveHorizontalCollision(i, j int, positions *[]int, overlapWidth int, events []TimelineEvent, config Config, minX, maxX int) {
	adjustment := (overlapWidth / 2) + 15 // Add buffer

	// Determine chronological order to maintain timeline sequence
	isBefore := events[i].Timestamp.Before(events[j].Timestamp)

	if isBefore {
		// i is chronologically before j, so i should be to the left, j to the right
		newI := (*positions)[i] - adjustment
		newJ := (*positions)[j] + adjustment

		// Ensure i stays left of j to maintain chronological order
		if newI >= newJ {
			// If the adjustment would reverse chronological order,
			// place them with minimum spacing while preserving order
			midPoint := ((*positions)[i] + (*positions)[j]) / 2
			newI = midPoint - adjustment
			newJ = midPoint + adjustment
		}

		// Ensure text stays within boundaries
		textWidthI := estimateEventTextWidth(events[i], config)
		textWidthJ := estimateEventTextWidth(events[j], config)

		if newI-textWidthI/2 < minX {
			newI = minX + textWidthI/2
		}
		if newJ+textWidthJ/2 > maxX {
			newJ = maxX - textWidthJ/2
		}

		// Final check to maintain chronological order
		if newI >= newJ {
			// Force minimal separation while preserving order
			newJ = newI + textWidthI/2 + textWidthJ/2 + 20
			if newJ+textWidthJ/2 > maxX {
				// If we can't fit j to the right, compress both towards center
				newJ = maxX - textWidthJ/2
				newI = newJ - textWidthI/2 - textWidthJ/2 - 20
			}
		}

		(*positions)[i] = newI
		(*positions)[j] = newJ
	} else {
		// j is chronologically before i, so j should be to the left, i to the right
		newI := (*positions)[i] + adjustment
		newJ := (*positions)[j] - adjustment

		// Ensure j stays left of i to maintain chronological order
		if newJ >= newI {
			// If the adjustment would reverse chronological order,
			// place them with minimum spacing while preserving order
			midPoint := ((*positions)[i] + (*positions)[j]) / 2
			newJ = midPoint - adjustment
			newI = midPoint + adjustment
		}

		// Ensure text stays within boundaries
		textWidthI := estimateEventTextWidth(events[i], config)
		textWidthJ := estimateEventTextWidth(events[j], config)

		if newJ-textWidthJ/2 < minX {
			newJ = minX + textWidthJ/2
		}
		if newI+textWidthI/2 > maxX {
			newI = maxX - textWidthI/2
		}

		// Final check to maintain chronological order
		if newJ >= newI {
			// Force minimal separation while preserving order
			newI = newJ + textWidthJ/2 + textWidthI/2 + 20
			if newI+textWidthI/2 > maxX {
				// If we can't fit i to the right, compress both towards center
				newI = maxX - textWidthI/2
				newJ = newI - textWidthJ/2 - textWidthI/2 - 20
			}
		}

		(*positions)[i] = newI
		(*positions)[j] = newJ
	}
}

// resolveHorizontalCollisionMinimal adjusts horizontal positions with minimal movement to preserve time proportionality
func resolveHorizontalCollisionMinimal(i, j int, positions *[]int, overlapWidth int, events []TimelineEvent, config Config, minX, maxX int) {
	// Use much smaller adjustments to minimize disruption of time proportionality
	adjustment := max(overlapWidth/2+3, 5) // Minimal adjustment, but at least 5 pixels

	// Determine chronological order to maintain timeline sequence
	isBefore := events[i].Timestamp.Before(events[j].Timestamp)

	if isBefore {
		// i is chronologically before j, so i should be to the left, j to the right
		newI := (*positions)[i] - adjustment/2
		newJ := (*positions)[j] + adjustment/2

		// Ensure text stays within boundaries
		textWidthI := estimateEventTextWidth(events[i], config)
		textWidthJ := estimateEventTextWidth(events[j], config)

		if newI-textWidthI/2 < minX {
			newI = minX + textWidthI/2
		}
		if newJ+textWidthJ/2 > maxX {
			newJ = maxX - textWidthJ/2
		}

		// Final check to maintain chronological order
		if newI >= newJ {
			// Force minimal separation while preserving order
			newJ = newI + max(textWidthI, textWidthJ)/2 + 10
			if newJ+textWidthJ/2 > maxX {
				// If we can't fit j to the right, compress both towards center
				newJ = maxX - textWidthJ/2
				newI = newJ - max(textWidthI, textWidthJ)/2 - 10
			}
		}

		(*positions)[i] = newI
		(*positions)[j] = newJ
	} else {
		// j is chronologically before i, so j should be to the left, i to the right
		newI := (*positions)[i] + adjustment/2
		newJ := (*positions)[j] - adjustment/2

		// Ensure text stays within boundaries
		textWidthI := estimateEventTextWidth(events[i], config)
		textWidthJ := estimateEventTextWidth(events[j], config)

		if newJ-textWidthJ/2 < minX {
			newJ = minX + textWidthJ/2
		}
		if newI+textWidthI/2 > maxX {
			newI = maxX - textWidthI/2
		}

		// Final check to maintain chronological order
		if newJ >= newI {
			// Force minimal separation while preserving order
			newI = newJ + max(textWidthI, textWidthJ)/2 + 10
			if newI+textWidthI/2 > maxX {
				// If we can't fit i to the right, compress both towards center
				newI = maxX - textWidthI/2
				newJ = newI - max(textWidthI, textWidthJ)/2 - 10
			}
		}

		(*positions)[i] = newI
		(*positions)[j] = newJ
	}
}

// absTimeDuration returns the absolute value of a time duration.
// For negative durations, it returns the positive equivalent.
func absTimeDuration(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}

// min returns the smaller of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max returns the larger of two integers.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// TextBounds represents the dimensions of a text element
type TextBounds struct {
	Width  int
	Height int
}

// estimateTextBounds calculates the approximate bounding box of text.
// It returns the width and height in pixels based on the text content and font size.
// Uses conservative estimates with average character width of 0.7 * fontSize
// and line height of 1.5 * fontSize for better spacing calculations.
func estimateTextBounds(text string, fontSize int) TextBounds {
	// More conservative estimates for better spacing
	avgCharWidth := float64(fontSize) * 0.7 // Slightly wider characters
	lineHeight := float64(fontSize) * 1.5   // More generous line height

	return TextBounds{
		Width:  int(float64(len(text)) * avgCharWidth),
		Height: int(lineHeight),
	}
}

// estimateWrappedTextBounds calculates bounds for wrapped text
func estimateWrappedTextBounds(lines []string, fontSize int) TextBounds {
	maxWidth := 0
	for _, line := range lines {
		lineBounds := estimateTextBounds(line, fontSize)
		if lineBounds.Width > maxWidth {
			maxWidth = lineBounds.Width
		}
	}

	lineHeight := int(float64(fontSize) * 1.2)
	totalHeight := len(lines) * lineHeight

	return TextBounds{
		Width:  maxWidth,
		Height: totalHeight,
	}
}

// drawEventWithCallout draws a single event with a pre-calculated callout length
func drawEventWithCallout(svg *strings.Builder, event TimelineEvent, x, y int, config Config, index int, allPositions []int, calloutLength int) {
	// Determine if event should be above or below the timeline
	above := index%2 == 0

	// Calculate vertical offset from timeline
	if !above {
		calloutLength = -calloutLength
	}

	eventY := y + calloutLength

	// Store the original eventY for text positioning
	textStartY := eventY

	// For below-timeline events, adjust eventY (line endpoint) to provide clearance above the first text element
	if !above {
		// Get the first text element to determine its height
		columnOrder := getColumnOrder(config)
		for _, elementName := range columnOrder {
			text := getElementText(event, elementName, config)
			if text != "" {
				style := getColumnStyle(elementName, config)
				bounds := estimateTextBounds(text, style.FontSize)
				// Move the callout endpoint DOWN (closer to timeline) to create a gap above the text
				// Use configurable gap between callout line end and text start
				eventY += bounds.Height + config.Timeline.TextElementPadding + config.Timeline.CalloutTextGap
				break
			}
		}
	} else {
		// For above-timeline events, adjust eventY (line endpoint) to provide clearance above the first text element
		// Get the first text element to determine its height
		columnOrder := getColumnOrder(config)
		for _, elementName := range columnOrder {
			text := getElementText(event, elementName, config)
			if text != "" {
				style := getColumnStyle(elementName, config)
				bounds := estimateTextBounds(text, style.FontSize)
				// Move the callout endpoint UP (closer to timeline) to create a gap above the text
				// Use configurable gap between callout line end and text start
				eventY -= bounds.Height + config.Timeline.TextElementPadding + config.Timeline.CalloutTextGap
				break
			}
		}
	}

	// Draw smart connecting line (stepped for better visual clarity)
	if absInt(calloutLength) > config.Timeline.MinCalloutLength+10 {
		// For longer callouts, use a stepped line to reduce visual clutter
		midY := y + (calloutLength / 3) // First segment
		svg.WriteString(fmt.Sprintf(`<path d="M%d,%d L%d,%d L%d,%d" stroke="%s" stroke-width="1" fill="none"/>`,
			x, y, x, midY, x, eventY, config.Colors.Timeline))
	} else {
		// For short callouts, use simple straight line
		svg.WriteString(fmt.Sprintf(`<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="%s" stroke-width="1"/>`,
			x, y, x, eventY, config.Colors.Timeline))
	}

	// Draw event marker
	drawEventMarker(svg, x, y, config)

	// Draw title using configurable positioning with the original eventY
	positions := calculateConfigurableTextPositions(event, textStartY, above, config)

	// Draw each text element according to display_order
	columnOrder := getColumnOrder(config)
	for _, elementName := range columnOrder {
		if position, exists := positions[elementName]; exists {
			text := getElementText(event, elementName, config)
			if text != "" {
				style := getColumnStyle(elementName, config)
				debugPrint("Drawing %s '%s' at position (%d, %d) with style: %s %dpx %s",
					elementName, text, x, position, style.FontFamily, style.FontSize, style.Color)

				// Use inline styling for maximum flexibility
				svg.WriteString(fmt.Sprintf(`<text x="%d" y="%d" text-anchor="middle" font-family="%s" font-size="%d" font-weight="%s" fill="%s">%s</text>`,
					x, position, style.FontFamily, style.FontSize, style.FontWeight, style.Color, escapeXML(text)))
			}
		}
	}
}

// drawEvent draws a single event on the timeline with configurable text elements
func drawEvent(svg *strings.Builder, event TimelineEvent, x, y int, config Config, index int, allPositions []int) {
	// Determine if event should be above or below the timeline
	above := index%2 == 0

	// Calculate callout length based on collision avoidance and boundary constraints
	calloutLength := calculateCalloutLength(x, index, allPositions, above, config, y)

	// Calculate vertical offset from timeline
	if !above {
		calloutLength = -calloutLength
	}

	eventY := y + calloutLength

	// Draw connecting line
	svg.WriteString(fmt.Sprintf(`<line x1="%d" y1="%d" x2="%d" y2="%d" stroke="%s" stroke-width="1"/>`,
		x, y, x, eventY, config.Colors.Timeline))

	// Draw event marker
	drawEventMarker(svg, x, y, config)

	// Draw title using configurable positioning
	positions := calculateConfigurableTextPositions(event, eventY, above, config)

	// Draw each text element according to display_order
	columnOrder := getColumnOrder(config)
	for _, elementName := range columnOrder {
		if position, exists := positions[elementName]; exists {
			text := getElementText(event, elementName, config)
			if text != "" {
				style := getColumnStyle(elementName, config)
				debugPrint("Drawing %s '%s' at position (%d, %d) with style: %s %dpx %s",
					elementName, text, x, position, style.FontFamily, style.FontSize, style.Color)

				// Use inline styling for maximum flexibility
				svg.WriteString(fmt.Sprintf(`<text x="%d" y="%d" text-anchor="middle" font-family="%s" font-size="%d" font-weight="%s" fill="%s">%s</text>`,
					x, position, style.FontFamily, style.FontSize, style.FontWeight, style.Color, escapeXML(text)))
			}
		}
	}
}

// wrapText wraps an array of words into lines that don't exceed maxWidth characters.
// It takes a slice of words and returns a slice of strings, where each string
// represents a line that fits within the specified maximum width.
// Words are never broken - if a single word exceeds maxWidth, it will be placed
// on its own line regardless of the width constraint.
func wrapText(words []string, maxWidth int) []string {
	if len(words) == 0 {
		return []string{}
	}

	var lines []string
	var currentLine strings.Builder

	for _, word := range words {
		if currentLine.Len() == 0 {
			currentLine.WriteString(word)
		} else if currentLine.Len()+1+len(word) <= maxWidth {
			currentLine.WriteString(" " + word)
		} else {
			lines = append(lines, currentLine.String())
			currentLine.Reset()
			currentLine.WriteString(word)
		}
	}

	if currentLine.Len() > 0 {
		lines = append(lines, currentLine.String())
	}

	return lines
}

// escapeXML escapes special XML characters in a string to ensure valid SVG output.
// It replaces XML special characters (&, <, >, ", ') with their corresponding
// XML entity references (&amp;, &lt;, &gt;, &quot;, &apos;) to prevent
// malformed XML when the string is embedded in SVG content.
func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

// getOutputFilename determines the output filename for the SVG file.
// If outputFile is provided and not empty, it returns that filename.
// Otherwise, it derives the filename from the CSV file by replacing
// the extension with .svg (e.g., "data.csv" becomes "data.svg").
func getOutputFilename(csvFile, outputFile string) string {
	if outputFile != "" {
		return outputFile
	}

	// Use CSV filename with .svg extension
	base := filepath.Base(csvFile)
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext) + ".svg"
}

func main() {
	// Parse command line arguments
	debugFlag := flag.Bool("debug", false, "Enable debug mode for verbose output")
	csvFile := flag.String("csv", "", "CSV file with timeline data (required)")
	configFile := flag.String("config", "", "YAML configuration file (optional)")
	outputFile := flag.String("output", "", "Output SVG filename (optional)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		fmt.Fprintf(os.Stderr, "  --debug             Enable debug mode for verbose output\n")
		fmt.Fprintf(os.Stderr, "  --csv <file>        CSV file with timeline data (required)\n")
		fmt.Fprintf(os.Stderr, "  --config <file>     YAML configuration file (optional)\n")
		fmt.Fprintf(os.Stderr, "  --output <file>     Output SVG filename (optional)\n")
		fmt.Fprintf(os.Stderr, "\nThe CSV file should have columns for timestamp and other data.\n")
		fmt.Fprintf(os.Stderr, "If no config file is specified, default settings will be used.\n")
		fmt.Fprintf(os.Stderr, "If no output file is specified, the CSV filename with .svg extension will be used.\n")
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  %s --csv timeline.csv --config config.yaml --output timeline.svg\n", os.Args[0])
	}

	flag.Parse()
	debugMode = *debugFlag

	// Validate required arguments
	if *csvFile == "" {
		fmt.Fprintf(os.Stderr, "Error: CSV file is required. Use --csv to specify the file.\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Load configuration
	config, err := loadConfig(*configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}
	debugPrint("Configuration loaded. Font size: %d, Show dates: %t", config.Font.Size, config.Timeline.ShowDates)

	// Parse CSV file
	events, err := parseCSV(*csvFile, config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing CSV file: %v\n", err)
		os.Exit(1)
	}
	debugPrint("Parsed %d events from %s", len(events), *csvFile)

	if len(events) == 0 {
		fmt.Fprintf(os.Stderr, "Error: No events found in CSV file\n")
		os.Exit(1)
	}

	fmt.Printf("Loaded %d events from %s\n", len(events), *csvFile)

	// Generate SVG
	svgContent := generateSVG(events, config)
	if svgContent == "" {
		fmt.Fprintf(os.Stderr, "Error: Failed to generate SVG content\n")
		os.Exit(1)
	}

	// Determine output filename
	outputPath := getOutputFilename(*csvFile, *outputFile)

	// Write SVG file
	err = os.WriteFile(outputPath, []byte(svgContent), 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing SVG file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Timeline SVG generated successfully: %s\n", outputPath)
}

// calculateCalloutLength determines the optimal callout line length for collision avoidance with boundary constraints
func calculateCalloutLength(x, index int, allPositions []int, above bool, config Config, timelineY int) int {
	if !config.Timeline.AvoidTextOverlap {
		return config.Timeline.MinCalloutLength
	}

	// Find events on the same side of the timeline that might cause vertical collisions
	sameHeightEvents := []struct {
		index int
		x     int
	}{}

	for i, pos := range allPositions {
		eventAbove := i%2 == 0
		if eventAbove == above {
			sameHeightEvents = append(sameHeightEvents, struct {
				index int
				x     int
			}{i, pos})
		}
	}

	// Sort by x position for easier collision detection
	sort.Slice(sameHeightEvents, func(i, j int) bool {
		return sameHeightEvents[i].x < sameHeightEvents[j].x
	})

	// Find this event's position in the sorted list
	currentEventIndex := -1
	for i, event := range sameHeightEvents {
		if event.index == index {
			currentEventIndex = i
			break
		}
	}

	if currentEventIndex == -1 {
		return config.Timeline.MinCalloutLength
	}

	// Calculate callout length based on horizontal proximity to other events on same side
	baseLength := config.Timeline.MinCalloutLength
	lengthRange := config.Timeline.MaxCalloutLength - config.Timeline.MinCalloutLength

	// Check for nearby events on the same side and determine required separation
	minTextSpacing := config.Timeline.MinTextSpacing // Use actual configured spacing

	// Count how many events are within collision distance
	collisionRisk := 0
	for i, event := range sameHeightEvents {
		if i != currentEventIndex {
			distance := absInt(event.x - x)
			// Use a more sensitive threshold for collision detection
			collisionThreshold := minTextSpacing * 3 // 3x the minimum spacing for early detection
			if distance < collisionThreshold {
				collisionRisk++
				debugPrint("Event %d: nearby event at distance %d (threshold %d)",
					index, distance, collisionThreshold)
			}
		}
	}

	debugPrint("Event %d: collisionRisk=%d, sameHeightEvents=%d", index, collisionRisk, len(sameHeightEvents))

	// Check for very close events (within 30 pixels) to force aggressive level usage
	veryCloseEvents := 0
	for i, event := range sameHeightEvents {
		if i != currentEventIndex {
			distance := absInt(event.x - x)
			if distance < 30 { // Very close threshold
				veryCloseEvents++
			}
		}
	}

	// Calculate staggered heights based on position in the collision group
	if collisionRisk > 0 || len(sameHeightEvents) > 4 {
		// Create alternating heights for closely spaced events
		levelSpacing := lengthRange / max(config.Timeline.CalloutLevels, 3) // At least 3 levels

		// Use more aggressive level distribution for clustered events
		heightLevel := 0
		totalEventsOnSide := len(sameHeightEvents)

		if veryCloseEvents >= 2 {
			// Force all levels when events are at nearly identical positions
			heightLevel = currentEventIndex % config.Timeline.CalloutLevels
			debugPrint("Event %d: Using ALL %d levels due to %d very close events (within 30px)",
				index, config.Timeline.CalloutLevels, veryCloseEvents)
		} else if totalEventsOnSide > 6 || collisionRisk >= 3 {
			// For very crowded areas, distribute across ALL available levels
			heightLevel = currentEventIndex % config.Timeline.CalloutLevels
			debugPrint("Event %d: Using all %d levels due to high density (%d events, collision risk %d)",
				index, config.Timeline.CalloutLevels, totalEventsOnSide, collisionRisk)
		} else if totalEventsOnSide > 4 || collisionRisk >= 2 {
			// For moderately crowded areas, use 3/4 of available levels
			usableLevels := max(config.Timeline.CalloutLevels*3/4, 4)
			heightLevel = currentEventIndex % usableLevels
			debugPrint("Event %d: Using %d of %d levels for moderate density (%d events, collision risk %d)",
				index, usableLevels, config.Timeline.CalloutLevels, totalEventsOnSide, collisionRisk)
		} else {
			// For light collision areas, use half the configured levels
			halfLevels := max(config.Timeline.CalloutLevels/2, 2) // At least 2 levels
			heightLevel = currentEventIndex % halfLevels
			debugPrint("Event %d: Using %d of %d levels for light density (%d events, collision risk %d)",
				index, halfLevels, config.Timeline.CalloutLevels, totalEventsOnSide, collisionRisk)
		}

		additionalHeight := heightLevel * levelSpacing
		baseLength += additionalHeight

		debugPrint("Event %d: collisionRisk=%d, heightLevel=%d, additionalHeight=%d",
			index, collisionRisk, heightLevel, additionalHeight)
	} // Add extra spacing for very crowded areas
	if collisionRisk > 2 {
		densityBonus := (collisionRisk - 2) * 20 // Increased from 15 to 20
		baseLength += densityBonus
		debugPrint("Event %d: adding density bonus %d for %d nearby events",
			index, densityBonus, collisionRisk)
	}

	// Don't exceed maximum length
	if baseLength > config.Timeline.MaxCalloutLength {
		baseLength = config.Timeline.MaxCalloutLength
	}

	// Apply boundary constraints to prevent text overflow
	maxSafeCallout := calculateMaxSafeCallout(timelineY, above, config)
	if baseLength > maxSafeCallout {
		baseLength = maxSafeCallout
	}

	debugPrint("Event %d (x=%d, above=%v): final callout length=%d", index, x, above, baseLength)
	return baseLength
}

// calculateMaxSafeCallout determines the maximum safe callout length to prevent text overflow.
// It calculates the available vertical space between the timeline and the SVG boundaries,
// taking into account the estimated text height for title, date, and notes elements.
// For above-timeline events, it ensures text doesn't exceed the top margin.
// For below-timeline events, it ensures text doesn't exceed the bottom margin.
// Returns a callout length that keeps all text within the SVG bounds.
func calculateMaxSafeCallout(timelineY int, above bool, config Config) int {
	// Estimate text height based on configuration
	// Title: font size + 2 (bold), Date: font size - 1, Notes: estimated 4 lines max of font size - 2
	titleHeight := config.Font.Size + 2 + 5 // +5 for spacing
	dateHeight := 0
	if config.Timeline.ShowDates {
		dateHeight = config.Font.Size - 1 + 5 // +5 for spacing
	}
	notesHeight := (config.Font.Size-2)*4 + (3 * 4) // 4 lines max with 3px spacing between

	estimatedTextHeight := titleHeight + dateHeight + notesHeight + 20 // +20 buffer

	if above {
		// For above timeline, ensure text doesn't go beyond top margin
		availableSpace := timelineY - config.Layout.MarginTop
		maxCallout := availableSpace - estimatedTextHeight
		if maxCallout < config.Timeline.MinCalloutLength {
			maxCallout = config.Timeline.MinCalloutLength
		}
		return maxCallout
	} else {
		// For below timeline, ensure text doesn't go beyond bottom margin
		svgBottom := config.Layout.Height - config.Layout.MarginBottom
		availableSpace := svgBottom - timelineY
		maxCallout := availableSpace - estimatedTextHeight
		if maxCallout < config.Timeline.MinCalloutLength {
			maxCallout = config.Timeline.MinCalloutLength
		}
		return maxCallout
	}
}

// drawEventMarker draws the appropriate marker shape at the specified position on the timeline.
// It supports multiple marker shapes (circle, square, diamond, triangle) with configurable
// size, fill color, stroke color, and stroke width. The marker is rendered as SVG elements
// and appended to the provided string builder.
//
// Supported shapes:
//   - "circle": Circular marker with configurable radius
//   - "square": Rectangular marker with equal width and height
//   - "diamond": Diamond-shaped marker created using a rotated square polygon
//   - "triangle": Upward-pointing triangular marker
//   - Default: Falls back to circle for unknown shapes
func drawEventMarker(svg *strings.Builder, x, y int, config Config) {
	size := config.EventMarker.Size
	fillColor := config.EventMarker.FillColor
	strokeColor := config.EventMarker.StrokeColor
	strokeWidth := config.EventMarker.StrokeWidth

	switch strings.ToLower(config.EventMarker.Shape) {
	case "circle":
		svg.WriteString(fmt.Sprintf(`<circle cx="%d" cy="%d" r="%d" fill="%s" stroke="%s" stroke-width="%d"/>`,
			x, y, size, fillColor, strokeColor, strokeWidth))

	case "square":
		halfSize := size
		svg.WriteString(fmt.Sprintf(`<rect x="%d" y="%d" width="%d" height="%d" fill="%s" stroke="%s" stroke-width="%d"/>`,
			x-halfSize, y-halfSize, size*2, size*2, fillColor, strokeColor, strokeWidth))

	case "diamond":
		// Draw diamond as a rotated square using polygon
		svg.WriteString(fmt.Sprintf(`<polygon points="%d,%d %d,%d %d,%d %d,%d" fill="%s" stroke="%s" stroke-width="%d"/>`,
			x, y-size, // top
			x+size, y, // right
			x, y+size, // bottom
			x-size, y, // left
			fillColor, strokeColor, strokeWidth))

	case "triangle":
		// Draw upward pointing triangle
		height := int(float64(size) * 1.5) // Make triangle a bit taller for better visibility
		svg.WriteString(fmt.Sprintf(`<polygon points="%d,%d %d,%d %d,%d" fill="%s" stroke="%s" stroke-width="%d"/>`,
			x, y-height, // top point
			x-size, y+height/2, // bottom left
			x+size, y+height/2, // bottom right
			fillColor, strokeColor, strokeWidth))

	default:
		// Default to circle if unknown shape
		svg.WriteString(fmt.Sprintf(`<circle cx="%d" cy="%d" r="%d" fill="%s" stroke="%s" stroke-width="%d"/>`,
			x, y, size, fillColor, strokeColor, strokeWidth))
	}
}

// absInt returns the absolute value of an integer.
// For negative integers, it returns the positive equivalent.
// For positive integers or zero, it returns the value unchanged.
func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
