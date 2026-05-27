package main

import (
	"fmt"
	"log"

	"github.com/fabienChaillou/terraform-commander/terraform/command"
)

func main() {

	log.Print("Start terraform commander module!")
	r := command.NewRegistry()

	for _, v := range r.Actions() {
		fmt.Println("CMD name: ", v)
	}

	cmd, ok := r.Lookup("init")
	if !ok {
		log.Fatal("ERROR cmd tf not found!")
	}

	log.Print("TF CMD found: ", cmd.Help())
	log.Print("End terraform commander module!")
}
