package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"

	"github.com/itchyny/gojq"
	"github.com/urfave/cli"
	"golang.org/x/crypto/ssh/terminal"
)

func main() {
	app := cli.NewApp()
	app.Usage = "COVID19 Victorian Exposure Sites"
	app.Commands = []cli.Command{
		{
			Name:  "config",
			Usage: "Seed state",
			Action: func(c *cli.Context) error {
				config()
				return nil
			},
		},
		{
			Name:  "check",
			Usage: "Run a check now",
			Action: func(c *cli.Context) error {
				check()
				return nil
			},
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func prompt(title string, value string, stdin io.Reader) string {
	if stdin == nil {
		stdin = io.Reader(os.Stdin)
	}
	scanner := bufio.NewScanner(stdin)
	p := title
	if value != "" {
		p = fmt.Sprintf("%s (%s)", title, value)
	}
	if terminal.IsTerminal(int(os.Stdin.Fd())) {
		fmt.Printf("%s:", p)
	}
	scanner.Scan()
	t := scanner.Text()
	if t == "" {
		return value
	}
	return t
}

func config() (state State) {

	if os.Getenv("TOKEN") != "" {
		state.Get()
	} else {
		os.Setenv("TOKEN", prompt("Meeiot TOKEN", "", nil))
	}

	state.Sid = prompt("Twilio SID", state.Sid, nil)
	state.Token = prompt("Twilio Token", state.Token, nil)
	state.Mobile = prompt("Mobile", state.Mobile, nil)
	state.Suburb = prompt("Suburb", state.Suburb, nil)
	state.Previous = []string{}

	state.Save()
	return
}

func check() (state State) {
	err := state.Get()
	if err != nil {
		log.Fatalf("Unable to load state: %s", err.Error())
	}

	fmt.Printf("STATE: %v", state)

	raw, err := latest_data(state)
	if err != nil {
		log.Fatalf("Unable to retreive new data: %s", err.Error())
	}

	data := filter(raw)
	added, removed := validate(data, state)
	if len(added) > 0 || len(removed) > 0 {
		message := ""
		if len(added) > 0 {
			message = "New exposure sites:"
			for _, n := range added {
				message = fmt.Sprintf("%s\n%s (%s)\n", message, n, "https://www.google.com/maps/place/"+url.QueryEscape(n))
				log.Printf("NEW: %s", n)
			}
		}

		if len(removed) > 0 {
			message = message + "\nRemoved exposure sites:"
			for _, n := range removed {
				message = fmt.Sprintf("%s\n%s\n", message, n)
				log.Printf("REMOVED: %s", n)
			}
		}

		state.Previous = data
		state.Save()
		notify(message, state)
	}
	return
}

func filter(jsondata []byte) []string {
	var data interface{}
	processed := []string{}
	json.Unmarshal(jsondata, &data)
	query, _ := gojq.Parse(`[.result.records[]|[.Site_streetaddress,.Suburb,.Site_state,.Site_postcode]|join(" ")]|unique`)
	iter := query.Run(data)
	filtered, _ := iter.Next()
	for _, v := range filtered.([]interface{}) {
		processed = append(processed, fmt.Sprintf("%v", v))
	}
	log.Printf("Current exposure sites: %d", len(processed))
	return processed
}

func validate(current []string, state State) ([]string, []string) {
	added := []string{}
	removed := []string{}

	for _, c := range current {
		found := false
		for _, p := range state.Previous {
			if c == p {
				found = true
				break
			}
		}
		if !found {
			added = append(added, c)
		}
	}

	for _, p := range state.Previous {
		found := false
		for _, c := range current {
			if c == p {
				found = true
				break
			}
		}
		if !found {
			removed = append(removed, p)
		}
	}

	return added, removed
}
