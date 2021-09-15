package main

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/jmespath/go-jmespath"
	"github.com/urfave/cli"
)

type State struct {
	Hash   string `json:"hash"`
	Sid    string `json:"sid"`
	Token  string `json:"token"`
	Mobile string `json:"mobile"`
	Suburb string `json:"suburb"`
}

func main() {
	app := cli.NewApp()
	app.Commands = []cli.Command{
		{
			Name:  "config",
			Usage: "add a task to the list",
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

func config() {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("meeiot Token: ")
	scanner.Scan()
	os.Setenv("TOKEN", scanner.Text())

	state := State{}

	fmt.Print("Twilio SID: ")
	scanner.Scan()
	state.Sid = scanner.Text()

	fmt.Print("Twilio Token: ")
	scanner.Scan()
	state.Token = scanner.Text()

	fmt.Print("Mobile: ")
	scanner.Scan()
	state.Mobile = scanner.Text()

	fmt.Print("Suburb to check: ")
	scanner.Scan()
	state.Suburb = scanner.Text()
	state.Hash = "0"

	save_state(state)
}

func check() {
	data := lastest_data()
	state := get_state()
	generated := hash(data, state)
	if generated != state.Hash {
		state.Hash = generated
		log.Printf("New exposure site in %s (%t)\n", state.Suburb, save_state(state))
		notify(state)
	}
}

func hash(jsondata []byte, state State) string {
	var data interface{}
	json.Unmarshal(jsondata, &data)
	query := fmt.Sprintf("result.records[?Suburb==`\"%s\"`]", state.Suburb)
	result, _ := jmespath.Search(query, data)
	j, _ := json.Marshal(result)
	hash := fmt.Sprintf("%x", md5.Sum(j))
	log.Printf("Generated hash: %s\n", hash)
	return hash
}

func lastest_data() []byte {
	resp, err := http.Get("https://discover.data.vic.gov.au/api/3/action/datastore_search?resource_id=afb52611-6061-4a2b-9110-74c920bede77")
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

	enc, _ := ioutil.ReadAll(resp.Body)

	dec, _ := base64.StdEncoding.DecodeString(string(enc))

	json.Unmarshal(dec, &state)
	log.Printf("State Loaded: hash=%s suburb=%s", state.Hash, state.Suburb)
	return state
}

func save_state(state State) bool {
	b, _ := json.Marshal(state)
	value := base64.URLEncoding.EncodeToString(b)
	url := "https://www.meeiot.org/put/" + os.Getenv("TOKEN") + "/state=" + value
	resp, err := http.Post(url, "application/json", bytes.NewBuffer([]byte{0}))
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	data, _ := ioutil.ReadAll(resp.Body)
	if string(data) != "0:0" {
		log.Fatal(string(data))
	}
	var result bool
	json.Unmarshal(data, &result)
	return result
}

func notify(state State) string {

	endpoint := "https://api.twilio.com/2010-04-01/Accounts/" + state.Sid + "/Messages"

	params := url.Values{}
	params.Set("From", "exposure")
	params.Set("To", state.Mobile)
	params.Set("Body", "New exposure sites for "+state.Suburb)

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
