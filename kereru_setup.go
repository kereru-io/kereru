package main

import "os"
import "log"
import "fmt"
import "crypto/rand"
import "flag"
import "io/ioutil"
import "path/filepath"
import "gopkg.in/yaml.v2"
import "github.com/tcnksm/go-input"

func main() {
	var configPath = flag.String("config", "/etc/kereru/config.yml", "path to the config file")
	flag.Parse()

	ConfigFilename, _ := filepath.Abs(*configPath)

	var err error
	query := " "
	ui := &input.UI{
		Writer: os.Stdout,
		Reader: os.Stdin,
	}

	query = "What is the database name?"
	config.DatabaseName, err = ui.Ask(query, &input.Options{
		Default:  "kereru",
		Required: true,
		Loop:     true,
	})
	if err != nil {
		log.Fatal(err)
	}

	query = "What is the database user?"
	config.DatabaseUser, err = ui.Ask(query, &input.Options{
		Default:  "twitter",
		Required: true,
		Loop:     true,
	})
	if err != nil {
		log.Fatal(err)
	}

	query = "What is the database password?"
	config.DatabasePassword, err = ui.Ask(query, &input.Options{
		Required: true,
		Loop:     true,
		Mask:     true,
	})
	if err != nil {
		log.Fatal(err)
	}

	query = "What is the hostname?"
	config.WebHost, err = ui.Ask(query, &input.Options{
		Default:  "localhost",
		Required: true,
		Loop:     true,
	})
	if err != nil {
		log.Fatal(err)
	}

	query = "What is webserver port number?"
	config.WebPort, err = ui.Ask(query, &input.Options{
		Default:  "8080",
		Required: true,
		Loop:     true,
	})
	if err != nil {
		log.Fatal(err)
	}

	query = "Should webserver use SSL/TLS?"
	useTLS, err := ui.Ask(query, &input.Options{
		Default:  "N",
		Required: true,
		Loop:     true,
	})
	if err != nil {
		log.Fatal(err)
	}
	if (useTLS == "Y") || (useTLS == "Yes") {
		config.TLS = true
	} else {
		config.TLS = false
	}

	if config.TLS == true {
		config.SecureCookie = true
		query = "What is full path to the TLS Certificate?"
		config.Cert, err = ui.Ask(query, &input.Options{
			Default:  "/etc/kereru/cert/https-server.crt",
			Required: true,
			Loop:     true,
		})
		if err != nil {
			log.Fatal(err)
		}

		query = "What is full path to the TLS Key?"
		config.Key, err = ui.Ask(query, &input.Options{
			Default:  "/etc/kereru/cert/https-server.key",
			Required: true,
			Loop:     true,
		})
		if err != nil {
			log.Fatal(err)
		}
	}

	query = "What is full path for Web Root?"
	config.WebRoot, err = ui.Ask(query, &input.Options{
		Default:  "/usr/share/kereru",
		Required: true,
		Loop:     true,
	})
	if err != nil {
		log.Fatal(err)
	}

	query = "What is full path for media uploads?"
	config.UploadPath, err = ui.Ask(query, &input.Options{
		Default:  "/var/kereru/uploads",
		Required: true,
		Loop:     true,
	})
	if err != nil {
		log.Fatal(err)
	}

	query = "What is Twitter API Consumer Key?"
	config.OauthConsumerKey, err = ui.Ask(query, &input.Options{
		Required: true,
		Loop:     true,
	})
	if err != nil {
		log.Fatal(err)
	}

	query = "What is Twitter API Consumer Secret?"
	config.OauthConsumerSecret, err = ui.Ask(query, &input.Options{
		Required: true,
		Loop:     true,
	})
	if err != nil {
		log.Fatal(err)
	}

	query = "What is Twitter API Token?"
	config.OauthToken, err = ui.Ask(query, &input.Options{
		Required: true,
		Loop:     true,
	})
	if err != nil {
		log.Fatal(err)
	}

	query = "What is Twitter API Token Secret?"
	config.OauthTokenSecret, err = ui.Ask(query, &input.Options{
		Required: true,
		Loop:     true,
	})
	if err != nil {
		log.Fatal(err)
	}

	token := make([]byte, 32)
	rand.Read(token)
	config.CsrfToken = fmt.Sprintf("%x", token[:])

	config.DebugLevel = 1
	config.Delay = 0

	YAMLData, err := yaml.Marshal(&config)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	err = ioutil.WriteFile(ConfigFilename, YAMLData, 0640)
}
