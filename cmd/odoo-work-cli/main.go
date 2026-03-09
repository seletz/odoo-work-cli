package main

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/BurntSushi/toml"
	"github.com/seletz/odoo-work-cli/internal/config"
	"github.com/seletz/odoo-work-cli/internal/odoo"
	"github.com/seletz/odoo-work-cli/internal/tui"
	"github.com/spf13/cobra"
)

var cfgFile string

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "odoo-work-cli",
	Short: "CLI for managing Odoo 17 timesheets",
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file path (skip discovery)")

	rootCmd.AddCommand(projectsCmd)
	rootCmd.AddCommand(tasksCmd)
	rootCmd.AddCommand(timesheetsCmd)
	rootCmd.AddCommand(fieldsCmd)
	rootCmd.AddCommand(whoamiCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(tuiCmd)
	rootCmd.AddCommand(entriesCmd)

	timesheetsCmd.Flags().StringVar(&tsWeek, "week", "", "ISO week (e.g. 2026-W10), defaults to current week")
	entriesCmd.Flags().StringVar(&entriesWeek, "week", "", "ISO week (e.g. 2026-W10), defaults to current week")
	entriesCmd.Flags().StringVar(&entriesDate, "date", "", "specific date (YYYY-MM-DD), overrides --week")
	entriesCmd.Flags().StringVar(&entriesProject, "project", "", "filter by project name (substring, case-insensitive)")
	entriesCmd.Flags().StringVar(&entriesTask, "task", "", "filter by task name (substring, case-insensitive)")
	entriesCmd.Flags().StringVar(&entriesStatus, "status", "", "filter by validation status (e.g. draft, validated)")
	tuiCmd.Flags().StringVar(&tuiWeek, "week", "", "ISO week (e.g. 2026-W10), defaults to current week")
	configCmd.Flags().BoolVar(&configMerged, "merged", false, "print merged TOML config (password redacted)")

	entriesCmd.AddCommand(entriesAddCmd)
	entriesAddCmd.Flags().Int64Var(&addProjectID, "project-id", 0, "Odoo project ID (required)")
	entriesAddCmd.Flags().Int64Var(&addTaskID, "task-id", 0, "Odoo task ID (optional)")
	entriesAddCmd.Flags().StringVar(&addDate, "date", "", "entry date YYYY-MM-DD (defaults to today)")
	entriesAddCmd.Flags().Float64Var(&addHours, "hours", 0, "hours worked (required, > 0)")
	entriesAddCmd.Flags().StringVar(&addDescription, "description", "", "work description (required)")
	_ = entriesAddCmd.MarkFlagRequired("hours")
	_ = entriesAddCmd.MarkFlagRequired("description")
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
	return odoo.NewXMLRPCClient(cfg.URL, cfg.Database, cfg.Username, cfg.Password, cfg.Models)
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

// weekDateRange returns the Monday and Sunday of the ISO week specified
// as "2006-W02" format, or the current week if empty.
func weekDateRange(week string) (string, string, error) {
	monday, err := tui.ParseWeekMonday(week)
	if err != nil {
		return "", "", err
	}
	sunday := monday.AddDate(0, 0, 6)
	return monday.Format("2006-01-02"), sunday.Format("2006-01-02"), nil
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

		dateFrom, dateTo, err := weekDateRange(tsWeek)
		if err != nil {
			return err
		}

		entries, err := client.ListTimesheets(dateFrom, dateTo)
		if err != nil {
			return err
		}
		fmt.Printf("Week: %s to %s\n\n", dateFrom, dateTo)
		fmt.Printf("%-8s %-12s %-25s %-25s %-30s %s\n", "ID", "Date", "Project", "Task", "Description", "Hours")
		fmt.Printf("%-8s %-12s %-25s %-25s %-30s %s\n",
			"--------", "------------", "-------------------------", "-------------------------", "------------------------------", "-----")
		var total float64
		for _, e := range entries {
			fmt.Printf("%-8d %-12s %-25s %-25s %-30s %.2f\n", e.ID, e.Date, e.Project, e.Task, e.Name, e.Hours)
			total += e.Hours
		}
		fmt.Printf("\nTotal: %.2f hours\n", total)
		return nil
	},
}

// parseDateRange returns a single-day date range for the given YYYY-MM-DD string.
func parseDateRange(date string) (string, string, error) {
	d, err := time.Parse("2006-01-02", date)
	if err != nil {
		return "", "", fmt.Errorf("invalid date %q: expected YYYY-MM-DD", date)
	}
	s := d.Format("2006-01-02")
	return s, s, nil
}

// filterEntries returns entries matching the project, task, and status filters.
// Project and task use case-insensitive substring match. Status uses exact match.
// Empty filter matches all.
func filterEntries(entries []odoo.TimesheetEntry, project, task, status string) []odoo.TimesheetEntry {
	if project == "" && task == "" && status == "" {
		return entries
	}
	projectLower := strings.ToLower(project)
	taskLower := strings.ToLower(task)
	var result []odoo.TimesheetEntry
	for _, e := range entries {
		if project != "" && !strings.Contains(strings.ToLower(e.Project), projectLower) {
			continue
		}
		if task != "" && !strings.Contains(strings.ToLower(e.Task), taskLower) {
			continue
		}
		if status != "" && e.ValidatedStatus != status {
			continue
		}
		result = append(result, e)
	}
	return result
}

var (
	entriesWeek    string
	entriesDate    string
	entriesProject string
	entriesTask    string
	entriesStatus  string
)

var entriesCmd = &cobra.Command{
	Use:   "entries",
	Short: "List individual timesheet entries with full detail",
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

		var dateFrom, dateTo string
		if entriesDate != "" {
			dateFrom, dateTo, err = parseDateRange(entriesDate)
		} else {
			dateFrom, dateTo, err = weekDateRange(entriesWeek)
		}
		if err != nil {
			return err
		}

		entries, err := client.ListTimesheets(dateFrom, dateTo)
		if err != nil {
			return err
		}

		entries = filterEntries(entries, entriesProject, entriesTask, entriesStatus)

		if entriesDate != "" {
			fmt.Printf("Date: %s\n\n", dateFrom)
		} else {
			fmt.Printf("Week: %s to %s\n\n", dateFrom, dateTo)
		}

		fmt.Printf("%-8s %-12s %-25s %-25s %-6s %-10s %s\n",
			"ID", "Date", "Project", "Task", "Hours", "Status", "Description")
		fmt.Printf("%-8s %-12s %-25s %-25s %-6s %-10s %s\n",
			"--------", "------------", "-------------------------", "-------------------------", "------", "----------", "------------------------------")

		var total float64
		for _, e := range entries {
			fmt.Printf("%-8d %-12s %-25s %-25s %-6s %-10s %s\n",
				e.ID, e.Date, e.Project, e.Task, tui.FormatHours(e.Hours), e.ValidatedStatus, e.Name)
			total += e.Hours
		}

		fmt.Printf("\nTotal: %s (%d entries)\n", tui.FormatHours(total), len(entries))
		return nil
	},
}

// buildTimesheetWriteParams constructs and validates TimesheetWriteParams from CLI flag values.
// An empty date defaults to today.
func buildTimesheetWriteParams(projectID, taskID int64, date, description string, hours float64) (odoo.TimesheetWriteParams, error) {
	if date == "" {
		date = time.Now().Format("2006-01-02")
	} else {
		if _, err := time.Parse("2006-01-02", date); err != nil {
			return odoo.TimesheetWriteParams{}, fmt.Errorf("invalid date %q: expected YYYY-MM-DD", date)
		}
	}
	p := odoo.TimesheetWriteParams{
		ProjectID: projectID,
		TaskID:    taskID,
		Date:      date,
		Name:      description,
		Hours:     hours,
	}
	if err := odoo.ValidateTimesheetParams(p); err != nil {
		return odoo.TimesheetWriteParams{}, err
	}
	return p, nil
}

var (
	addProjectID   int64
	addTaskID      int64
	addDate        string
	addHours       float64
	addDescription string
)

var entriesAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Create a new timesheet entry",
	RunE: func(cmd *cobra.Command, args []string) error {
		params, err := buildTimesheetWriteParams(addProjectID, addTaskID, addDate, addDescription, addHours)
		if err != nil {
			return err
		}

		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		client, err := newClient(cfg)
		if err != nil {
			return err
		}
		defer client.Close()

		id, err := client.CreateTimesheet(params)
		if err != nil {
			return err
		}

		fmt.Printf("Created entry %d\n", id)
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

		m := tui.NewModel(client, tui.MondayTime{Time: monday}, cfg.Hours, cfg.Bundesland)
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
