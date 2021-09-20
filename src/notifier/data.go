package main

import (
	"io/ioutil"
	"log"
	"net/http"
)

func latest_data(state State) (data []byte, err error) {
	resp, err := http.Get("https://discover.data.vic.gov.au/api/3/action/datastore_search?resource_id=afb52611-6061-4a2b-9110-74c920bede77&q=" + state.Suburb)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	data, err = ioutil.ReadAll(resp.Body)
	log.Println("Retrieved new exposure data")
	return
}
