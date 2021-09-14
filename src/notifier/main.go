package main

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/jmespath/go-jmespath"
)

func main() {
	suburb := os.Args[1]
	data := lastest_data()
	generated := hash(data, suburb)
	saved := get_value("hash")
	log.Printf("Saved hash: %s", saved)
	if generated != saved {
		log.Printf("New exposure site in %s (%t)\n", suburb, set_value("hash", generated))
		notify(suburb)
	}
}

func hash(jsondata []byte, suburb string) string {
	var data interface{}
	json.Unmarshal(jsondata, &data)
	query := fmt.Sprintf("result.records[?Suburb==`\"%s\"`]", suburb)
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

func get_value(key string) string {
	var value string
	url := fmt.Sprintf("https://keyvalue.immanuel.co/api/KeyVal/GetValue/%s/%s", os.Getenv("TOKEN"), key)
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	data, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal(data, &value)
	return value
}

func set_value(key string, value string) bool {
	var result bool
	url := fmt.Sprintf("https://keyvalue.immanuel.co/api/KeyVal/UpdateValue/%s/%s/%s", os.Getenv("TOKEN"), key, value)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer([]byte{0}))
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	data, _ := ioutil.ReadAll(resp.Body)
	//log.Printf("SET key %s=%s", key, value)
	json.Unmarshal(data, &result)
	return result
}

func notify(suburb string) string {
	account_sid := get_value("sid")
	auth_token := get_value("token")

	endpoint := "https://api.twilio.com/2010-04-01/Accounts/" + account_sid + "/Messages"

	params := url.Values{}
	params.Set("From", "exposure")
	params.Set("To", "+61432071731")
	params.Set("Body", "New exposure sites for "+suburb)

	body := *strings.NewReader(params.Encode())

	client := &http.Client{}
	req, _ := http.NewRequest("POST", endpoint, &body)
	req.SetBasicAuth(account_sid, auth_token)
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
