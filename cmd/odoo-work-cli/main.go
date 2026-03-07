package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/seletz/odoo-work-cli/internal/config"
	"github.com/seletz/odoo-work-cli/internal/odoo"
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
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default $HOME/.config/odoo-work-cli/config.toml)")

	rootCmd.AddCommand(projectsCmd)
	rootCmd.AddCommand(tasksCmd)
	rootCmd.AddCommand(timesheetsCmd)
	rootCmd.AddCommand(fieldsCmd)
	rootCmd.AddCommand(whoamiCmd)

	timesheetsCmd.Flags().StringVar(&tsWeek, "week", "", "ISO week (e.g. 2026-W10), defaults to current week")
}

var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "List Odoo projects",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadFromEnv()
		if err != nil {
			return err
		}
		client, err := odoo.NewXMLRPCClient(cfg.URL, cfg.Database, cfg.Username, cfg.Password)
		if err != nil {
			return err
		}
		defer client.Close()
		projects, err := client.ListProjects()
		if err != nil {
			return err
		}
		fmt.Printf("%-8s %-40s %s\n", "ID", "Name", "Active")
		fmt.Printf("%-8s %-40s %s\n", "--------", "----------------------------------------", "------")
		for _, p := range projects {
			fmt.Printf("%-8d %-40s %v\n", p.ID, p.Name, p.Active)
		}
		return nil
	},
}

var tasksCmd = &cobra.Command{
	Use:   "tasks [project-id]",
	Short: "List Odoo tasks",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadFromEnv()
		if err != nil {
			return err
		}
		client, err := odoo.NewXMLRPCClient(cfg.URL, cfg.Database, cfg.Username, cfg.Password)
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
	var year, isoWeek int
	if week == "" {
		now := time.Now()
		year, isoWeek = now.ISOWeek()
	} else {
		_, err := fmt.Sscanf(week, "%d-W%d", &year, &isoWeek)
		if err != nil {
			return "", "", fmt.Errorf("invalid week format %q (expected YYYY-Www): %w", week, err)
		}
	}
	// Find Monday of ISO week 1 for the given year.
	jan4 := time.Date(year, 1, 4, 0, 0, 0, 0, time.Local)
	weekday := jan4.Weekday()
	if weekday == 0 {
		weekday = 7
	}
	monday1 := jan4.AddDate(0, 0, -int(weekday-1))
	monday := monday1.AddDate(0, 0, (isoWeek-1)*7)
	sunday := monday.AddDate(0, 0, 6)
	return monday.Format("2006-01-02"), sunday.Format("2006-01-02"), nil
}

var tsWeek string

var timesheetsCmd = &cobra.Command{
	Use:   "timesheets",
	Short: "List Odoo timesheets for a week",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadFromEnv()
		if err != nil {
			return err
		}
		client, err := odoo.NewXMLRPCClient(cfg.URL, cfg.Database, cfg.Username, cfg.Password)
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

var fieldsCmd = &cobra.Command{
	Use:   "fields <model>",
	Short: "Inspect Odoo model fields",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadFromEnv()
		if err != nil {
			return err
		}
		client, err := odoo.NewXMLRPCClient(cfg.URL, cfg.Database, cfg.Username, cfg.Password)
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
		cfg, err := config.LoadFromEnv()
		if err != nil {
			return err
		}
		client, err := odoo.NewXMLRPCClient(cfg.URL, cfg.Database, cfg.Username, cfg.Password)
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
