package main

import (
	"log"
	"os"

	"github.com/digitalysin/go-telnet/client"
	"github.com/digitalysin/go-telnet/commandline"
)

type goTelnet struct{}

func newGoTelnet() *goTelnet {
	return new(goTelnet)
}

func (g *goTelnet) run() {
	telnetClient := g.createTelnetClient()
	if err := telnetClient.ProcessData(os.Stdin, os.Stdout); err != nil {
		log.Fatalf("%s", err)
	}
}

func (g *goTelnet) createTelnetClient() *client.TelnetClient {
	commandLine := commandline.Read()
	telnetClient := client.NewTelnetClient(commandLine)
	return telnetClient
}
