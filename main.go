package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"slices"
)

var err error

type Command struct {
	Name string
	Help string
	Run  func(args []string) error
}

var commands = []Command{
	{Name: "garbage", Help: "Run garbage collection and remove old generations", Run: garbageCmd},
	{Name: "rebuild", Help: "Rebuild a NixOS configuration", Run: rebuildCmd},
	{Name: "update", Help: "Update a flake.lock file", Run: updateCmd},
	{Name: "help", Help: "Print this help", Run: printHelpCmd},
}

func printHelpCmd(_ []string) error {
	flag.Usage()
	return nil
}

func garbageCmd(args []string) error {

	flagSet := flag.NewFlagSet("garbage", flag.ExitOnError)
	flagSet.Usage = func() {
		fmt.Fprintln(os.Stderr, `Run garbage collection and remove old generations.

Usage:
    no garbage [flags]

Flags:
    -h, --help  Print this help.`)
		fmt.Fprintln(os.Stderr)
	}
	flagSet.Parse(args)

	fmt.Println("Starting system cleanup...")

	trashCmd := exec.Command("nix-collect-garbage", "-d")

	trashCmd.Stdout = os.Stdout
	trashCmd.Stderr = os.Stdout

	err = trashCmd.Run()
	if err != nil {
		log.Fatal(err)
	}

	profileCmd := exec.Command("sudo", "nix", "profile", "wipe-history", "--profile", "/nix/var/nix/profiles/system", "--older-than", "7d")

	profileCmd.Stdout = os.Stdout
	profileCmd.Stderr = os.Stdout

	err = profileCmd.Run()
	if err != nil {
		log.Fatal(err)
	}

	return nil
}

func rebuildCmd(args []string) error {
	var dir string
	var activate = "switch"

	hostName, err := os.Hostname()
	if err != nil {
		log.Fatal(err)
	}

	flagSet := flag.NewFlagSet("rebuild", flag.ExitOnError)

	flagSet.StringVar(&hostName, "config", hostName, "nixos configuration to use")
	flagSet.StringVar(&hostName, "c", hostName, "nixos configuration to use")

	flagSet.StringVar(&dir, "directory", ".", "run in this dir")
	flagSet.StringVar(&dir, "d", ".", "run in this dir")

	flagSet.Func("activate", "rebuild activation", func(flagValue string) error {
		for _, allowedValue := range []string{"boot", "dry-activate", "switch"} {
			if flagValue == allowedValue {
				activate = flagValue
				return nil
			}
		}
		fmt.Fprintf(os.Stderr, `must be one of: "boot", "dry-activate", "switch"\n\n`)
		flagSet.Usage()
		os.Exit(1)

		return nil
	})
	flagSet.Func("a", "rebuild activation", func(flagValue string) error {
		for _, allowedValue := range []string{"boot", "dry-activate", "switch"} {
			if flagValue == allowedValue {
				activate = flagValue
				return nil
			}
		}
		fmt.Fprintf(os.Stderr, `must be one of: "boot", "dry-activate", "switch"\n\n`)
		flagSet.Usage()
		os.Exit(1)

		return nil
	})

	flagSet.Usage = func() {
		fmt.Fprintln(os.Stderr, `Rebuild a NixOS configuration.

Usage:
    no rebuild [flags]

Flags:
    -a, --activate   STRING  Set how the rebuild will be activated. (default 'switch')
    -c, --config     STRING  Specify which nixos configuration. (default 'hostname')
    -d, --directory  STRING  Run in this directory, must be full path. (default '.')
    -h, --help               Print this help.`)
		fmt.Fprintln(os.Stderr)
	}
	flagSet.Parse(args)

	err = os.Chdir(dir)
	fmt.Println("Rebuilding NixOS...")

	// Not sure if I want the log file, I like the default output better than piped
	// logFile, err := os.Create(path.Join(dir, "nixos-rebuild.log"))
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// defer logFile.Close()

	cmd := exec.Command("sudo", "nixos-rebuild", activate, "--flake", ".#"+hostName)

	// multiWriter := io.MultiWriter(logFile, os.Stdout)
	//
	// cmd.Stdout = multiWriter
	// cmd.Stderr = multiWriter
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout

	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}

	return nil
}

func updateCmd(args []string) error {
	var dir string
	var switchBool bool

	hostName, err := os.Hostname()
	if err != nil {
		log.Fatal(err)
	}

	flagSet := flag.NewFlagSet("rebuild", flag.ExitOnError)

	flagSet.StringVar(&dir, "directory", ".", "run in this dir")
	flagSet.StringVar(&dir, "d", ".", "run in this dir")

	flagSet.BoolVar(&switchBool, "switch", false, "switch after update")
	flagSet.BoolVar(&switchBool, "s", false, "switch after update")

	flagSet.Usage = func() {
		fmt.Fprintln(os.Stderr, `Update a 'flake.lock' file.

Usage:
    no update [flags]

Flags:
    -d, --directory  STRING  Run in this directory, must be full path. (default '.')
    -s, --switch     BOOL    Rebuild system config and activate on boot. (default 'false')
    -h, --help               Print this help.`)
		fmt.Fprintln(os.Stderr)
	}
	flagSet.Parse(args)

	err = os.Chdir(dir)
	fmt.Printf("Updating flake in %s...", dir)

	updateCmd := exec.Command("sudo", "nix", "flake", "update")

	updateCmd.Stdout = os.Stdout
	updateCmd.Stderr = os.Stdout

	err = updateCmd.Run()
	if err != nil {
		log.Fatal(err)
	}

	if switchBool == true {
		fmt.Println("Rebuilding NixOS...")

		switchCmd := exec.Command("sudo", "nixos-rebuild", "boot", "--flake", ".#"+hostName)

		switchCmd.Stdout = os.Stdout
		switchCmd.Stderr = os.Stdout

		err = switchCmd.Run()
		if err != nil {
			log.Fatal(err)
		}
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
