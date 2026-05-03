package terraform

import (
	"fmt"
	"strings"
)

// Command defines the interface for all Terraform commands
type Command interface {
	Run(args []string) error
	Help() string
	Validate(payload map[string]any) (map[string]any, []ValidationError)
	Name() string
}

// SubCommand defines an optional sub-command interface
type SubCommand interface {
	Command
	SubCommands() []string
	RunSub(sub string, args []string) error
}

// ValidationError represents a single validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Field, e.Message)
}

// CommandResult is returned after validation
type CommandResult struct {
	Command    string            `json:"command"`
	SubCommand string            `json:"sub_command,omitempty"`
	Args       []string          `json:"args,omitempty"`
	Payload    map[string]any    `json:"payload"`
	HelpText   string            `json:"help,omitempty"`
	Valid      bool              `json:"valid"`
	Errors     []ValidationError `json:"errors,omitempty"`
}

// BaseCommand provides shared helpers for all commands
type BaseCommand struct {
	name string
}

func (b *BaseCommand) Name() string { return b.name }

func (b *BaseCommand) Run(args []string) error {
	fmt.Printf("terraform %s %s\n", b.name, strings.Join(args, " "))
	return nil
}

// getString safely extracts a string field from a payload
func getString(payload map[string]any, key string) (string, bool) {
	v, ok := payload[key]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

// getBool safely extracts a bool field from a payload
func getBool(payload map[string]any, key string) (bool, bool) {
	v, ok := payload[key]
	if !ok {
		return false, false
	}
	b, ok := v.(bool)
	return b, ok
}

// getStringSlice safely extracts a []string field from a payload
func getStringSlice(payload map[string]any, key string) ([]string, bool) {
	v, ok := payload[key]
	if !ok {
		return nil, false
	}
	switch val := v.(type) {
	case []string:
		return val, true
	case []any:
		result := make([]string, 0, len(val))
		for _, item := range val {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result, true
	}
	return nil, false
}
