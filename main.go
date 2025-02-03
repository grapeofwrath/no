package main

import (
	"flag"
	"os"
	"os/exec"
	"os/user"
	"slices"
	"strings"

	"github.com/charmbracelet/log"
)

type Command struct {
	Name string
	Help string
	Run  func(args []string) error
}

type Operation struct {
	Name string
	Help string
}

var err error
var logger = log.New(os.Stderr)
var dir string

var commands = []Command{
	{
		Name: "garbage",
		Help: "Run garbage collection and remove old generations",
		Run:  garbageCmd,
	},
	{
		Name: "home",
		Help: "Rebuild a Home Manager configuration",
		Run:  homeCmd,
	},
	{
		Name: "rebuild",
		Help: "Rebuild a NixOS configuration",
		Run:  rebuildCmd,
	},
	{
		Name: "update",
		Help: "Update a flake.lock file",
		Run:  updateCmd,
	},
	{
		Name: "help",
		Help: "Print this help",
		Run:  printHelpCmd,
	},
}

func printHelpCmd(_ []string) error {
	flag.Usage()
	return nil
}

func garbageCmd(args []string) error {
	var burn bool

	flagSet := flag.NewFlagSet("garbage", flag.ExitOnError)
	flagSet.BoolVar(&burn, "burn", false, "Remove all old configurations")
	flagSet.BoolVar(&burn, "b", false, "Remove all old configurations")
	flagSet.Usage = func() {
		logger.Print(`Run garbage collection and remove old generations.

Usage:

    no garbage [flags]

Flags:

    -b, --burn  BOOL
        Removes all previous system configurations from boot

    -h, --help
        Print this help.`)
	}
	flagSet.Parse(args)

	logger.Info("Starting system cleanup...")

	sysTrashCmd := exec.Command("sudo", "nix-collect-garbage", "-d")

	sysTrashCmd.Stdout = os.Stdout
	sysTrashCmd.Stderr = os.Stdout

	err = sysTrashCmd.Run()
	if err != nil {
		logger.Fatal(err)
	}

	userTrashCmd := exec.Command("nix-collect-garbage", "-d")

	userTrashCmd.Stdout = os.Stdout
	userTrashCmd.Stderr = os.Stdout

	err = userTrashCmd.Run()
	if err != nil {
		logger.Fatal(err)
	}

	switch burn {
	case false:
		profileCmd := exec.Command(
			"sudo",
			"nix",
			"profile",
			"wipe-history",
			"--profile",
			"/nix/var/nix/profiles/system",
			"--older-than",
			"7d")

		profileCmd.Stdout = os.Stdout
		profileCmd.Stderr = os.Stdout

		err = profileCmd.Run()
		if err != nil {
			logger.Fatal(err)
		}

	case true:
		logger.Warn("BURN ORDER ACTIVATED")
		logger.Print("purging all previous system configurations from boot...")
		profileBurnCmd := exec.Command(
			"sudo",
			"/run/current-system/bin/switch-to-configuration",
			"boot")

		profileBurnCmd.Stdout = os.Stdout
		profileBurnCmd.Stderr = os.Stderr

		err = profileBurnCmd.Run()
		if err != nil {
			logger.Fatal(err)
		}
	}

	return nil
}

func homeCmd(args []string) error {
	var operation = "switch"
	var operations = []Operation{
		{
			Name: "build",
			Help: "Build the new configuration into result directory"},
		{
			Name: "instantiate",
			Help: "Instantiate the new configurations and print the result"},
		{
			Name: "switch",
			Help: "Build and activate the new configuration"},
	}
	var opsHelp []string
	for _, op := range operations {
		opsHelp = append(opsHelp, op.Name+"\n        "+op.Help)
	}
	var opsHelpMsg = strings.Join(opsHelp, "\n\n    ")

	user, err := user.Current()
	if err != nil {
		logger.Fatal(err)
	}

	hostName, err := os.Hostname()
	if err != nil {
		logger.Fatal(err)
	}

	profile := user.Username + "@" + hostName

	flagSet := flag.NewFlagSet("home", flag.ExitOnError)

	flagSet.Func("operation", "rebuild operation", func(flagValue string) error {
		for _, op := range operations {
			if flagValue == op.Name {
				operation = flagValue
				return nil
			}
		}
		logger.Errorf("operation must be one of:\n\n    %s\n", opsHelpMsg)
		flagSet.Usage()
		os.Exit(1)

		return nil
	})
	flagSet.Func("o", "rebuild operation", func(flagValue string) error {
		for _, op := range operations {
			if flagValue == op.Name {
				operation = flagValue
				return nil
			}
		}
		logger.Errorf("operation must be one of:\n\n    %s\n", opsHelpMsg)
		flagSet.Usage()
		os.Exit(1)

		return nil
	})

	flagSet.StringVar(&profile, "profile", profile, "home-manager profile")
	flagSet.StringVar(&profile, "p", profile, "home-manager profile")

	flagSet.Usage = func() {
		logger.Print(`Manage a Home Manager configuration.

Usage:

    no home [flags]

Flags:

    -o, --operation  STRING
        Specify which operation to run. (default 'switch')

    -p, --profile  STRING
        Home Manager profile to use. (default 'user@host')

    -h, --help
        Print this help.

Examples:

    Build a configuration for the specified profile
        no home -o build -p <user>-<host>`)
	}
	flagSet.Parse(args)

	err = os.Chdir(dir)

	logger.Info("Rebuilding Home Manager for " + profile + "...")

	cmd := exec.Command(
		"home-manager",
		operation,
		"--flake",
		".#"+profile)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout

	err = cmd.Run()
	if err != nil {
		logger.Fatal(err)
	}

	return nil
}

func rebuildCmd(args []string) error {
	var operation = "switch"
	var operations = []Operation{
		{
			Name: "boot",
			Help: "Build the new configuration and make it the boot default"},
		{
			Name: "build",
			Help: "Build the new configuration into result directory"},
		{
			Name: "build-vm",
			Help: "Build a script that starts a NixOS virtual machine with the desired configuration"},
		{
			Name: "build-vm-with-bootloader",
			Help: "Like build-vm, but boots using the regular boot loader of your configuration"},
		{
			Name: "dry-activate",
			Help: "Build the new configuration, but show what changes would be performed instead of activating it"},
		{
			Name: "switch",
			Help: "Build and activate the new configuration"},
		{
			Name: "test",
			Help: "Build and activate the new configuration, but do not add it to the boot menu"},
	}
	var opsHelp []string
	for _, op := range operations {
		opsHelp = append(opsHelp, op.Name+"\n        "+op.Help)
	}
	var opsHelpMsg = strings.Join(opsHelp, "\n\n    ")

	var hostName, err = os.Hostname()
	if err != nil {
		logger.Fatal(err)
	}

	flagSet := flag.NewFlagSet("rebuild", flag.ExitOnError)

	flagSet.StringVar(&hostName, "config", hostName, "nixos configuration to use")
	flagSet.StringVar(&hostName, "c", hostName, "nixos configuration to use")

	flagSet.Func("operation", "rebuild operation", func(flagValue string) error {
		for _, op := range operations {
			if flagValue == op.Name {
				operation = flagValue
				return nil
			}
		}
		logger.Errorf("operation must be one of:\n\n    %s\n", opsHelpMsg)
		flagSet.Usage()
		os.Exit(1)

		return nil
	})
	flagSet.Func("o", "rebuild operation", func(flagValue string) error {
		for _, op := range operations {
			if flagValue == op.Name {
				operation = flagValue
				return nil
			}
		}
		logger.Errorf("operation must be one of:\n\n    %s\n", opsHelpMsg)
		flagSet.Usage()
		os.Exit(1)

		return nil
	})

	flagSet.Usage = func() {
		logger.Print(`Rebuild a NixOS configuration.

Usage:

    no rebuild [flags]

Flags:

    -c, --config  STRING
        Specify which nixos configuration. (default 'hostname')

    -o, --operation  STRING
        Specify which operation to run. (default 'switch')

    -h, --help
        Print this help.

Examples:

    Rebuild the current configuration and activate on boot
        no rebuild -o boot

    Rebuild a specific configuration and dry-activate it
        no rebuild -c <configName> -o dry-activate`)
	}
	flagSet.Parse(args)

	err = os.Chdir(dir)
	logger.Info("Rebuilding NixOS for " + hostName + "...")

	cmd := exec.Command(
		"sudo",
		"nixos-rebuild",
		operation,
		"--flake",
		".#"+hostName)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout

	err = cmd.Run()
	if err != nil {
		logger.Fatal(err)
	}

	return nil
}

func updateCmd(args []string) error {
	var rebuildBool bool

	hostName, err := os.Hostname()
	if err != nil {
		logger.Fatal(err)
	}

	flagSet := flag.NewFlagSet("rebuild", flag.ExitOnError)

	flagSet.BoolVar(&rebuildBool, "rebuild", false, "rebuild after update")
	flagSet.BoolVar(&rebuildBool, "r", false, "rebuild after update")

	flagSet.Usage = func() {
		logger.Print(`Update a 'flake.lock' file.

Usage:

    no update [flags] inputs...

Flags:

    -r, --rebuild  BOOL
        Rebuild system config and activate on boot. (default 'false')

    -h, --help
        Print this help.

Examples:

    Update all inputs of the flake.lock file in the current directory
        no update

    Update a flake.lock and rebuild the system configuration, activating on boot
        no update -r

    Update a single input
        no update nixpkgs

    Update multiple inputs
        no update nixpkgs nixpkgs-unstable`)
	}
	flagSet.Parse(args)

	var inputs = strings.Join(flagSet.Args()[0:], " ")

	err = os.Chdir(dir)
	logger.Infof("Updating flake in %s ...\n", dir)

	updateCmd := exec.Command(
		"sudo",
		"nix",
		"flake",
		"update")
	if len(inputs) != 0 {
		updateCmd = exec.Command(
			"sudo",
			"nix",
			"flake",
			"update",
			inputs)
	}

	updateCmd.Stdout = os.Stdout
	updateCmd.Stderr = os.Stdout

	err = updateCmd.Run()
	if err != nil {
		logger.Fatal(err)
	}

	if rebuildBool == true {
		logger.Info("Rebuilding NixOS...")

		rebuildCmd := exec.Command(
			"sudo",
			"nixos-rebuild",
			"boot",
			"--flake",
			".#"+hostName)

		rebuildCmd.Stdout = os.Stdout
		rebuildCmd.Stderr = os.Stdout

		err = rebuildCmd.Run()
		if err != nil {
			logger.Fatal(err)
		}
	}

	return nil
}

func main() {
	flag.StringVar(&dir, "directory", ".", "run in this dir")
	flag.StringVar(&dir, "d", ".", "run in this dir")

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

func usage() {
	intro := `no is a NixOS and Home Manager CLI helper written in Go.

Usage:

    no [flags] <command> [command flags]`
	logger.Print(intro)

	logger.Print("\nCommands:\n")
	for _, cmd := range commands {
		logger.Printf("    %-8s %s\n", cmd.Name, cmd.Help)
	}

	logger.Print(`
Flags:

    -d, --directory  PATH
        Run in this directory, must be full path. (default '.')

    -h, --help
        Print this help.

Examples:

    Rebuild the current NixOS configuration in the specified directory
        no -d /home/user/dotfiles rebuild`)

	logger.Print("\nRun `no <command> -h` to get help for a specific command")
}

func runCommand(name string, args []string) {
	cmdIdx := slices.IndexFunc(commands, func(cmd Command) bool {
		return cmd.Name == name
	})

	if cmdIdx < 0 {
		logger.Errorf("command \"%s\" not found\n\n", name)
		flag.Usage()
		os.Exit(1)
	}

	if err := commands[cmdIdx].Run(args); err != nil {
		logger.Errorf("Error: %s", err.Error())
		os.Exit(1)
	}
}
