package main

import (
	"errors"
	"fmt"
	"github.com/fatih/structs"
	"github.com/spf13/cobra"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"text/template"
)

var ServicesCmd = &cobra.Command{
	Use:   "services",
	Short: "service management commands",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		pathConfigFile = cmd.Flag("config").Value.String()

		// Check if config file exists
		if _, err := os.Stat(pathConfigFile); os.IsNotExist(err) {
			log.Printf("Services:: Config `%s` file must exist", pathConfigFile)
			return err
		}

		// Check if yamls folder exists otherwise creates it
		if _, err := os.Stat(pathYMLsDir); os.IsNotExist(err) {
			os.MkdirAll(pathYMLsDir, os.ModeDir|os.ModePerm)
		}

		// Check if certs folder exists otherwise creates it
		if _, err := os.Stat(pathCertsDir); os.IsNotExist(err) {
			os.MkdirAll(pathCertsDir, os.ModeDir|os.ModePerm)
		}

		// Check if templates folder does not exists returns
		if _, err := os.Stat(pathTemplatesDir); os.IsNotExist(err) {
			log.Printf("Services:: Templates `%s` folder must exists", pathTemplatesDir)
			return err
		}

		return nil
	},
}

// Install all the enabled services
var InstallCmd = &cobra.Command{
	Use:   "install",
	Short: "install nested services",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Init Flag variables
		clean := cmd.Flag("clean").Changed

		configYaml := LoadNestedConfig()
		if configYaml == nil {
			return errors.New("could not load config file")
		}

		if clean {
			var ans string = "N"
			fmt.Print("You are going to remove all the data, and it is unrecoverable. Are you sure? [y/N] ")
			fmt.Scanf("%s", &ans)
			if ans == "y" {
				os.RemoveAll(pathYMLsDir)
			}
		}

		UpdateYamlFiles(configYaml)
		b, _ := ioutil.ReadFile(pathConfigFile)
		t := template.New("Config File")
		t.Delims("/#/<", ">/#/")
		t, err := t.Parse(string(b))
		if err != nil {
			log.Println(err.Error())
		}

		return nil
	},
}

// Stop all or specified services
var StopCmd = &cobra.Command{
	Use:   "stop",
	Short: "stop services",
	RunE: func(cmd *cobra.Command, args []string) error {
		configYaml := LoadNestedConfig()
		if configYaml == nil {
			return errors.New("could not load config file")
		}

		// Stopping services
		var services []string
		if len(args) > 0 {
			services = args
		} else {
			services = structs.Names(configYaml.EnabledServices)
		}

		fmt.Println("Stopping Services ...")
		for _, srv := range services {
			configYaml.StopService(srv)
		}
		return nil
	},
}

// Start all or specified services
var StartCmd = &cobra.Command{
	Use:   "start",
	Short: "start services",
	RunE: func(cmd *cobra.Command, args []string) error {
		configYaml := LoadNestedConfig()
		if configYaml == nil {
			return errors.New("could not load config file")
		}

		// Creating Docker Networks
		fmt.Println("Creating Docker Networks ...")
		dockerNetworks := []string{"db-mongo", "cache-redis", "arsaces", "cyrus", "gobryas", "webapp"}
		for _, net := range dockerNetworks {
			exec.Command("docker", "network", "create", fmt.Sprintf("%s-net", net)).Run()
		}

		// Starting Services
		var services []string
		if len(args) > 0 {
			services = args
		} else {
			services = structs.Names(configYaml.EnabledServices)
		}
		fmt.Println("Creating and Running Services ...")
		for _, srv := range services {
			configYaml.StartService(srv)
		}

		return nil
	},
}

// Update all or specified
var UpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "update dockers",
	RunE: func(cmd *cobra.Command, args []string) error {
		configYaml := LoadNestedConfig()
		if configYaml == nil {
			return errors.New("could not load config file")
		}

		var services []string
		if len(args) > 0 {
			services = args
		} else {
			services = structs.Names(configYaml.EnabledServices)
		}
		fmt.Println("Creating and Running Services ...")
		for _, srv := range services {
			configYaml.UpdateService(srv)
		}

		return nil
	},
}

func init() {
	RootCmd.AddCommand(ServicesCmd)
	ServicesCmd.AddCommand(InstallCmd, UpdateCmd, StartCmd, StopCmd)
	InstallCmd.Flags().Bool("clean", false, "Clean install (all data will be removed)")
}
