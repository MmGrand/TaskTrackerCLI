package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
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

	data, err := json.Marshal(tasks)
	if err != nil {
		return fmt.Errorf("serialize: %w", err)
	}

	if err := os.WriteFile(name, data, 0644); err != nil {
		return fmt.Errorf("write %s: %w", name, err)
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

func IndexByID(tasks []Task, id int) int {
	for i, task := range tasks {
		if task.ID == id {
			return i
		}
	}

	return -1
}

func Add(tasks []Task, description string) ([]Task, Task, error) {
	description = strings.TrimSpace(description)
	if description == "" {
		return tasks, Task{}, errors.New("description is empty")
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
	taskIndex := IndexByID(tasks, id)
	if taskIndex == -1 {
		return fmt.Errorf("task with id %d not found", id)
	}

	description = strings.TrimSpace(description)
	if description == "" {
		return errors.New("description is empty")
	}

	tasks[taskIndex].Description = description
	tasks[taskIndex].UpdatedAt = time.Now()

	return nil
}

func Delete(tasks []Task, id int) ([]Task, error) {
	taskIndex := IndexByID(tasks, id)
	if taskIndex == -1 {
		return tasks, fmt.Errorf("task with id %d not found", id)
	}

	result := append(tasks[:taskIndex], tasks[taskIndex+1:]...)

	return result, nil
}

func SetStatus(tasks []Task, id int, status Status) error {
	taskIndex := IndexByID(tasks, id)
	if taskIndex == -1 {
		return fmt.Errorf("task with id %d not found", id)
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
		usage()
		return errors.New("no command given")
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
	default:
		usage()
		return fmt.Errorf("unknown command %q", command)
	}
}

func cmdAdd(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: task-cli add <description>")
	}

	tasks, err := Load(FileName)
	if err != nil {
		return err
	}

	tasks, task, err := Add(tasks, args[0])
	if err != nil {
		return err
	}

	if err := Save(FileName, tasks); err != nil {
		return err
	}

	fmt.Printf("Task added successfully (ID: %d)\n", task.ID)
	return nil
}

func cmdUpdate(args []string) error {
	if len(args) != 2 {
		return errors.New("usage: task-cli update <id> <description>")
	}

	id, err := parseID(args[0])
	if err != nil {
		return err
	}

	tasks, err := Load(FileName)
	if err != nil {
		return err
	}

	err = Update(tasks, id, args[1])
	if err != nil {
		return err
	}

	if err := Save(FileName, tasks); err != nil {
		return err
	}

	fmt.Printf("Task updated successfully (ID: %d)\n", id)
	return nil
}

func cmdDelete(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: task-cli delete <id>")
	}

	id, err := parseID(args[0])
	if err != nil {
		return err
	}

	tasks, err := Load(FileName)
	if err != nil {
		return err
	}

	tasks, err = Delete(tasks, id)
	if err != nil {
		return err
	}

	if err := Save(FileName, tasks); err != nil {
		return err
	}

	fmt.Printf("Task deleted successfully (ID: %d)\n", id)
	return nil
}

func cmdMark(args []string, status Status) error {
	if len(args) != 1 {
		return errors.New("usage: task-cli mark-done|mark-in-progress <id>")
	}

	id, err := parseID(args[0])
	if err != nil {
		return err
	}

	tasks, err := Load(FileName)
	if err != nil {
		return err
	}

	err = SetStatus(tasks, id, status)
	if err != nil {
		return err
	}

	if err := Save(FileName, tasks); err != nil {
		return err
	}

	fmt.Printf("Task status updated successfully (ID: %d)\n", id)
	return nil
}

func cmdList(args []string) error {
	if len(args) > 1 {
		return errors.New("usage: task-cli list [todo|in-progress|done]")
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

func usage() {
	fmt.Fprintln(os.Stderr, `Usage: task-cli <command> [arguments]

Commands:
  add <description>          add a new task
  update <id> <description>  change a task description
  delete <id>                remove a task
  mark-in-progress <id>      set status to in-progress
  mark-done <id>             set status to done
  list [status]              list tasks (status: todo, in-progress, done)`)
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
