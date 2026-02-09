# Technical Architecture

## Design Patterns

### 1. Model-View-Controller (MVC) Pattern

The project loosely follows MVC:

- **Model** (`internal/model/config.go`)
  - Represents application data and business logic
  - Handles configuration persistence (load/save)
  - Provides validation and defaults

- **View** (`internal/tui/form.go`)
  - Bubbletea TUI components
  - User interface rendering
  - Input handling and display

- **Controller** (`cmd/tui.go`, `main.go`)
  - Orchestrates flow between model and view
  - Handles user actions
  - Manages application state

### 2. Template Pattern

Used in infrastructure code generation:

- **OpenTofu Generator** (`internal/generator/tofu.go`)
  - Go text/template for `main.tf` generation
  - Separates template from data
  - Type-safe template execution

- **Ansible Generator** (`internal/generator/ansible.go`)
  - Go text/template for `site.yml` generation
  - Embedded Jinja2 syntax for Ansible variables
  - Escaping strategy for double-templating

### 3. Command Pattern

Used in workflow execution:

- **Runner** (`internal/workflow/runner.go`)
  - Encapsulates deployment commands
  - Provides rollback capability (future)
  - Centralizes error handling

### 4. Builder Pattern

Implicit in the TUI:

- **Model Builder** (`tui.NewModel`)
  - Constructs complex TUI model
  - Initializes multiple text inputs
  - Sets up focus and styling

## Code Organization

### Package Structure

```
github.com/Felipalds/go-kubernetes-helper/
├── cmd/                    # Command implementations
│   └── tui                # TUI command package
├── internal/              # Private application code
│   ├── generator/         # Infrastructure code generators
│   ├── model/             # Data models
│   ├── tui/               # TUI components
│   ├── utils/             # Shared utilities
│   └── workflow/          # Deployment orchestration
└── main.go               # Entry point
```

### Dependency Flow

```
main.go
  ├─> model.LoadConfig()
  ├─> cmd.RunTUI()
  │     └─> tui.NewModel()
  │           └─> textinput.New() (bubbles)
  ├─> model.Config.Save()
  └─> workflow.NewRunner()
        ├─> utils.InitLogger()
        └─> Runner.Run()
              ├─> generator.GenerateTofu()
              ├─> runCommand("tofu init")
              ├─> runCommand("tofu apply")
              ├─> getTofuOutput()
              ├─> generator.GenerateAnsible()
              └─> runCommand("ansible-playbook")
```

## Key Components Deep Dive

### TUI Model (internal/tui/form.go)

**State Management**:
```go
type Model struct {
    focusIndex int              // Current focused input (0-12)
    inputs     []textinput.Model // 12 text inputs + submit button
    config     *model.Config     // Reference to shared config
    done       bool              // Submission flag
    quitting   bool              // Cancellation flag
}
```

**Update Cycle**:
1. Receives `tea.Msg` (keyboard events)
2. Updates focus index based on navigation keys
3. Delegates input updates to active textinput
4. Returns new model state and commands

**Rendering**:
- Focused input: Pink foreground
- Blurred input: Gray foreground
- Submit button: Highlighted when focused
- Help text at bottom

### Workflow Runner (internal/workflow/runner.go)

**Responsibilities**:
- Generate infrastructure code
- Execute external commands (tofu, ansible-playbook)
- Capture and log output
- Parse Tofu outputs (JSON)
- Generate dynamic inventory

**Error Handling**:
- Command failures return wrapped errors
- Output always logged (success or failure)
- Early exit on first error
- User-friendly error messages

**Inventory Generation**:
```
[init]
<first_ip> ansible_user=ubuntu ansible_ssh_common_args='...' ansible_ssh_private_key_file=<path>

[join]
<second_ip> ansible_user=ubuntu ...
<third_ip> ansible_user=ubuntu ...
```

### Configuration Model (internal/model/config.go)

**Validation Strategy**:
- Load from file if exists
- Apply defaults for missing values
- Auto-fix known invalid values (version migration)
- Validate on save (JSON serialization check)

**Security**:
- File permissions: 0600 (owner read/write only)
- No credentials in logs
- Password masking in TUI

## External Dependencies

### Infrastructure Tools

**OpenTofu** (required):
- Terraform-compatible IaC tool
- Commands: `init`, `apply`, `output`
- Expected in PATH

**Ansible** (required):
- Configuration management tool
- Commands: `ansible-playbook`
- Expected in PATH

### AWS Resources (required):
- Valid AWS credentials (access key + secret)
- Existing VPC with subnet
- Security group with required ports open
- SSH key pair registered in EC2

### Go Libraries

**Bubbletea Ecosystem**:
- `bubbletea`: TUI framework (Elm architecture)
- `bubbles`: Pre-built TUI components (textinput)
- `lipgloss`: Styling and layout

**Other**:
- `cobra`: CLI framework
- `zap`: Structured logging

## Data Flow

### Configuration Flow

```
config.json (disk)
      ↓
  LoadConfig()
      ↓
  Config struct (memory)
      ↓
  TUI Model.config (reference)
      ↓
  User edits in TUI
      ↓
  updateConfig() (on submit)
      ↓
  Config.Save()
      ↓
  config.json (disk) [updated]
      ↓
  Template execution
      ↓
  main.tf, site.yml (generated)
```

### Deployment Flow

```
TUI Submit
    ↓
Config saved
    ↓
GenerateTofu() → main.tf
    ↓
tofu init
    ↓
tofu apply → EC2 instances created
    ↓
tofu output -json instance_ips → []string
    ↓
Generate hosts.ini from IPs
    ↓
GenerateAnsible() → site.yml
    ↓
ansible-playbook → Cluster deployed
    ↓
Complete
```

## Error Handling Strategy

1. **Configuration Errors**:
   - Invalid JSON: Return error, don't proceed
   - Missing file: Use defaults, warn user
   - Invalid values: Auto-fix or use defaults

2. **Command Errors**:
   - Capture stderr/stdout
   - Log full output
   - Return wrapped error with context
   - Exit workflow immediately

3. **TUI Errors**:
   - Validation errors: Show in-place (red text)
   - Program errors: Exit with error message
   - User cancellation: Clean exit

## Extensibility Points

### Adding New Cloud Providers

1. Create `internal/generator/<provider>.go`
2. Implement provider-specific template
3. Add provider selection to Config
4. Update workflow to choose generator

### Adding New Configuration Options

1. Add field to `Config` struct
2. Add textinput in `tui.NewModel()`
3. Update `updateConfig()` method
4. Update templates to use new field

### Adding Pre/Post Deployment Hooks

1. Add hook functions to `workflow.Runner`
2. Call in `Run()` at appropriate stages
3. Add configuration for hook commands

### Adding Alternative TUI Modes

1. Create new model in `internal/tui/`
2. Add mode selector in `cmd.RunTUI()`
3. Share configuration between modes

## Testing Considerations

### Unit Testing
- Config load/save logic
- Template generation
- Input validation

### Integration Testing
- Full workflow execution (requires AWS)
- Tofu/Ansible command execution
- Error scenarios

### Manual Testing
- TUI navigation and input
- Different configuration combinations
- Deployment on real infrastructure

## Performance Considerations

- **TUI**: Minimal overhead, instant response
- **Tofu**: 30-120 seconds for EC2 provisioning
- **Ansible**: 5-15 minutes for RKE2 + Rancher deployment
- **Total**: ~6-17 minutes end-to-end

## Security Best Practices

1. **Credential Management**:
   - Use environment variables or AWS credential chain
   - Rotate access keys regularly
   - Use IAM roles when possible

2. **File Permissions**:
   - config.json: 0600
   - SSH private keys: 0600
   - Generated files: 0644 (no secrets)

3. **Network Security**:
   - Security groups should restrict access
   - Use VPN or bastion for production
   - Enable AWS CloudTrail for auditing

4. **Secret Management**:
   - Consider AWS Secrets Manager
   - Avoid committing config.json
   - Use .gitignore for sensitive files
