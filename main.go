package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
)

func usage() {
	intro := `no is a NixOS and Home Manager CLI helper written in Go.

    Usage:
        no [flags] <command> [command flags]`
	fmt.Fprintln(os.Stderr, intro)

	fmt.Fprintln(os.Stderr, "\nCommands:")
	// print commands help

	fmt.Fprintln(os.Stderr, "\nFlags:")
	flag.PrintDefaults()

	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Run `no <command> -h` to get help for a specific command")
}

func main() {
	hostName, err := os.Hostname()
	if err != nil {
		log.Fatal(err)
	}

	// homeDir, err := os.UserHomeDir()
	// err = os.Chdir(homeDir + "/dotfiles/")

	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	flag.StringVar(&hostName, "host", hostName, "Useful if the current hostname differs from the configuration")
	flag.StringVar(&cwd, "dir", cwd, "Sets the directory for the command to run in")

	rebuildSub := flag.NewFlagSet("rebuild", flag.ExitOnError)
	rebuildHost := rebuildSub.String("host", hostName, "host")

	updateSub := flag.NewFlagSet("update", flag.ExitOnError)

	flag.Usage = usage

	if len(os.Args) < 2 {
		flag.Usage()
		os.Exit(1)
	}

	err = os.Chdir(cwd)

	switch os.Args[1] {

	case "rebuild":
		rebuildSub.Parse(os.Args[2:])
		fmt.Println("Rebuilding NixOS for", *rebuildHost, "...")

		logFile, err := os.Create(path.Join(cwd, "nixos-rebuild.log"))
		if err != nil {
			log.Fatal(err)
		}
		defer logFile.Close()

		rebuild := exec.Command("nixos-rebuild", "dry-activate", "--flake", ".#"+*rebuildHost)

		multiWriter := io.MultiWriter(logFile, os.Stdout)

		rebuild.Stdout = multiWriter
		rebuild.Stderr = multiWriter

		err = rebuild.Run()
		if err != nil {
			log.Fatal(err)
		}
	case "update":
		updateSub.Parse(os.Args[2:])
		fmt.Println("Updating flake.lock...")
	}
}
