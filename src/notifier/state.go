package main

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

type State struct {
	Sid      string   `json:"sid"`
	Token    string   `json:"token"`
	Mobile   string   `json:"mobile"`
	Suburb   string   `json:"suburb"`
	Previous []string `json:"previous"`
}

func (s *State) Get() (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()

	url := fmt.Sprintf("https://www.meeiot.org/get/%s/state", os.Getenv("TOKEN"))
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	rawb64, _ := ioutil.ReadAll(resp.Body)
	rawb64 = []byte(strings.Replace(string(rawb64), ".", "=", -1))

	if string(rawb64) == "52:invalid user token\n" {
		return errors.New("Invalid meeiot user token")
	}

	raw, err := base64.URLEncoding.DecodeString(string(rawb64))

	if err != nil {
		return err
	}

	br := bytes.NewReader(raw)
	gz, _ := gzip.NewReader(br)
	data, err := ioutil.ReadAll(gz)

	if err != nil {
		return err
	}

	json.Unmarshal(data, &s)
	log.Printf("State Loaded: previous exposure sites: %d\n", len(s.Previous))
	return nil
}

func (s *State) Save() (err error) {
	j, _ := json.Marshal(s)
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	gz.Write(j)
	gz.Flush()
	gz.Close()
	value := base64.URLEncoding.EncodeToString(b.Bytes())
	url := "https://www.meeiot.org/put/" + os.Getenv("TOKEN") + "/state=" + strings.Replace(value, "=", ".", -1)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	raw, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != 200 || string(raw) != "0:0" {
		err = errors.New(string(raw))
		return err
	}

	log.Printf("State saved.")
	return err
}
