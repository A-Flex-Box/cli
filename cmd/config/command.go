package config

import (
	"fmt"
	"os"

	"github.com/A-Flex-Box/cli/internal/config"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/spf13/cobra"
)

var (
	highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	special   = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}
	muted     = lipgloss.AdaptiveColor{Light: "#6B6B6B", Dark: "#9B9B9B"}
)

// NewCmd returns the config command with all subcommands. Receives injected config.
func NewCmd(cfg *config.Root, mgr *config.Manager) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "config",
		Short:   "Manage relay configuration",
		Example: "cli config list",
	}
	cmd.AddCommand(newListCmd(cfg), newUseCmd(cfg, mgr), newAddCmd(cfg, mgr), newRmCmd(cfg, mgr))
	return cmd
}

func newListCmd(cfg *config.Root) *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Short:   "Show table of relays (highlight active)",
		Example: "cli config list",
		Run: func(cmd *cobra.Command, args []string) {
			active := cfg.Wormhole.ActiveRelay
			relays := cfg.Wormhole.Relays

			rows := make([][]string, 0, len(relays))
			for name, addr := range relays {
				mark := ""
				if name == active {
					mark = " *"
				}
				rows = append(rows, []string{name + mark, addr})
			}

			t := table.New().
				Border(lipgloss.RoundedBorder()).
				BorderStyle(lipgloss.NewStyle().Foreground(highlight)).
				Headers("Name", "Address").
				Rows(rows...).
				Width(60)

			headerStyle := lipgloss.NewStyle().Foreground(highlight).Bold(true).Padding(0, 1)
			t = t.StyleFunc(func(row, col int) lipgloss.Style {
				if row == table.HeaderRow {
					return headerStyle
				}
				s := lipgloss.NewStyle().Padding(0, 1)
				if col == 0 {
					s = s.Foreground(special)
				} else {
					s = s.Foreground(muted)
				}
				return s
			})

			fmt.Println(t.Render())
			fmt.Println(lipgloss.NewStyle().Foreground(muted).Render(" * = active"))
		},
	}
}

func newUseCmd(cfg *config.Root, mgr *config.Manager) *cobra.Command {
	return &cobra.Command{
		Use:     "use [name]",
		Short:   "Switch active relay",
		Example: "cli config use public",
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]
			if cfg.Wormhole.Relays[name] == "" {
				fmt.Printf("Relay '%s' not found\n", name)
				os.Exit(1)
			}
			cfg.Wormhole.ActiveRelay = name
			if err := mgr.Save(cfg); err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Active relay set to '%s'\n", name)
		},
	}
}

func newAddCmd(cfg *config.Root, mgr *config.Manager) *cobra.Command {
	return &cobra.Command{
		Use:   "add [name] [addr]",
		Short: "Add relay alias",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			name, addr := args[0], args[1]
			if cfg.Wormhole.Relays == nil {
				cfg.Wormhole.Relays = make(map[string]string)
			}
			cfg.Wormhole.Relays[name] = addr
			if err := mgr.Save(cfg); err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Added relay '%s' -> %s\n", name, addr)
		},
	}
}

func newRmCmd(cfg *config.Root, mgr *config.Manager) *cobra.Command {
	return &cobra.Command{
		Use:     "rm [name]",
		Short:   "Remove relay alias",
		Example: "cli config rm myrelay",
		Args:    cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]
			if cfg.Wormhole.Relays == nil {
				cfg.Wormhole.Relays = make(map[string]string)
			}
			delete(cfg.Wormhole.Relays, name)
			if err := mgr.Save(cfg); err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Removed relay '%s'\n", name)
		},
	}
}
