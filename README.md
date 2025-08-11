# Timeline2SVG

A Go application that generates SVG timeline visualizations from CSV data.

## Documentation Notice

This documentation and code comments were generated and maintained with the assistance of AI (GitHub Copilot). The AI helped with:

- Writing comprehensive Go documentation comments following Go conventions
- Creating detailed README documentation with usage examples
- Modernizing the command-line interface from positional to named arguments
- Ensuring documentation accuracy through automated testing validation

## Quick Start

```bash
# Download and build
go build -o timeline2svg

# Basic usage with named arguments
timeline2svg --csv your-data.csv

# With configuration and custom output
timeline2svg --csv your-data.csv --config config.yaml --output timeline.svg

# Enable debug mode to see positioning algorithms
timeline2svg --debug --csv your-data.csv
```

## Features

- Converts CSV data with timestamp, title, and notes into beautiful SVG timelines
- **Time-proportional positioning**: Events are positioned based on actual time intervals for intuitive visualization
- **Advanced Temporal Clustering**: Automatically detects events that occur close in time (within 2 hours) and applies specialized positioning algorithms to visualize temporal relationships
- **Intelligent Constraint-Based Positioning**: Two-phase positioning system that balances time proportionality with collision avoidance
- **Backward-Working Callout Optimization**: Tests multiple callout height combinations to find the optimal layout that minimizes temporal distortion
- **Ultra-Aggressive Clustering Mode**: For tightly clustered events, allows controlled text overlap to preserve temporal relationships
- **Dynamic Temporal Distortion Analysis**: Measures and minimizes positioning errors using weighted algorithms that prioritize temporal accuracy
- **Configurable event markers**: Choose from circle, square, diamond, or triangle shapes with custom colors
- **Advanced collision avoidance**: Multi-level callout system with variable-length lines prevents text overlap
- **Smart boundary constraints**: Callout lengths are automatically limited to prevent text from overflowing beyond SVG margins
- **Smart text stacking**: Events close in time use different callout lengths to stack text at multiple levels
- **Debug mode**: Comprehensive logging of positioning algorithms and constraint solving process
- Configurable styling through YAML configuration files
- Flexible timestamp format support
- Automatic text wrapping for long notes
- Alternating event positioning (above/below timeline)
- Optional date and time display
- Horizontal buffer configuration for timeline spacing

## Usage

```bash
timeline2svg [options]
```

### Options

- `--csv <file>` (required): CSV file containing timeline data
- `--config <file>` (optional): YAML configuration file for styling
- `--output <file>` (optional): Output SVG filename
- `--debug`: Enable debug mode for verbose output showing positioning algorithms, constraint solving, and temporal clustering analysis

If no config file is specified, default settings will be used.
If no output file is specified, the CSV filename with `.svg` extension will be used.

### Examples

```bash
# Basic usage with default settings
timeline2svg --csv events.csv

# With custom configuration
timeline2svg --csv events.csv --config my-config.yaml

# With custom configuration and output filename
timeline2svg --csv events.csv --config my-config.yaml --output timeline.svg

# Debug mode to see positioning algorithms and temporal clustering analysis
timeline2svg --debug --csv events.csv

# Debug mode with custom configuration and output filename
timeline2svg --debug --csv events.csv --config my-config.yaml --output timeline.svg

# Arguments can be specified in any order
timeline2svg --output timeline.svg --debug --csv events.csv --config my-config.yaml
```

## Temporal Clustering & Positioning Algorithms

Timeline2SVG uses sophisticated algorithms to handle events that occur close in time:

### Temporal Cluster Detection

- **Automatic Detection**: Events occurring within 2 hours of each other are automatically identified as temporal clusters
- **Specialized Processing**: Cluster events receive different positioning treatment to preserve temporal relationships
- **Visual Cohesion**: Maintains tight visual grouping for temporally related events

### Constraint-Based Positioning System

1. **Phase 1 - Time-Proportional Positioning**: Initial positions based on actual time intervals
2. **Phase 2 - Backward Optimization**: Tests multiple callout height combinations to minimize temporal distortion
3. **Phase 3 - Constraint Solving**: Applies collision avoidance while preserving temporal relationships

### Ultra-Aggressive Clustering Mode

For events in temporal clusters:

- **Controlled Text Overlap**: Allows minimal text overlap to preserve tight temporal grouping
- **Minimum Separation**: Maintains 20px minimum separation for readability
- **Temporal Priority**: Prioritizes temporal accuracy over traditional collision avoidance

## CSV Format

The CSV file must contain 3 columns:

1. `timestamp` - Date/time in various supported formats
2. `title` - Event title
3. `notes` - Optional event description

### Supported Timestamp Formats

- RFC3339: `2006-01-02T15:04:05Z07:00`
- ISO DateTime: `2006-01-02 15:04:05`
- ISO DateTime Short: `2006-01-02 15:04`
- ISO Date: `2006-01-02`
- US Format: `01/02/2006 15:04:05`, `01/02/2006 15:04`, `01/02/2006`
- European Format: `02/01/2006 15:04:05`, `02/01/2006 15:04`, `02/01/2006`

### Example CSV

```csv
timestamp,title,notes
2024-01-15 09:00,Project Start,Initial project kickoff meeting
2024-02-01 14:30,Requirements Complete,All requirements documented
2024-03-15 11:15,Design Phase,System architecture finalized
2024-04-30T08:45:00Z,Development Sprint 1,Core functionality implementation begins
```

## Configuration

Configuration is done through YAML files. See `detailed-styling-config.yaml` for an example with advanced styling, or `temporal-clustering-config.yaml` for a configuration optimized for temporal clustering visualization.

### Configuration Structure

```yaml
font:
  family: "Arial, sans-serif"  # Font family for text
  size: 12                     # Base font size

colors:
  background: "#ffffff"        # SVG background color
  timeline: "#333333"         # Timeline line color
  events: "#4285f4"           # Event marker color
  text: "#333333"             # Title text color
  notes: "#666666"            # Notes text color

layout:
  width: 1200                 # SVG width in pixels
  height: 800                 # SVG height in pixels
  margin_top: 50              # Top margin
  margin_bottom: 50           # Bottom margin
  margin_left: 100            # Left margin
  margin_right: 100           # Right margin
  event_radius: 8             # Event marker radius
  event_spacing: 120          # Vertical spacing from timeline

timeline:
  line_width: 2               # Timeline line width
  show_dates: true            # Show dates below/above event titles
  show_times: true            # Show times along with dates when available
  horizontal_buffer: 50       # Horizontal buffer space before first and after last event
  avoid_text_overlap: true    # Enable collision avoidance for overlapping text
  min_text_spacing: 80        # Minimum horizontal spacing to trigger overlap avoidance
                             # (Set lower values like 10 for time-proportional positioning)
  min_callout_length: 60      # Minimum length of vertical callout lines
  max_callout_length: 180     # Maximum length of vertical callout lines
  callout_levels: 4           # Number of different callout levels for stacking
                             # (Higher values like 8 provide more positioning options)

event_marker:
  shape: "circle"             # Marker shape: circle, square, diamond, triangle
  size: 8                     # Size of the marker in pixels
  fill_color: "#4285f4"       # Fill color of the marker
  stroke_color: "#333333"     # Stroke (border) color of the marker
  stroke_width: 2             # Width of the marker border
```

## Building

```bash
go build -o timeline2svg.exe
```

## Testing

Use the provided sample data:

```bash
# Basic timeline generation
go run main.go --csv sample-data.csv

# Test temporal clustering with debug output
go run main.go --debug --csv close-events-sample.csv

# With custom output filename
go run main.go --csv close-events-sample.csv --output my-timeline.svg

# View the generated timeline
start close-events-sample.svg  # Windows
open close-events-sample.svg   # macOS
xdg-open close-events-sample.svg  # Linux
```

This will generate SVG files with the timeline visualization.

## Algorithm Details

### Temporal Distortion Measurement

The system measures how much event positioning deviates from true time proportionality:

- **Dynamic Clustering Analysis**: Automatically adjusts weights based on detected temporal clusters
- **Weighted Distortion Calculation**: Events in clusters get higher weights (4.0x) for accuracy
- **Optimization Target**: Minimizes weighted temporal distortion while maintaining collision avoidance

### Constraint Solver Process

1. **Constraint Generation**: Creates minimum separation requirements between overlapping events
2. **Buffer Adjustment**: Uses different separation buffers based on temporal clustering:
   - **Cluster Events**: -50px buffer (allows controlled overlap)
   - **Mixed Events**: 5px buffer (moderate spacing)
   - **Non-Cluster Events**: 15px buffer (standard spacing)
3. **Iterative Solving**: Adjusts positions until all constraints are satisfied

### Performance Characteristics

- **Callout Combinations**: Tests up to 8 different callout height patterns
- **Optimization Scope**: Focuses computation on temporal clusters for efficiency
- **Convergence**: Constraint solver typically converges in 1-2 iterations
- **Scalability**: Optimized for timelines with 5-50 events

## Dependencies

- Go 1.21+
- gopkg.in/yaml.v3

## Best Practices

### For Time-Proportional Layouts

- Set `min_text_spacing` to a low value (10-20) in your config
- Increase `callout_levels` to 6-8 for more positioning flexibility
- Use `--debug` mode to monitor temporal distortion measurements

### For Temporal Clustering

- Events within 2 hours are automatically clustered
- Debug mode shows cluster detection and optimization progress
- Ultra-aggressive constraints allow controlled text overlap for tight clustering

### Configuration Recommendations

```yaml
timeline:
  min_text_spacing: 10      # Allow time-proportional positioning
  callout_levels: 8         # More positioning options
  max_callout_length: 200   # Allow longer callouts for flexibility
```

## Troubleshooting

### Common Issues

#### Text Overlap Despite Collision Avoidance

- This is expected behavior for temporal clusters (events within 2 hours)
- Use `--debug` to see constraint solver decisions
- Adjust `min_text_spacing` to control when overlap avoidance triggers

#### Events Not Time-Proportional

- Check `min_text_spacing` - high values force constraint solving
- Verify timestamp format is correctly parsed
- Use debug mode to see ideal vs. final positions

#### Poor Clustering Visualization

- System automatically detects clusters within 2-hour windows
- Temporal clusters get specialized positioning treatment
- Debug output shows cluster detection and optimization results

### Debug Output Interpretation

- `Temporal distortion`: Lower values indicate better time proportionality
- `Constraint violations`: Shows collision detection results
- `Ultra-aggressive constraints`: Indicates temporal clustering mode active
- `Optimization improvement`: Percentage improvement in temporal accuracy
