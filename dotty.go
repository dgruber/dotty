package main

import (
	"fmt"
	"github.com/dgruber/drmaa2interface"
	"github.com/dgruber/wfl"
	"github.com/dgruber/wfl/pkg/context/docker"
	"os"
	"os/signal"
	"syscall"
)

func panicf(e error) {
	panic(e)
}

func gottyDirectory() string {
	return fmt.Sprintf("%s/src/github.com/dgruber/dotty/linuxamd64", os.Getenv("GOPATH"))
}

func cli() string {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "dotty <docker/image>\n") // "cloudfoundry/cflinuxfs3"
		os.Exit(1)
	}
	return os.Args[1]
}

func main() {
	image := cli()

	ctx := docker.NewDockerContext().OnError(panicf)
	wf := wfl.NewWorkflow(ctx).OnError(panicf)

	stageIn := make(map[string]string)
	stageIn[gottyDirectory()] = "/gotty"

	cfTemplate := drmaa2interface.JobTemplate{
		RemoteCommand: "/gotty/gotty",
		Args:          []string{"--permit-write", "/bin/sh"},
		JobCategory:   image,
		OutputPath:    "/dev/stdout",
		ErrorPath:     "/dev/stdout",
		StageInFiles:  stageIn,
	}

	cfTemplate.ExtensionList = map[string]string{
		"exposedPorts": "8789:8080/tcp", // ports redirected from container 8080 to local host 8789
		"user":         "root"}          // user name in container (needs to exist)

	gotty := wf.RunT(cfTemplate).OnError(panicf)

	// macos: start browser
	wfl.NewWorkflow(wfl.NewProcessContext().OnError(panicf)).OnError(panicf).Run("open", "http://localhost:8789")

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-signals
		fmt.Println("killing container")
		gotty.Kill().Wait()
		fmt.Println("exiting")
	}()

	gotty.Wait()

}
