package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/diptopandit/tork/internal/application"
	"github.com/diptopandit/tork/internal/domain"
	"github.com/diptopandit/tork/internal/infrastructure/config"
)

// NewRootCmd builds the Cobra root command wired to the given services.
func NewRootCmd(taskSvc *application.TaskService, listSvc *application.ListService, cfg *config.Config) *cobra.Command {
	root := &cobra.Command{
		Use:   "tork",
		Short: "Terminal task manager",
		Long:  "tork – a fast, offline-first terminal task manager.",
	}

	root.AddCommand(
		newAddCmd(taskSvc, listSvc, cfg),
		newListCmd(taskSvc, cfg),
		newShowCmd(taskSvc, cfg),
		newEditCmd(taskSvc, cfg),
		newStatusCmd(taskSvc, cfg),
		newDoneCmd(taskSvc, cfg),
		newDeleteCmd(taskSvc),
		newSearchCmd(taskSvc),
		newUpdateCmd(taskSvc),
		newListsCmd(listSvc),
		newListCreateCmd(listSvc),
		newListRenameCmd(listSvc),
		newListDeleteCmd(listSvc),
	)
	return root
}

// ── add ──────────────────────────────────────────────────────────────────────

func newAddCmd(taskSvc *application.TaskService, listSvc *application.ListService, cfg *config.Config) *cobra.Command {
	var (
		listName    string
		priority    string
		dueDate     string
		tags        string
		description string
	)

	cmd := &cobra.Command{
		Use:   "add <title>",
		Short: "Add a new task",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			title := strings.Join(args, " ")

			// Resolve or create list.
			var listID string
			if listName == "" {
				lists, err := listSvc.GetAllLists()
				if err != nil {
					return err
				}
				if len(lists) == 0 {
					l, err := listSvc.CreateList("Default", nil)
					if err != nil {
						return err
					}
					listID = l.ID
				} else {
					listID = lists[0].ID
				}
			} else {
				lists, err := listSvc.GetAllLists()
				if err != nil {
					return err
				}
				for _, l := range lists {
					if strings.EqualFold(l.Name, listName) {
						listID = l.ID
						break
					}
				}
				if listID == "" {
					l, err := listSvc.CreateList(listName, nil)
					if err != nil {
						return err
					}
					listID = l.ID
				}
			}

			pri, err := ParsePriority(cfg.Priorities, priority)
			if err != nil {
				return err
			}
			due, err := ParseDate(dueDate)
			if err != nil {
				return err
			}

			var tagList []string
			if tags != "" {
				for _, t := range strings.Split(tags, ",") {
					if s := strings.TrimSpace(t); s != "" {
						tagList = append(tagList, s)
					}
				}
			}

			task, err := taskSvc.CreateTask(application.CreateTaskInput{
				ListID:      listID,
				Title:       title,
				Description: description,
				Status:      domain.Status(config.DefaultStatus(cfg.Statuses)),
				Priority:    pri,
				DueDate:     due,
				Tags:        tagList,
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stdout, "Created task #%d: %s\n", task.NumID, task.Title)
			return nil
		},
	}

	priNames := make([]string, len(cfg.Priorities))
	for i, p := range cfg.Priorities {
		priNames[i] = p.Name
	}
	cmd.Flags().StringVarP(&listName, "list", "l", "", "list name")
	cmd.Flags().StringVarP(&priority, "priority", "p", "", "priority ("+strings.Join(priNames, "/")+")")
	cmd.Flags().StringVarP(&dueDate, "due", "d", "", "due date DD-MM-YYYY or YYYY-MM-DD")
	cmd.Flags().StringVarP(&tags, "tags", "t", "", "comma-separated tags")
	cmd.Flags().StringVarP(&description, "description", "D", "", "task description")
	return cmd
}

// ── list ─────────────────────────────────────────────────────────────────────

func newListCmd(taskSvc *application.TaskService, cfg *config.Config) *cobra.Command {
	var (
		statusFlag   string
		priorityFlag string
		listID       string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			b := application.NewFilterBuilder()
			if listID != "" {
				b.WithLists(listID)
			}
			if statusFlag != "" {
				s, err := ParseStatus(cfg.Statuses, statusFlag)
				if err != nil {
					return err
				}
				b.WithStatuses(s)
			}
			if priorityFlag != "" {
				p, err := ParsePriority(cfg.Priorities, priorityFlag)
				if err != nil {
					return err
				}
				b.WithPriorities(p)
			}

			tasks, err := taskSvc.ListTasks(b.Build())
			if err != nil {
				return err
			}
			if len(tasks) == 0 {
				fmt.Println("No tasks found.")
				return nil
			}
			for _, t := range tasks {
				due := ""
				if t.DueDate != nil {
					due = " due:" + t.DueDate.Format("2006-01-02")
				}
				fmt.Printf("#%-4d %-8s %-8s %s%s\n",
					t.NumID, string(t.Status), config.PriorityLabel(cfg.Priorities, int(t.Priority)), t.Title, due)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&statusFlag, "status", "s", "", "filter by status")
	cmd.Flags().StringVarP(&priorityFlag, "priority", "p", "", "filter by priority")
	cmd.Flags().StringVarP(&listID, "list", "l", "", "filter by list ID")
	return cmd
}

// ── done ─────────────────────────────────────────────────────────────────────

func newDoneCmd(taskSvc *application.TaskService, cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "done <id>",
		Short: "Mark a task as done",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := resolveTaskID(taskSvc, args[0])
			if err != nil {
				return err
			}
			// Use third configured status as "done", or fallback.
			var st domain.Status
			if len(cfg.Statuses) >= 3 {
				st = domain.Status(cfg.Statuses[2].Name)
			} else {
				st = domain.StatusDone
			}
			_, err = taskSvc.UpdateTask(application.UpdateTaskInput{
				ID:     id,
				Status: &st,
			})
			if err != nil {
				return err
			}
			fmt.Println("Marked done.")
			return nil
		},
	}
}

// ── delete ───────────────────────────────────────────────────────────────────

func newDeleteCmd(taskSvc *application.TaskService) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := resolveTaskID(taskSvc, args[0])
			if err != nil {
				return err
			}
			if err := taskSvc.DeleteTask(id); err != nil {
				return err
			}
			fmt.Println("Deleted.")
			return nil
		},
	}
}

// ── search ───────────────────────────────────────────────────────────────────

func newSearchCmd(taskSvc *application.TaskService) *cobra.Command {
	return &cobra.Command{
		Use:   "search <query>",
		Short: "Full-text search tasks",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.Join(args, " ")
			tasks, err := taskSvc.ListTasks(application.TaskFilter{Search: query})
			if err != nil {
				return err
			}
			if len(tasks) == 0 {
				fmt.Println("No results.")
				return nil
			}
			for _, t := range tasks {
				fmt.Printf("#%-4d %s\n", t.NumID, t.Title)
			}
			return nil
		},
	}
}

// ── show ─────────────────────────────────────────────────────────────────────

func newShowCmd(taskSvc *application.TaskService, cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Show full task details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := resolveTaskID(taskSvc, args[0])
			if err != nil {
				return err
			}
			t, err := taskSvc.GetTask(id)
			if err != nil {
				return err
			}
			fmt.Printf("ID:          %s\n", t.ID)
			fmt.Printf("NumID:       #%d\n", t.NumID)
			fmt.Printf("Title:       %s\n", t.Title)
			fmt.Printf("Status:      %s\n", t.Status)
			fmt.Printf("Priority:    %s\n", config.PriorityLabel(cfg.Priorities, int(t.Priority)))
			if t.Description != "" {
				fmt.Printf("Description: %s\n", t.Description)
			}
			if t.DueDate != nil {
				fmt.Printf("Due:         %s\n", t.DueDate.Format("02-01-2006"))
			}
			if len(t.Tags) > 0 {
				fmt.Printf("Tags:        %s\n", strings.Join(t.Tags, ", "))
			}
			fmt.Printf("Created:     %s\n", t.CreatedAt.Format("02-01-2006 15:04"))
			fmt.Printf("Updated:     %s\n", t.UpdatedAt.Format("02-01-2006 15:04"))

			updates, err := taskSvc.GetUpdates(t.ID)
			if err == nil && len(updates) > 0 {
				fmt.Printf("\nUpdates (%d):\n", len(updates))
				for _, u := range updates {
					fmt.Printf("  [%s] %s\n", u.CreatedAt.Format("02-01-2006 15:04"), u.Body)
				}
			}
			return nil
		},
	}
}

// ── edit ─────────────────────────────────────────────────────────────────────

func newEditCmd(taskSvc *application.TaskService, cfg *config.Config) *cobra.Command {
	var (
		title       string
		description string
		priority    string
		dueDate     string
		status      string
		tags        string
	)

	cmd := &cobra.Command{
		Use:   "edit <id>",
		Short: "Edit an existing task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := resolveTaskID(taskSvc, args[0])
			if err != nil {
				return err
			}
			in := application.UpdateTaskInput{ID: id}

			if cmd.Flags().Changed("title") {
				in.Title = &title
			}
			if cmd.Flags().Changed("description") {
				in.Description = &description
			}
			if cmd.Flags().Changed("status") {
				s, err := ParseStatus(cfg.Statuses, status)
				if err != nil {
					return err
				}
				in.Status = &s
			}
			if cmd.Flags().Changed("priority") {
				p, err := ParsePriority(cfg.Priorities, priority)
				if err != nil {
					return err
				}
				in.Priority = &p
			}
			if cmd.Flags().Changed("due") {
				d, err := ParseDate(dueDate)
				if err != nil {
					return err
				}
				in.DueDate = d
			}
			if cmd.Flags().Changed("tags") {
				var tagList []string
				for _, t := range strings.Split(tags, ",") {
					if s := strings.TrimSpace(t); s != "" {
						tagList = append(tagList, s)
					}
				}
				in.Tags = tagList
			}

			updated, err := taskSvc.UpdateTask(in)
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stdout, "Updated task #%d: %s\n", updated.NumID, updated.Title)
			return nil
		},
	}

	statusNames := config.StatusNames(cfg.Statuses)
	priNames2 := make([]string, len(cfg.Priorities))
	for i, p := range cfg.Priorities {
		priNames2[i] = p.Name
	}
	cmd.Flags().StringVar(&title, "title", "", "new title")
	cmd.Flags().StringVarP(&description, "description", "D", "", "new description")
	cmd.Flags().StringVarP(&status, "status", "s", "", "new status ("+strings.Join(statusNames, "/")+")")
	cmd.Flags().StringVarP(&priority, "priority", "p", "", "new priority ("+strings.Join(priNames2, "/")+")")
	cmd.Flags().StringVarP(&dueDate, "due", "d", "", "new due date DD-MM-YYYY or YYYY-MM-DD")
	cmd.Flags().StringVarP(&tags, "tags", "t", "", "new comma-separated tags")
	return cmd
}

// ── status ───────────────────────────────────────────────────────────────────

func newStatusCmd(taskSvc *application.TaskService, cfg *config.Config) *cobra.Command {
	statusNames := config.StatusNames(cfg.Statuses)
	return &cobra.Command{
		Use:   "status <id> <status>",
		Short: "Change task status (" + strings.Join(statusNames, "/") + ")",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := resolveTaskID(taskSvc, args[0])
			if err != nil {
				return err
			}
			s, err := ParseStatus(cfg.Statuses, args[1])
			if err != nil {
				return err
			}
			updated, err := taskSvc.UpdateTask(application.UpdateTaskInput{
				ID:     id,
				Status: &s,
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stdout, "Status → %s: %s\n", updated.Status, updated.Title)
			return nil
		},
	}
}

// ── update (add comment) ─────────────────────────────────────────────────────

func newUpdateCmd(taskSvc *application.TaskService) *cobra.Command {
	return &cobra.Command{
		Use:   "update <id> <message>",
		Short: "Add an update/comment to a task",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := resolveTaskID(taskSvc, args[0])
			if err != nil {
				return err
			}
			body := strings.Join(args[1:], " ")
			u, err := taskSvc.AddUpdate(id, body)
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stdout, "Update added at %s\n", u.CreatedAt.Format("02-01-2006 15:04"))
			return nil
		},
	}
}

// ── lists ────────────────────────────────────────────────────────────────────

func newListsCmd(listSvc *application.ListService) *cobra.Command {
	return &cobra.Command{
		Use:   "lists",
		Short: "List all task lists",
		RunE: func(cmd *cobra.Command, args []string) error {
			lists, err := listSvc.GetAllLists()
			if err != nil {
				return err
			}
			if len(lists) == 0 {
				fmt.Println("No lists.")
				return nil
			}
			for _, l := range lists {
				fmt.Printf("[%s] %s  (created %s)\n",
					l.ID[:8], l.Name, l.CreatedAt.Format("02-01-2006"))
			}
			return nil
		},
	}
}

// ── list-create ──────────────────────────────────────────────────────────────

func newListCreateCmd(listSvc *application.ListService) *cobra.Command {
	return &cobra.Command{
		Use:   "list-create <name>",
		Short: "Create a new task list",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			l, err := listSvc.CreateList(args[0], nil)
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stdout, "Created list %s: %s\n", l.ID[:8], l.Name)
			return nil
		},
	}
}

// ── list-rename ──────────────────────────────────────────────────────────────

func newListRenameCmd(listSvc *application.ListService) *cobra.Command {
	return &cobra.Command{
		Use:   "list-rename <id> <new-name>",
		Short: "Rename a task list",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			l, err := listSvc.UpdateList(args[0], args[1])
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stdout, "Renamed list to: %s\n", l.Name)
			return nil
		},
	}
}

// ── list-delete ──────────────────────────────────────────────────────────────

func newListDeleteCmd(listSvc *application.ListService) *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "list-delete <id>",
		Short: "Delete a task list and all its tasks",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !force {
				fmt.Fprintf(os.Stderr, "This will delete the list and ALL its tasks. Use --force to confirm.\n")
				return fmt.Errorf("operation requires --force flag")
			}
			if err := listSvc.DeleteList(args[0]); err != nil {
				return err
			}
			fmt.Println("List deleted.")
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "confirm destructive deletion")
	return cmd
}
