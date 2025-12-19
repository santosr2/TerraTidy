package main

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/santosr2/terratidy/internal/config"
	"github.com/santosr2/terratidy/internal/plugins"
	"github.com/spf13/cobra"
)

var pluginsCmd = &cobra.Command{
	Use:   "plugins",
	Short: "Plugin management commands",
	Long: `Manage TerraTidy plugins.

Plugins allow you to extend TerraTidy with custom rules, engines, and output formatters.
Plugins are Go shared libraries (.so files) that implement the TerraTidy plugin interfaces.

Plugin directories can be configured in .terratidy.yaml:

  plugins:
    enabled: true
    directories:
      - ~/.terratidy/plugins
      - ./plugins`,
}

var pluginsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed plugins",
	Run: func(_ *cobra.Command, _ []string) {
		cfg, err := config.Load(cfgFile)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		if !cfg.Plugins.Enabled {
			fmt.Println("Plugins are not enabled in configuration")
			return
		}

		manager := plugins.NewManager(cfg.Plugins.Directories)
		if err := manager.LoadAll(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error loading plugins: %v\n", err)
			os.Exit(1)
		}

		pluginList := manager.ListPlugins()
		if len(pluginList) == 0 {
			fmt.Println("No plugins installed")
			fmt.Printf("\nPlugin directories searched:\n")
			for _, dir := range cfg.Plugins.Directories {
				fmt.Printf("  - %s\n", dir)
			}
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "NAME\tVERSION\tTYPE\tDESCRIPTION")
		_, _ = fmt.Fprintln(w, "----\t-------\t----\t-----------")

		for _, p := range pluginList {
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
				p.Metadata.Name,
				p.Metadata.Version,
				p.Metadata.Type,
				p.Metadata.Description,
			)
		}
		_ = w.Flush()
	},
}

var pluginsInfoCmd = &cobra.Command{
	Use:   "info [plugin-name]",
	Short: "Show detailed information about a plugin",
	Args:  cobra.ExactArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		pluginName := args[0]

		cfg, err := config.Load(cfgFile)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		manager := plugins.NewManager(cfg.Plugins.Directories)
		if err := manager.LoadAll(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error loading plugins: %v\n", err)
			os.Exit(1)
		}

		var found *plugins.Plugin
		for _, p := range manager.ListPlugins() {
			if p.Metadata.Name == pluginName {
				found = p
				break
			}
		}

		if found == nil {
			_, _ = fmt.Fprintf(os.Stderr, "Plugin not found: %s\n", pluginName)
			os.Exit(1)
		}

		fmt.Printf("Name:        %s\n", found.Metadata.Name)
		fmt.Printf("Version:     %s\n", found.Metadata.Version)
		fmt.Printf("Type:        %s\n", found.Metadata.Type)
		fmt.Printf("Description: %s\n", found.Metadata.Description)
		fmt.Printf("Author:      %s\n", found.Metadata.Author)
		fmt.Printf("Path:        %s\n", found.Metadata.Path)

		// Show additional info based on type
		switch found.Metadata.Type {
		case plugins.PluginTypeRule:
			if rp, ok := found.Instance.(plugins.RulePlugin); ok {
				rules := rp.GetRules()
				fmt.Printf("\nProvides %d rule(s):\n", len(rules))
				for _, r := range rules {
					fmt.Printf("  - %s: %s\n", r.Name(), r.Description())
				}
			}
		case plugins.PluginTypeEngine:
			if ep, ok := found.Instance.(plugins.EnginePlugin); ok {
				fmt.Printf("\nEngine name: %s\n", ep.Name())
			}
		case plugins.PluginTypeFormatter:
			if fp, ok := found.Instance.(plugins.FormatterPlugin); ok {
				fmt.Printf("\nFormatter name: %s\n", fp.Name())
			}
		}
	},
}

var pluginsInitCmd = &cobra.Command{
	Use:   "init [name]",
	Short: "Initialize a new plugin project",
	Args:  cobra.ExactArgs(1),
	Run: func(_ *cobra.Command, args []string) {
		pluginName := args[0]

		// Create plugin directory
		dir := filepath.Join(".", pluginName)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error creating directory: %v\n", err)
			os.Exit(1)
		}

		// Create main.go
		mainContent := fmt.Sprintf(`// %s - TerraTidy Plugin
// Build with: go build -buildmode=plugin -o %s.so

package main

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/santosr2/terratidy/internal/plugins"
	"github.com/santosr2/terratidy/pkg/sdk"
)

// PluginMetadata provides information about this plugin
var PluginMetadata = &plugins.PluginMetadata{
	Name:        "%s",
	Version:     "1.0.0",
	Description: "Custom rules plugin",
	Author:      "Your Name",
	Type:        plugins.PluginTypeRule,
}

// Plugin implements the RulePlugin interface
type Plugin struct {
	rules []sdk.Rule
}

// New creates a new instance of the plugin
func New() plugins.RulePlugin {
	p := &Plugin{}
	p.rules = []sdk.Rule{
		&ExampleRule{},
	}
	return p
}

// GetRules returns all rules provided by this plugin
func (p *Plugin) GetRules() []sdk.Rule {
	return p.rules
}

// ExampleRule is an example custom rule
type ExampleRule struct{}

func (r *ExampleRule) Name() string {
	return "%s.example"
}

func (r *ExampleRule) Description() string {
	return "Example rule - replace with your implementation"
}

func (r *ExampleRule) Severity() sdk.Severity {
	return sdk.SeverityWarning
}

func (r *ExampleRule) Check(ctx *sdk.Context, file *hcl.File) ([]sdk.Finding, error) {
	// Implement your rule logic here
	return nil, nil
}

func (r *ExampleRule) Fix(ctx *sdk.Context, file *hcl.File) error {
	// Implement fix logic if auto-fixable
	return nil
}
`, pluginName, pluginName, pluginName, pluginName)

		mainPath := filepath.Join(dir, "main.go")
		if err := os.WriteFile(mainPath, []byte(mainContent), 0o644); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error writing main.go: %v\n", err)
			os.Exit(1)
		}

		// Create go.mod
		goModContent := fmt.Sprintf(`module %s

go 1.25

require github.com/santosr2/terratidy v0.1.0
`, pluginName)

		goModPath := filepath.Join(dir, "go.mod")
		if err := os.WriteFile(goModPath, []byte(goModContent), 0o644); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error writing go.mod: %v\n", err)
			os.Exit(1)
		}

		// Create Makefile
		makefileContent := fmt.Sprintf(`# %s Plugin Makefile

.PHONY: build install clean

build:
	go build -buildmode=plugin -o %s.so

install: build
	mkdir -p ~/.terratidy/plugins
	cp %s.so ~/.terratidy/plugins/

clean:
	rm -f %s.so
`, pluginName, pluginName, pluginName, pluginName)

		makefilePath := filepath.Join(dir, "Makefile")
		if err := os.WriteFile(makefilePath, []byte(makefileContent), 0o644); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error writing Makefile: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Plugin project created: %s/\n", dir)
		fmt.Println("\nNext steps:")
		fmt.Printf("  1. cd %s\n", pluginName)
		fmt.Println("  2. Edit main.go to implement your rules")
		fmt.Println("  3. make build")
		fmt.Println("  4. make install")
	},
}

func init() {
	rootCmd.AddCommand(pluginsCmd)
	pluginsCmd.AddCommand(pluginsListCmd)
	pluginsCmd.AddCommand(pluginsInfoCmd)
	pluginsCmd.AddCommand(pluginsInitCmd)
}
