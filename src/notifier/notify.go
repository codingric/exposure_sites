package main

import (
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

func notify(message string, state State) (response string, err error) {

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
		return
	}
	defer res.Body.Close()

	raw, _ := ioutil.ReadAll(res.Body)
	response = string(raw)

	if res.StatusCode != 201 {
		err = errors.New("Reponse: " + strconv.Itoa(res.StatusCode))
	}
	return
}
