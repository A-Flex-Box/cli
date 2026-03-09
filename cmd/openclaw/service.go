package openclaw

import (
	"context"
	"fmt"

	"github.com/A-Flex-Box/cli/internal/openclaw/service"
	"github.com/spf13/cobra"
)

func newServiceCmd(ctx *Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "service",
		Short:   "Manage OpenClaw service",
		Long:    `Start, stop, restart, and check status of the OpenClaw gateway service.`,
		Example: "cli openclaw service start\n  cli openclaw service status\n  cli openclaw service logs",
	}

	cmd.AddCommand(newServiceStartCmd(ctx))
	cmd.AddCommand(newServiceStopCmd(ctx))
	cmd.AddCommand(newServiceRestartCmd(ctx))
	cmd.AddCommand(newServiceStatusCmd(ctx))
	cmd.AddCommand(newServiceLogsCmd(ctx))

	return cmd
}

func newServiceStartCmd(ctx *Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "start",
		Short:   "Start the OpenClaw service",
		Example: "cli openclaw service start",
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr := getServiceManager(ctx)
			fmt.Println("Starting OpenClaw service...")

			if err := mgr.Start(context.Background()); err != nil {
				return fmt.Errorf("failed to start service: %w", err)
			}

			fmt.Println("OpenClaw service started.")
			return nil
		},
	}

	return cmd
}

func newServiceStopCmd(ctx *Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "stop",
		Short:   "Stop the OpenClaw service",
		Example: "cli openclaw service stop",
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr := getServiceManager(ctx)
			fmt.Println("Stopping OpenClaw service...")

			if err := mgr.Stop(context.Background()); err != nil {
				return fmt.Errorf("failed to stop service: %w", err)
			}

			fmt.Println("OpenClaw service stopped.")
			return nil
		},
	}

	return cmd
}

func newServiceRestartCmd(ctx *Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "restart",
		Short:   "Restart the OpenClaw service",
		Example: "cli openclaw service restart",
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr := getServiceManager(ctx)
			fmt.Println("Restarting OpenClaw service...")

			if err := mgr.Restart(context.Background()); err != nil {
				return fmt.Errorf("failed to restart service: %w", err)
			}

			fmt.Println("OpenClaw service restarted.")
			return nil
		},
	}

	return cmd
}

func newServiceStatusCmd(ctx *Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "status",
		Short:   "Check service status",
		Example: "cli openclaw service status",
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr := getServiceManager(ctx)
			info := mgr.Status()

			fmt.Println("OpenClaw Service Status")
			fmt.Println("=======================")
			fmt.Printf("Status: %s\n", info.Status)
			fmt.Printf("Port: %d\n", info.Port)
			if info.PID > 0 {
				fmt.Printf("PID: %d\n", info.PID)
			}
			if info.Version != "" {
				fmt.Printf("Version: %s\n", info.Version)
			}
			if info.Uptime != "" {
				fmt.Printf("Uptime: %s\n", info.Uptime)
			}

			return nil
		},
	}

	return cmd
}

func newServiceLogsCmd(ctx *Context) *cobra.Command {
	var lines int

	cmd := &cobra.Command{
		Use:     "logs",
		Short:   "View service logs",
		Example: "cli openclaw service logs\n  cli openclaw service logs --lines 100",
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr := getServiceManager(ctx)
			logs, err := mgr.GetLogs(lines)
			if err != nil {
				return fmt.Errorf("failed to get logs: %w", err)
			}

			fmt.Println("OpenClaw Service Logs")
			fmt.Println("=====================")
			for _, line := range logs {
				fmt.Println(line)
			}

			return nil
		},
	}

	cmd.Flags().IntVarP(&lines, "lines", "n", 50, "Number of lines to show")

	return cmd
}

func getServiceManager(ctx *Context) *service.Manager {
	return service.NewManager(ctx.SysInfo.GetServiceType())
}
