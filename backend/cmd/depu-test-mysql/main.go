package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/dengbin9009/DePu/backend/internal/testmysql"
)

func main() {
	os.Exit(run())
}

func run() (exitCode int) {
	label := flag.String("label", "multi_account", "temporary database label")
	workingDirectory := flag.String("cwd", "", "child command working directory")
	flag.Parse()
	if flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "usage: depu-test-mysql [-label name] [-cwd path] command [args...]")
		return 2
	}

	database, err := testmysql.CreateDatabase(testmysql.AdminDSN(), *label)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer func() {
		if err := database.Cleanup(); err != nil {
			fmt.Fprintf(os.Stderr, "cleanup mysql test database %s: %v\n", database.Name, err)
			if exitCode == 0 {
				exitCode = 1
			}
		}
	}()

	fmt.Fprintf(os.Stderr, "[test-mysql] run=%s database=%s\n", testmysql.RunID(), database.Name)
	command := exec.Command(flag.Arg(0), flag.Args()[1:]...)
	command.Dir = *workingDirectory
	command.Env = append(os.Environ(),
		"DEPU_DB_DRIVER=mysql",
		"DEPU_DSN="+database.DSN,
		"DEPU_TEST_DATABASE="+database.Name,
		"DEPU_TEST_RUN_ID="+testmysql.RunID(),
	)
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Start(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	signals := make(chan os.Signal, 1)
	forwardingDone := make(chan struct{})
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)
	go func() {
		select {
		case receivedSignal := <-signals:
			_ = command.Process.Signal(receivedSignal)
		case <-forwardingDone:
		}
	}()

	err = command.Wait()
	close(forwardingDone)
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			return exitError.ExitCode()
		}
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}
