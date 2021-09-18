package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/itchyny/gojq"
	"github.com/urfave/cli"
)

type State struct {
	Sid      string   `json:"sid"`
	Token    string   `json:"token"`
	Mobile   string   `json:"mobile"`
	Suburb   string   `json:"suburb"`
	Previous []string `json:"previous"`
}

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

func prompt(title string, value string) string {
	scanner := bufio.NewScanner(os.Stdin)
	p := title
	if value != "" {
		p = fmt.Sprintf("%s (%s)", title, value)
	}
	fmt.Printf("%s:", p)
	scanner.Scan()
	t := scanner.Text()
	if t == "" {
		return value
	}
	return t
}

func config() {
	state := State{}

	if os.Getenv("TOKEN") != "" {
		state = get_state()
	} else {
		os.Setenv("TOKEN", prompt("Meeiot TOKEN", ""))
	}

	state.Sid = prompt("Twilio SID", state.Sid)
	state.Token = prompt("Twilio Token", state.Token)
	state.Mobile = prompt("Mobile", state.Mobile)
	state.Suburb = prompt("Suburb", state.Suburb)
	state.Previous = []string{}

	save_state(state)
}

func check() {
	state := get_state()
	raw := lastest_data(state)
	data := filter(raw)
	added, removed := validate(data, state)
	if len(added) > 0 || len(removed) > 0 {
		message := ""
		if len(added) > 0 {
			message = "New exposure sites:"
			for _, n := range added {
				message = fmt.Sprintf("%s\n%s (%s)\n", message, n, "https://www.google.com/maps/place/"+url.QueryEscape(n))
			}
		}

		state.Previous = data
		save_state(state)
		notify(message, state)
	}
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

func lastest_data(state State) []byte {
	resp, err := http.Get("https://discover.data.vic.gov.au/api/3/action/datastore_search?resource_id=afb52611-6061-4a2b-9110-74c920bede77&q=" + state.Suburb)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	data, _ := ioutil.ReadAll(resp.Body)
	log.Println("Retrieved new exposure data")
	return data
}

func get_state() State {
	var state State
	url := fmt.Sprintf("https://www.meeiot.org/get/%s/state", os.Getenv("TOKEN"))
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	rawb64, _ := ioutil.ReadAll(resp.Body)
	raw, _ := base64.URLEncoding.DecodeString(strings.Replace(string(rawb64), ".", "=", -1))
	br := bytes.NewReader(raw)
	gz, _ := gzip.NewReader(br)
	data, _ := ioutil.ReadAll(gz)
	json.Unmarshal(data, &state)
	log.Printf("State Loaded: previous exposure sites:%d\n", len(state.Previous))
	return state
}

func save_state(state State) {
	j, _ := json.Marshal(state)
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	gz.Write(j)
	gz.Flush()
	gz.Close()
	value := base64.URLEncoding.EncodeToString(b.Bytes())
	url := "https://www.meeiot.org/put/" + os.Getenv("TOKEN") + "/state=" + strings.Replace(value, "=", ".", -1)
	resp, err := http.Post(url, "application/json", nil)
	if err != nil {
		log.Println("Failed to post")
		log.Fatal(err)
	}
	defer resp.Body.Close()
	log.Println("State saved.")
}

func notify(message string, state State) string {

	endpoint := "https://api.twilio.com/2010-04-01/Accounts/" + state.Sid + "/Messages"

	params := url.Values{}
	params.Set("From", "exposure")
	params.Set("To", state.Mobile)
	params.Set("Body", message)

	body := *strings.NewReader(params.Encode())

	client := &http.Client{}
	req, _ := http.NewRequest("POST", endpoint, &body)
	req.SetBasicAuth(state.Sid, state.Token)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	res, err := client.Do(req)

	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	data, _ := ioutil.ReadAll(res.Body)
	return string(data)
}
