package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

var (
	ErrNotFound         = errors.New("task not found")
	ErrEmptyDescription = errors.New("description is empty")
	ErrUsage            = errors.New("usage")
)

type Status string

const (
	Todo       Status = "todo"
	InProgress Status = "in-progress"
	Done       Status = "done"
)

const FileName = "tasks.json"

type Task struct {
	ID          int       `json:"id"`
	Description string    `json:"description"`
	Status      Status    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func Load(name string) ([]Task, error) {
	data, err := os.ReadFile(name)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}

		return nil, fmt.Errorf("load %s: %w", name, err)
	}

	if len(bytes.TrimSpace(data)) == 0 {
		return nil, nil
	}

	var tasks []Task
	if err := json.Unmarshal(data, &tasks); err != nil {
		return nil, fmt.Errorf("parse %s: %w", name, err)
	}

	for _, task := range tasks {
		if _, err := ParseStatus(string(task.Status)); err != nil {
			return nil, fmt.Errorf("parse %s: task %d: %w", name, task.ID, err)
		}
	}

	return tasks, nil
}

func Save(name string, tasks []Task) error {
	if tasks == nil {
		tasks = []Task{}
	}

	tmp, err := os.CreateTemp(filepath.Dir(name), "tasks-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}

	defer func() {
		tmp.Close()
		os.Remove(tmp.Name())
	}()

	if err := tmp.Chmod(0644); err != nil {
		return fmt.Errorf("chmod %s: %w", tmp.Name(), err)
	}

	enc := json.NewEncoder(tmp)
	enc.SetIndent("", "  ")
	if err := enc.Encode(tasks); err != nil {
		return fmt.Errorf("write %s: %w", tmp.Name(), err)
	}

	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("sync %s: %w", tmp.Name(), err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close %s: %w", tmp.Name(), err)
	}

	if err := os.Rename(tmp.Name(), name); err != nil {
		return fmt.Errorf("replace %s: %w", name, err)
	}

	return nil
}

func ParseStatus(s string) (Status, error) {
	switch Status(s) {
	case Todo, InProgress, Done:
		return Status(s), nil
	default:
		return "", fmt.Errorf("unknown status %q", s)
	}
}

func NextID(tasks []Task) int {
	maxID := 0
	for _, task := range tasks {
		maxID = max(maxID, task.ID)
	}

	return maxID + 1
}

func find(tasks []Task, id int) (int, error) {
	i := slices.IndexFunc(tasks, func(task Task) bool {
		return task.ID == id
	})

	if i == -1 {
		return -1, fmt.Errorf("task %d: %w", id, ErrNotFound)
	}

	return i, nil
}

func Add(tasks []Task, description string) ([]Task, Task, error) {
	description = strings.TrimSpace(description)
	if description == "" {
		return tasks, Task{}, ErrEmptyDescription
	}

	now := time.Now()
	task := Task{
		ID:          NextID(tasks),
		Description: description,
		Status:      Todo,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	tasks = append(tasks, task)

	return tasks, task, nil
}

func Update(tasks []Task, id int, description string) error {
	taskIndex, err := find(tasks, id)
	if err != nil {
		return err
	}

	description = strings.TrimSpace(description)
	if description == "" {
		return ErrEmptyDescription
	}

	tasks[taskIndex].Description = description
	tasks[taskIndex].UpdatedAt = time.Now()

	return nil
}

func Delete(tasks []Task, id int) ([]Task, error) {
	taskIndex, err := find(tasks, id)
	if err != nil {
		return tasks, err
	}

	return slices.Delete(tasks, taskIndex, taskIndex+1), nil
}

func SetStatus(tasks []Task, id int, status Status) error {
	taskIndex, err := find(tasks, id)
	if err != nil {
		return err
	}

	tasks[taskIndex].Status = status
	tasks[taskIndex].UpdatedAt = time.Now()

	return nil
}

func Filter(tasks []Task, status Status) []Task {
	result := make([]Task, 0, len(tasks))
	for _, task := range tasks {
		if task.Status == status {
			result = append(result, task)
		}
	}

	return result
}

func run(args []string) error {
	if len(args) == 0 {
		usage(os.Stderr)
		return fmt.Errorf("%w: no command given", ErrUsage)
	}
	command, rest := args[0], args[1:]

	switch command {
	case "add":
		return cmdAdd(rest)
	case "update":
		return cmdUpdate(rest)
	case "delete":
		return cmdDelete(rest)
	case "mark-in-progress":
		return cmdMark(rest, InProgress)
	case "mark-done":
		return cmdMark(rest, Done)
	case "list":
		return cmdList(rest)
	case "help", "-h", "--help":
		usage(os.Stdout)
		return nil
	default:
		usage(os.Stderr)
		return fmt.Errorf("%w: unknown command %q", ErrUsage, command)
	}
}

func mutate(fn func([]Task) ([]Task, error)) error {
	tasks, err := Load(FileName)
	if err != nil {
		return err
	}

	tasks, err = fn(tasks)
	if err != nil {
		return err
	}

	return Save(FileName, tasks)
}

func cmdAdd(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("%w: task-cli add <description>", ErrUsage)
	}

	var added Task

	err := mutate(func(tasks []Task) ([]Task, error) {
		var err error
		tasks, added, err = Add(tasks, args[0])
		return tasks, err
	})
	if err != nil {
		return err
	}

	fmt.Printf("Task added successfully (ID: %d)\n", added.ID)
	return nil
}

func cmdUpdate(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("%w: task-cli update <id> <description>", ErrUsage)
	}

	id, err := parseID(args[0])
	if err != nil {
		return err
	}

	err = mutate(func(tasks []Task) ([]Task, error) {
		return tasks, Update(tasks, id, args[1])
	})
	if err != nil {
		return err
	}

	fmt.Printf("Task updated successfully (ID: %d)\n", id)
	return nil
}

func cmdDelete(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("%w: task-cli delete <id>", ErrUsage)
	}

	id, err := parseID(args[0])
	if err != nil {
		return err
	}

	err = mutate(func(tasks []Task) ([]Task, error) {
		return Delete(tasks, id)
	})
	if err != nil {
		return err
	}

	fmt.Printf("Task deleted successfully (ID: %d)\n", id)
	return nil
}

func cmdMark(args []string, status Status) error {
	if len(args) != 1 {
		return fmt.Errorf("%w: task-cli mark-done|mark-in-progress <id>", ErrUsage)
	}

	id, err := parseID(args[0])
	if err != nil {
		return err
	}

	err = mutate(func(tasks []Task) ([]Task, error) {
		return tasks, SetStatus(tasks, id, status)
	})
	if err != nil {
		return err
	}

	fmt.Printf("Task status updated successfully (ID: %d)\n", id)
	return nil
}

func cmdList(args []string) error {
	if len(args) > 1 {
		return fmt.Errorf("%w: task-cli list [todo|in-progress|done]", ErrUsage)
	}

	tasks, err := Load(FileName)
	if err != nil {
		return err
	}

	if len(args) == 1 {
		status, err := ParseStatus(args[0])
		if err != nil {
			return err
		}
		tasks = Filter(tasks, status)
	}

	if len(tasks) == 0 {
		fmt.Println("No tasks found")
		return nil
	}

	for _, task := range tasks {
		fmt.Printf("%-4d %-12s %s\n", task.ID, task.Status, task.Description)
	}

	return nil
}

func parseID(s string) (int, error) {
	id, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid task ID %q", s)
	}

	return id, nil
}

func usage(w io.Writer) {
	fmt.Fprintln(w, `Usage: task-cli <command> [arguments]

Commands:
  add <description>          add a new task
  update <id> <description>  change a task description
  delete <id>                remove a task
  mark-in-progress <id>      set status to in-progress
  mark-done <id>             set status to done
  list [status]              list tasks (status: todo, in-progress, done)
  help                       show this help`)
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		if errors.Is(err, ErrUsage) {
			os.Exit(2)
		}
		os.Exit(1)
	}
}
