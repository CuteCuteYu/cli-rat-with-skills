package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

const commandsFile = "commands.json"

type Command struct {
	ID          string    `json:"id"`
	Content     string    `json:"content"`
	ClientID    string    `json:"client_id"`
	Status      string    `json:"status"`
	Result      string    `json:"result,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
}

type CommandStore struct {
	Commands []Command `json:"commands"`
}

func loadCommands() CommandStore {
	data, err := os.ReadFile(commandsFile)
	if err != nil {
		return CommandStore{Commands: []Command{}}
	}

	var store CommandStore
	if err := json.Unmarshal(data, &store); err != nil {
		return CommandStore{Commands: []Command{}}
	}
	return store
}

func saveCommands(store CommandStore) {
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		fmt.Printf("序列化命令失败: %v\n", err)
		return
	}

	if err := os.WriteFile(commandsFile, data, 0644); err != nil {
		fmt.Printf("保存命令失败: %v\n", err)
	}
}

func getPendingCommand(clientID string) *Command {
	store := loadCommands()

	for i := range store.Commands {
		if store.Commands[i].ClientID == clientID && store.Commands[i].Status == "pending" {
			return &store.Commands[i]
		}
	}

	return nil
}

func saveCommand(cmd *Command) {
	store := loadCommands()

	for i := range store.Commands {
		if store.Commands[i].ID == cmd.ID {
			store.Commands[i] = *cmd
			saveCommands(store)
			return
		}
	}

	store.Commands = append(store.Commands, *cmd)
	if len(store.Commands) > maxCommands {
		store.Commands = store.Commands[len(store.Commands)-maxCommands:]
	}
	saveCommands(store)
}

func loadCommand(cmdID string) *Command {
	store := loadCommands()

	for _, cmd := range store.Commands {
		if cmd.ID == cmdID {
			return &cmd
		}
	}

	return nil
}

func getCommandByID(cmdID string) *Command {
	return loadCommand(cmdID)
}

func listCommands() []Command {
	store := loadCommands()
	commands := store.Commands
	if len(commands) > maxCommands {
		commands = commands[len(commands)-maxCommands:]
	}
	return commands
}
