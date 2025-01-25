package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"slices"
)

type Command struct {
	Name string
	Help string
	Run  func(args []string) error
}

var commands = []Command{
	{Name: "rebuild", Help: "Rebuild a NixOS configuration", Run: rebuildCmd},
	{Name: "update", Help: "Update a flake.lock file", Run: updateCmd},
	{Name: "help", Help: "Print this help", Run: printHelpCmd},
}

func printHelpCmd(_ []string) error {
	flag.Usage()
	return nil
}

func rebuildCmd(args []string) error {
	var dir string

	hostName, err := os.Hostname()
	if err != nil {
		log.Fatal(err)
	}

	flagSet := flag.NewFlagSet("rebuild", flag.ExitOnError)
	flagSet.StringVar(&hostName, "host", hostName, "nixos configuration to use from the flake")
	flagSet.StringVar(&dir, "dir", ".", "directory containing flake.nix, must be full path")
	flagSet.Usage = func() {
		fmt.Fprintln(os.Stderr, `Rebuild a NixOS configuration.

Usage:
    no rebuild [flags]

Flags:`)
		flagSet.PrintDefaults()
		fmt.Fprintln(os.Stderr)
	}
	flagSet.Parse(args)

	err = os.Chdir(dir)
	fmt.Println("Rebuilding NixOS...")

	logFile, err := os.Create(path.Join(dir, "nixos-rebuild.log"))
	if err != nil {
		log.Fatal(err)
	}
	defer logFile.Close()

	cmd := exec.Command("nixos-rebuild", "dry-activate", "--flake", ".#"+hostName)

	multiWriter := io.MultiWriter(logFile, os.Stdout)

	cmd.Stdout = multiWriter
	cmd.Stderr = multiWriter

	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}

	return nil
}

func updateCmd(args []string) error {
	var dir string

	flagSet := flag.NewFlagSet("rebuild", flag.ExitOnError)
	flagSet.StringVar(&dir, "dir", ".", "directory containing flake.nix, must be full path")
	flagSet.Usage = func() {
		fmt.Fprintln(os.Stderr, `Update a 'flake.lock' file.

Usage:
    no update [flags]

Flags:`)
		flagSet.PrintDefaults()
		fmt.Fprintln(os.Stderr)
	}
	flagSet.Parse(args)

	err := os.Chdir(dir)
	fmt.Printf("Updating flake in %s...", dir)

	cmd := exec.Command("nix", "flake", "update")

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout

	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	return nil
}

func usage() {
	intro := `no is a NixOS and Home Manager CLI helper written in Go.

Usage:
  no [flags] <command> [command flags]`
	fmt.Fprintln(os.Stderr, intro)

	fmt.Fprintln(os.Stderr, "\nCommands:")
	for _, cmd := range commands {
		fmt.Fprintf(os.Stderr, "  %-8s %s\n", cmd.Name, cmd.Help)
	}

	fmt.Fprintln(os.Stderr, "\nFlags:")
	flag.PrintDefaults()

	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Run `no <command> -h` to get help for a specific command")
}

func main() {
	// flag.StringVar(&cwd, "dir", cwd, "Sets the directory for the command to run in")

	flag.Usage = usage
	flag.Parse()

	if len(flag.Args()) < 1 {
		flag.Usage()
		os.Exit(1)
	}

	subCmd := flag.Arg(0)
	subCmdArgs := flag.Args()[1:]

	runCommand(subCmd, subCmdArgs)
}

func runCommand(name string, args []string) {
	cmdIdx := slices.IndexFunc(commands, func(cmd Command) bool {
		return cmd.Name == name
	})

	if cmdIdx < 0 {
		fmt.Fprintf(os.Stderr, "command \"%s\" not found\n\n", name)
		flag.Usage()
		os.Exit(1)
	}

	if err := commands[cmdIdx].Run(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s", err.Error())
		os.Exit(1)
	}
}
