package main

import (
	"flag"
	"os"
	"os/exec"
	"os/user"
	"slices"

	"github.com/charmbracelet/log"
)

var err error

var logger = log.New(os.Stderr)

type Command struct {
	Name string
	Help string
	Run  func(args []string, dir string) error
}

var commands = []Command{
	{Name: "garbage", Help: "Run garbage collection and remove old generations", Run: garbageCmd},
	{Name: "home", Help: "Rebuild a Home Manager configuration", Run: homeCmd},
	{Name: "rebuild", Help: "Rebuild a NixOS configuration", Run: rebuildCmd},
	{Name: "update", Help: "Update a flake.lock file", Run: updateCmd},
	{Name: "help", Help: "Print this help", Run: printHelpCmd},
}

func printHelpCmd(_ []string, _ string) error {
	flag.Usage()
	return nil
}

func garbageCmd(args []string, _ string) error {
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

func homeCmd(args []string, dir string) error {
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
	flagSet.StringVar(&profile, "profile", profile, "home-manager profile")
	flagSet.StringVar(&profile, "p", profile, "home-manager profile")
	flagSet.Usage = func() {
		logger.Print(`Rebuild a Home Manager configuration.

Usage:
    no home [flags]

Flags:
    -p, --profile  STRING
        Home Manager profile to use. (default 'user@host')
    -h, --help
        Print this help.`)
	}
	flagSet.Parse(args)

	err = os.Chdir(dir)

	logger.Info("Rebuilding Home Manager for " + profile + "...")

	cmd := exec.Command("home-manager", "switch", "--flake", ".#"+profile)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout

	err = cmd.Run()
	if err != nil {
		logger.Fatal(err)
	}

	return nil
}

func rebuildCmd(args []string, dir string) error {
	var activate = "switch"

	hostName, err := os.Hostname()
	if err != nil {
		logger.Fatal(err)
	}

	flagSet := flag.NewFlagSet("rebuild", flag.ExitOnError)

	flagSet.StringVar(&hostName, "config", hostName, "nixos configuration to use")
	flagSet.StringVar(&hostName, "c", hostName, "nixos configuration to use")

	flagSet.Func("activate", "rebuild activation", func(flagValue string) error {
		for _, allowedValue := range []string{"boot", "dry-activate", "switch"} {
			if flagValue == allowedValue {
				activate = flagValue
				return nil
			}
		}
		logger.Error(`must be one of: "boot", "dry-activate", "switch"\n\n`)
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
		logger.Error(`must be one of: "boot", "dry-activate", "switch"\n\n`)
		flagSet.Usage()
		os.Exit(1)

		return nil
	})

	flagSet.Usage = func() {
		logger.Print(`Rebuild a NixOS configuration.

Usage:
    no rebuild [flags]

Flags:
    -a, --activate  STRING
        Set how the rebuild will be activated. (default 'switch')
    -c, --config  STRING
        Specify which nixos configuration. (default 'hostname')
    -h, --help
        Print this help.`)
	}
	flagSet.Parse(args)

	err = os.Chdir(dir)
	logger.Info("Rebuilding NixOS for " + hostName + "...")

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
		logger.Fatal(err)
	}

	return nil
}

func updateCmd(args []string, dir string) error {
	var switchBool bool

	hostName, err := os.Hostname()
	if err != nil {
		logger.Fatal(err)
	}

	flagSet := flag.NewFlagSet("rebuild", flag.ExitOnError)

	flagSet.BoolVar(&switchBool, "switch", false, "switch after update")
	flagSet.BoolVar(&switchBool, "s", false, "switch after update")

	flagSet.Usage = func() {
		logger.Print(`Update a 'flake.lock' file.

Usage:
    no update [flags]

Flags:
    -s, --switch  BOOL
        Rebuild system config and activate on boot. (default 'false')
    -h, --help
        Print this help.`)
	}
	flagSet.Parse(args)

	err = os.Chdir(dir)
	logger.Infof("Updating flake in %s...\n", dir)

	updateCmd := exec.Command("sudo", "nix", "flake", "update")

	updateCmd.Stdout = os.Stdout
	updateCmd.Stderr = os.Stdout

	err = updateCmd.Run()
	if err != nil {
		logger.Fatal(err)
	}

	if switchBool == true {
		logger.Info("Rebuilding NixOS...")

		switchCmd := exec.Command(
			"sudo",
			"nixos-rebuild",
			"boot",
			"--flake",
			".#"+hostName)

		switchCmd.Stdout = os.Stdout
		switchCmd.Stderr = os.Stdout

		err = switchCmd.Run()
		if err != nil {
			logger.Fatal(err)
		}
	}

	return nil
}

func usage() {
	intro := `no is a NixOS and Home Manager CLI helper written in Go.

Usage:
    no [flags] <command> [command flags]`
	logger.Print(intro)

	logger.Print("\nCommands:")
	for _, cmd := range commands {
		logger.Printf("    %-8s %s", cmd.Name, cmd.Help)
	}

	logger.Print(`
Flags:
    -d, --directory  PATH
        Run in this directory, must be full path. (default '.')
    -h, --help
        Print this help.`)

	logger.Print("\nRun `no <command> -h` to get help for a specific command")
}

func main() {
	var dir string

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

	runCommand(subCmd, subCmdArgs, dir)
}

func runCommand(name string, args []string, dir string) {
	cmdIdx := slices.IndexFunc(commands, func(cmd Command) bool {
		return cmd.Name == name
	})

	if cmdIdx < 0 {
		logger.Errorf("command \"%s\" not found\n\n", name)
		flag.Usage()
		os.Exit(1)
	}

	if err := commands[cmdIdx].Run(args, dir); err != nil {
		logger.Errorf("Error: %s", err.Error())
		os.Exit(1)
	}
}
