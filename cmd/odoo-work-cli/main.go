package main

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/BurntSushi/toml"
	"github.com/seletz/odoo-work-cli/cmd/clock"
	"github.com/seletz/odoo-work-cli/cmd/entries"
	"github.com/seletz/odoo-work-cli/internal/config"
	"github.com/seletz/odoo-work-cli/internal/odoo"
	"github.com/seletz/odoo-work-cli/internal/parsing"
	"github.com/seletz/odoo-work-cli/internal/tui"
	"github.com/seletz/odoo-work-cli/internal/version"
	"github.com/spf13/cobra"
)

var cfgFile string

func main() {

	cfg, err := loadConfig()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	client, err := newClient(cfg)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	rootCmd := &cobra.Command{
		Use:     "odoo-work-cli",
		Short:   "CLI for managing Odoo 17 timesheets",
		Version: version.Version,

		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			if client != nil {
				client.Close()
			}
		},
	}

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file path (skip discovery)")

	rootCmd.AddCommand(projectsCmd)
	rootCmd.AddCommand(tasksCmd)
	rootCmd.AddCommand(timesheetsCmd)
	rootCmd.AddCommand(fieldsCmd)
	rootCmd.AddCommand(whoamiCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(tuiCmd)
	rootCmd.AddCommand(entries.CMD(client))
	rootCmd.AddCommand(clock.Cmd(client))

	if err = rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {

	timesheetsCmd.Flags().StringVar(&tsWeek, "week", "", "ISO week (e.g. 2026-W10), defaults to current week")

	tuiCmd.Flags().StringVar(&tuiWeek, "week", "", "ISO week (e.g. 2026-W10), defaults to current week")
	configCmd.Flags().BoolVar(&configMerged, "merged", false, "print merged TOML config (password redacted)")
	configCmd.AddCommand(configInstallCmd)
}

// loadConfig loads and merges config using file discovery and env vars.
func loadConfig() (*config.Config, error) {
	result, err := config.Discover(cfgFile)
	if err != nil {
		return nil, err
	}
	if err := result.Config.Validate(); err != nil {
		return nil, err
	}
	return result.Config, nil
}

// newClient creates a new Odoo client from the merged config.
func newClient(cfg *config.Config) (*odoo.XMLRPCClient, error) {
	return odoo.NewXMLRPCClient(cfg.URL, cfg.Database, cfg.Username, cfg.Password, cfg.WebPassword, cfg.TOTPSecret, cfg.Models)
}

var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "List Odoo projects",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		client, err := newClient(cfg)
		if err != nil {
			return err
		}
		defer client.Close()
		projects, err := client.ListProjects()
		if err != nil {
			return err
		}

		// Build dynamic column headers from config extra fields.
		var extraNames []string
		if mc, ok := cfg.Models["project"]; ok {
			for _, ef := range mc.ExtraFields {
				extraNames = append(extraNames, ef.Name)
			}
		}

		// Print header.
		header := fmt.Sprintf("%-6s %-30s %-20s %-15s %-15s %-20s",
			"ID", "Name", "Customer", "Company", "Phase", "Project Manager")
		sep := fmt.Sprintf("%-6s %-30s %-20s %-15s %-15s %-20s",
			"------", "------------------------------", "--------------------", "---------------", "---------------", "--------------------")
		for _, name := range extraNames {
			label := strings.ReplaceAll(name, "_", " ")
			header += fmt.Sprintf(" %-20s", label)
			sep += fmt.Sprintf(" %-20s", "--------------------")
		}
		header += fmt.Sprintf(" %s", "Active")
		sep += fmt.Sprintf(" %s", "------")
		fmt.Println(header)
		fmt.Println(sep)

		for _, p := range projects {
			line := fmt.Sprintf("%-6d %-30s %-20s %-15s %-15s %-20s",
				p.ID, p.Name, p.Customer, p.Company, p.Stage, p.ProjectManager)
			for _, name := range extraNames {
				line += fmt.Sprintf(" %-20s", p.ExtraFields[name])
			}
			line += fmt.Sprintf(" %v", p.Active)
			fmt.Println(line)
		}
		return nil
	},
}

var tasksCmd = &cobra.Command{
	Use:   "tasks [project-id]",
	Short: "List Odoo tasks",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		client, err := newClient(cfg)
		if err != nil {
			return err
		}
		defer client.Close()

		var projectID int64
		if len(args) == 1 {
			projectID, err = strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid project ID: %w", err)
			}
		}

		tasks, err := client.ListTasks(projectID)
		if err != nil {
			return err
		}
		fmt.Printf("%-8s %-40s %-30s %s\n", "ID", "Name", "Project", "Stage")
		fmt.Printf("%-8s %-40s %-30s %s\n", "--------", "----------------------------------------", "------------------------------", "--------------------")
		for _, t := range tasks {
			fmt.Printf("%-8d %-40s %-30s %s\n", t.ID, t.Name, t.Project, t.Stage)
		}
		return nil
	},
}

var tsWeek string

var timesheetsCmd = &cobra.Command{
	Use:   "timesheets",
	Short: "List Odoo timesheets for a week",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		client, err := newClient(cfg)
		if err != nil {
			return err
		}
		defer client.Close()

		dateFrom, dateTo, err := parsing.WeekDateRange(tsWeek)
		if err != nil {
			return err
		}

		entries, err := client.ListTimesheets(dateFrom, dateTo)
		if err != nil {
			return err
		}
		fmt.Printf("Week: %s to %s\n\n", dateFrom, dateTo)
		fmt.Printf("%-8s %-12s %-8s %-25s %-8s %-25s %-30s %s\n", "ID", "Date", "ProjID", "Project", "TaskID", "Task", "Description", "Hours")
		fmt.Printf("%-8s %-12s %-8s %-25s %-8s %-25s %-30s %s\n",
			"--------", "------------", "--------", "-------------------------", "--------", "-------------------------", "------------------------------", "-----")
		var total float64
		for _, e := range entries {
			fmt.Printf("%-8d %-12s %-8d %-25s %-8d %-25s %-30s %.2f\n", e.ID, e.Date, e.ProjectID, e.Project, e.TaskID, e.Task, e.Name, e.Hours)
			total += e.Hours
		}
		fmt.Printf("\nTotal: %.2f hours\n", total)
		return nil
	},
}

var fieldsCmd = &cobra.Command{
	Use:   "fields <model>",
	Short: "Inspect Odoo model fields",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		client, err := newClient(cfg)
		if err != nil {
			return err
		}
		defer client.Close()

		fields, err := client.GetFields(args[0])
		if err != nil {
			return err
		}
		fmt.Printf("%-30s %-15s %-30s %s\n", "Field", "Type", "Label", "Required")
		fmt.Printf("%-30s %-15s %-30s %s\n", "------------------------------", "---------------", "------------------------------", "--------")
		for _, f := range fields {
			fmt.Printf("%-30s %-15s %-30s %v\n", f.Name, f.Type, f.String, f.Required)
		}
		return nil
	},
}

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show current Odoo user info",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		client, err := newClient(cfg)
		if err != nil {
			return err
		}
		defer client.Close()
		info, err := client.WhoAmI()
		if err != nil {
			return err
		}
		fmt.Printf("ID:      %d\n", info.ID)
		fmt.Printf("Name:    %s\n", info.Name)
		fmt.Printf("Login:   %s\n", info.Login)
		fmt.Printf("Email:   %s\n", info.Email)
		fmt.Printf("Company: %s\n", info.Company)
		return nil
	},
}

var tuiWeek string

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Interactive weekly timesheet view",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		client, err := newClient(cfg)
		if err != nil {
			return err
		}
		defer client.Close()

		monday, err := tui.ParseWeekMonday(tuiWeek)
		if err != nil {
			return err
		}

		m := tui.NewModel(client, tui.MondayTime{Time: monday}, cfg.Hours, cfg.Bundesland, cfg.Keys, cfg.CompanyColors)
		p := tea.NewProgram(m)
		_, err = p.Run()
		return err
	},
}

var configMerged bool

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show discovered config files and merged configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		result, err := config.Discover(cfgFile)
		if err != nil {
			return err
		}

		if !configMerged {
			if len(result.Files) == 0 {
				fmt.Println("No config files discovered.")
			} else {
				fmt.Println("Discovered config files (merge order):")
				for _, f := range result.Files {
					fmt.Printf("  %s\n", f)
				}
			}
			return nil
		}

		// Password is tagged toml:"-" on Config, so the encoder
		// omits it automatically — output is a valid config file.
		var buf bytes.Buffer
		if err := toml.NewEncoder(&buf).Encode(result.Config); err != nil {
			return fmt.Errorf("encoding config: %w", err)
		}
		fmt.Print(buf.String())
		return nil
	},
}

var configInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Create a default config file in the platform config directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := config.DefaultConfigPath()
		if err != nil {
			return err
		}

		fmt.Printf("This will create a default config file at:\n  %s\n\n", path)

		if !confirmPrompt("Continue?") {
			fmt.Println("Aborted.")
			return nil
		}

		if err := config.InstallConfig(path); err != nil {
			return err
		}

		fmt.Printf("Config file created at: %s\n", path)
		fmt.Println("Edit it to set your Odoo URL, database, and username.")
		fmt.Println("Set ODOO_PASSWORD via environment variable (see .env.1p).")
		return nil
	},
}

// confirmPrompt prints msg and reads y/N from stdin.
func confirmPrompt(msg string) bool {
	fmt.Printf("%s [y/N] ", msg)
	var response string
	if _, err := fmt.Scanln(&response); err != nil {
		return false
	}
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}
