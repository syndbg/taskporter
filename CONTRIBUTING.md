# Contributing

## 🚀 Getting Started

### Prerequisites

- **Go 1.21+** - Taskporter is built with Go
- **Make** - For build automation
- **Git** - For version control

### Development Setup

1. **Fork and Clone**
   ```bash
   git clone https://github.com/yourusername/taskporter.git
   cd taskporter
   ```

2. **Install Dependencies**
   ```bash
   go mod download
   ```

3. **Verify Setup**
   ```bash
   make check
   ```

## 🏗️ Development Workflow

### Building and Testing

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Build the binary
make build

# Run linter
make lint

# Run all checks (test + lint)
make check

# Clean build artifacts
make clean
```

### Code Style and Quality

- **Follow Go conventions** - Use `go fmt`, `golint`, and `go vet`
- **Write tests** - All new code should include comprehensive tests
- **Use testify/require** - For assertions in tests
- **Nested tests** - Use `t.Run()` for test organization
- **One struct per file** - File names should match struct names
- **SOLID principles** - Follow dependency injection patterns
- **No global variables** - Use factory functions for dependency creation

## 🎯 Types of Contributions

### 🐛 Bug Fixes
- Report bugs via [GitHub Issues](https://github.com/syndbg/taskporter/issues) using our bug report template
- Include reproduction steps and environment details
- Submit PRs with clear commit messages

### ✨ New Features
- Request features via [GitHub Issues](https://github.com/syndbg/taskporter/issues) using our feature request template
- Discuss feature ideas in [GitHub Discussions](https://github.com/syndbg/taskporter/discussions)
- Ensure features align with project goals
- Include tests and documentation

### 📚 Documentation
- Fix typos and improve clarity
- Add examples and use cases
- Update README.md, CONTRIBUTING.md, and code comments

### 🔧 IDE Support
- Add support for new IDEs and editors
- Follow the extensible parser architecture
- Include comprehensive test coverage

## 🏗️ Adding New IDE Support

Taskporter is designed to be extensible. To add support for a new IDE:

### 1. Create Parser Package
```bash
mkdir -p internal/parser/youride/
```

Create the following files:
- `parser.go` - Main parser implementation
- `parser_test.go` - Comprehensive tests
- `types.go` - IDE-specific data structures (if needed)

### 2. Implement Parser Interface
Your parser should implement the common task interface:

```go
package youride

import "taskporter/internal/config"

type Parser struct {
    projectRoot string
}

func NewParser(projectRoot string) *Parser {
    return &Parser{projectRoot: projectRoot}
}

func (p *Parser) ParseTasks(configPath string) ([]*config.Task, error) {
    // Implementation here
}
```

### 3. Add Detection Logic
Update `internal/config/project_detector.go`:

```go
func (d *ProjectDetector) DetectYourIDE() ([]*config.Task, error) {
    // Detection and parsing logic
}
```

### 4. Integrate with Commands
Update both `internal/cmd/list.go` and `internal/cmd/run.go` to include your new IDE:

```go
// Add to scanning logic
yourIDETasks, err := detector.DetectYourIDE()
if err == nil {
    allTasks = append(allTasks, yourIDETasks...)
}
```

### 5. Add Comprehensive Tests
- **Unit tests** for parser functionality
- **Integration tests** with real configuration files
- **Test data** in `internal/test/youride-testdata/`
- **Edge cases** and error scenarios

### 6. Update Documentation
- Add IDE support to README.md features section
- Include configuration examples
- Update project structure documentation

## 🧪 Testing Guidelines

### Writing Tests
```go
func TestYourIDEParser(t *testing.T) {
    t.Run("ParseTasks", func(t *testing.T) {
        t.Run("should parse valid configuration", func(t *testing.T) {
            parser := NewParser("/test/project")
            tasks, err := parser.ParseTasks("path/to/config")

            require.NoError(t, err)
            require.NotEmpty(t, tasks)
            require.Equal(t, "expected-task-name", tasks[0].Name)
        })

        t.Run("should handle invalid configuration", func(t *testing.T) {
            // Error case testing
        })
    })
}
```

### Test Data
- Place test configuration files in `internal/test/youride-testdata/`
- Use realistic configuration examples
- Cover various IDE configuration patterns

### Test Coverage
- Aim for >90% test coverage
- Include edge cases and error scenarios
- Test all public functions and methods

## 📝 Commit Guidelines

### Commit Message Format
```
type(scope): brief description

Detailed explanation of the change (if needed)

Fixes #123
```

### Types
- `feat:` - New features
- `fix:` - Bug fixes
- `docs:` - Documentation changes
- `style:` - Code style changes (formatting, etc.)
- `refactor:` - Code refactoring
- `test:` - Adding or updating tests
- `chore:` - Maintenance tasks

### Examples
```
feat(parser): add WebStorm run configuration support

Add support for parsing WebStorm-specific run configurations
including Node.js and npm script configurations.

Fixes #45

---

fix(vscode): handle missing preLaunchTask gracefully

Prevent panic when VSCode launch configuration references
a non-existent preLaunchTask.

Fixes #67

---

docs(readme): add shell completion setup instructions

Include detailed setup instructions for bash, zsh, fish,
and PowerShell completion support.
```

## 🔍 Pull Request Process

### Before Submitting
1. **Run all tests** - `make check` should pass
2. **Update documentation** - If adding features or changing behavior
3. **Add test coverage** - For new functionality
4. **Follow coding style** - Consistent with existing codebase

### PR Description
When creating a pull request, GitHub will automatically populate the description with our PR template that includes sections for summary, type of change, testing checklist, and code style verification.

### Review Process
1. **Automated checks** must pass (CI/CD)
2. **Code review** by maintainers
3. **Testing** of new functionality
4. **Documentation** review if applicable

## 🤝 Code of Conduct

### Our Standards
- **Be respectful** - Treat all contributors with respect
- **Be inclusive** - Welcome contributors from all backgrounds
- **Be constructive** - Provide helpful feedback and suggestions
- **Be patient** - Help newcomers learn and contribute

### Unacceptable Behavior
- Harassment or discriminatory language
- Personal attacks or trolling
- Spam or off-topic discussions
- Any other conduct that would be inappropriate in a professional setting

## 🎯 Architecture Guidelines

### Project Structure
```
taskporter/
├── internal/
│   ├── cmd/               # CLI commands
│   ├── config/            # Configuration types and detection
│   ├── parser/            # IDE-specific parsers
│   │   ├── vscode/        # VSCode tasks & launch configs
│   │   ├── jetbrains/     # JetBrains run configurations
│   │   └── youride/       # Your new IDE parser
│   ├── runner/            # Task execution engine
│   ├── security/          # Security validation (paranoid mode)
│   └── test/              # Test data and fixtures
└── main.go                # Application entry point
```

### Design Principles
- **Dependency Injection** - Use factory functions, avoid globals
- **Single Responsibility** - One struct per file, clear interfaces
- **Testability** - All components easily testable in isolation
- **Extensibility** - Easy to add new IDE support
- **Security** - Optional paranoid mode for validation

### Key Interfaces
```go
// Task represents a unified task across all IDEs
type Task struct {
    Name        string
    Command     string
    Args        []string
    Env         map[string]string
    WorkingDir  string
    Type        TaskType
    Group       string
    Source      string
}

// Parser interface for IDE-specific parsers
type Parser interface {
    ParseTasks(configPath string) ([]*Task, error)
}
```

## 📞 Getting Help

### Community Support
- **GitHub Discussions** - General questions and ideas
- **GitHub Issues** - Bug reports and feature requests
- **Code Review** - Get feedback on your contributions

### Maintainer Contact
- Open an issue for bugs or feature requests
- Start a discussion for general questions
- Tag maintainers in PRs for review

## 🎮 Death Stranding Philosophy

*"Every step you take, you're making a path for someone else to follow."*

Your contributions to Taskporter help build bridges between isolated development environments. Each parser you add, each bug you fix, and each test you write creates a stronger strand in our developer ecosystem.

Thank you for helping connect the development world! 🌉

---

**Ready to contribute?** Start by exploring the codebase, running the tests, and choosing an issue that interests you. Every contribution, no matter how small, helps strengthen the strand!
