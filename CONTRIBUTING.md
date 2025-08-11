# Contributing to Timeline2SVG

Thank you for considering contributing to Timeline2SVG! This document provides guidelines and instructions for contributing to the project.

## Code of Conduct

By participating in this project, you agree to abide by our Code of Conduct. Please treat all contributors and users with respect.

## How to Contribute

### Reporting Bugs

1. Check existing issues to see if the bug has already been reported
2. Use the bug report template when creating a new issue
3. Include as much detail as possible:
   - Sample CSV data that reproduces the issue
   - Configuration files used
   - Command line arguments
   - Expected vs. actual behavior
   - Environment details (OS, Go version)

### Suggesting Features

1. Check existing issues and discussions for similar requests
2. Use the feature request template
3. Clearly describe the use case and benefit
4. Provide examples of how the feature would work

### Contributing Code

#### Prerequisites

- Go 1.21 or higher
- Git
- Make (optional, but recommended)

#### Development Setup

1. Fork the repository
2. Clone your fork:
   ```bash
   git clone https://github.com/yourusername/timeline2svg.git
   cd timeline2svg
   ```
3. Install dependencies:
   ```bash
   go mod download
   ```
4. Install development tools (optional):
   ```bash
   make install-tools
   ```

#### Development Workflow

1. Create a feature branch:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. Make your changes following the coding standards below

3. Test your changes:
   ```bash
   # Run all tests and quality checks
   make dev
   
   # Or run individual steps:
   make test
   make quality
   make build
   make quick-test
   ```

4. Commit your changes:
   ```bash
   git add .
   git commit -m "Add your descriptive commit message"
   ```

5. Push to your fork:
   ```bash
   git push origin feature/your-feature-name
   ```

6. Create a pull request using the PR template

#### Coding Standards

- **Go Code Style**: Follow standard Go formatting (use `gofmt`)
- **Documentation**: All exported functions must have Go doc comments
- **Error Handling**: Proper error handling with descriptive messages
- **Testing**: Add tests for new functionality
- **Commit Messages**: Use clear, descriptive commit messages

#### Code Structure

```
timeline2svg/
├── main.go              # Main application logic
├── go.mod              # Go module definition
├── go.sum              # Go module checksums
├── README.md           # Project documentation
├── .gitignore          # Git ignore rules
├── Makefile            # Build and development tasks
├── .golangci.yml       # Linter configuration
├── .github/            # GitHub templates and workflows
│   ├── workflows/      # CI/CD workflows
│   ├── ISSUE_TEMPLATE/ # Issue templates
│   └── pull_request_template.md
└── sample files/       # Example CSV and config files
```

#### Key Areas of the Code

- **CSV Parsing**: Functions that read and parse CSV input files
- **Temporal Algorithms**: Clustering and positioning logic for time-based layouts
- **SVG Generation**: Functions that create the final SVG output
- **Configuration**: YAML configuration parsing and validation
- **CLI Interface**: Command line argument parsing and help text

#### Adding New Features

When adding new features:

1. **Design First**: For significant changes, create an issue to discuss the design
2. **Backward Compatibility**: Ensure changes don't break existing functionality
3. **Configuration**: If adding configurable options, update the YAML schema
4. **Documentation**: Update README.md and add Go doc comments
5. **Testing**: Add appropriate test cases
6. **Examples**: Provide usage examples in documentation

#### Testing Guidelines

- Test with various CSV formats and edge cases
- Test configuration options and error conditions
- Test on different operating systems if possible
- Include both positive and negative test cases
- Test the generated SVG output manually when possible

### Quality Assurance

All contributions go through the following automated checks:

- **Code Formatting**: `gofmt` and `goimports`
- **Static Analysis**: `go vet`, `staticcheck`, `golangci-lint`
- **Security Scanning**: `govulncheck`
- **Testing**: Unit tests and integration tests
- **Cross-Platform Builds**: Linux, Windows, macOS

You can run these checks locally:

```bash
# Full quality check
make quality

# Individual checks
make fmt      # Format code
make vet      # Run go vet
make lint     # Run golangci-lint
make static   # Run staticcheck
make security # Run security scan
make test     # Run tests
```

### Documentation

- Update README.md for user-facing changes
- Add Go doc comments for all exported functions
- Update configuration examples if adding new options
- Include usage examples for new features

### Release Process

Releases are handled by maintainers:

1. Version tags trigger automated builds
2. Cross-platform binaries are automatically created
3. Release notes are generated from commit messages
4. GitHub releases include binary downloads

## Getting Help

- **Questions**: Open a discussion or issue
- **Problems**: Use the bug report template
- **Ideas**: Use the feature request template

## Recognition

Contributors are recognized in:
- Git commit history
- GitHub contributors page
- Release notes (for significant contributions)

Thank you for contributing to Timeline2SVG! 🚀
