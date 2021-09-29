package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"reflect"
	"testing"

	"bou.ke/monkey"
	"github.com/jarcoal/httpmock"
)

func TestMain(m *testing.M) {
	log.SetOutput(ioutil.Discard)
	//fmt.SetOutput(ioutil.Discard)
	os.Exit(m.Run())
}

func TestValidate(t *testing.T) {
	prev := State{Previous: []string{}}
	current := []string{"New"}

	add, removed := validate(current, prev)

	if len(add) != 1 || len(removed) != 0 {
		t.Errorf("New address not detected")
	}

	current = []string{}
	prev = State{Previous: []string{"Removed"}}

	add, removed = validate(current, prev)

	if len(add) != 0 || len(removed) != 1 {
		t.Errorf("Removed address not detected")
	}
}

func TestGetState(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "https://www.meeiot.org/get/incorrectjson/state",
		httpmock.NewStringResponder(200, `H4sIACD9R2EAA6uuBQBDv6ajAgAAAA..`))

	httpmock.RegisterResponder("GET", "https://www.meeiot.org/get/mocked/state",
		httpmock.NewStringResponder(200, `H4sIAID_R2EAA6tWKs5MUbICkzpKJfnZqXlAHoTWUcrNT8rMSQUKQBk6SsWlSaVFSSANEIaOUkFRallmfmmxklW0UmJKSlFqcbFSbC0AYGque1gAAAA.`))

	httpmock.RegisterResponder("GET", "https://www.meeiot.org/get/invalid/state",
		httpmock.NewStringResponder(200, `aGkK`))

	os.Setenv("TOKEN", "mocked")
	s := State{}
	err := s.Get()
	if err != nil {
		t.Errorf("Error: %v", err)
	}

	if s.Mobile != "mobile" || len(s.Previous) != 1 || s.Previous[0] != "address" || s.Sid != "sid" || s.Token != "token" {
		t.Errorf("Failed to load state correctly")
	}

	os.Setenv("TOKEN", "invalid")
	err = s.Get()
	if err == nil || err.Error() != "runtime error: invalid memory address or nil pointer dereference" {
		t.Error("Expected a panic")
	}
}

func TestSaveState(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "https://www.meeiot.org/put/invalid/state=H4sIAAAAAAAA_6pWKs5MUbJSUtJRKsnPTs2DMHPzkzJzUiHs4tKk0qIkCLugKLUsM7-0WMkqrzQnpxYAAAD__wEAAP__tV73GD0AAAA.",
		httpmock.NewStringResponder(200, "52:invalid user token"))

	httpmock.RegisterResponder("GET", "https://www.meeiot.org/put/mocked/state=H4sIAAAAAAAA_6pWKs5MUbJSUtJRKsnPTs2DMHPzkzJzUiHs4tKk0qIkCLugKLUsM7-0WMkqrzQnpxYAAAD__wEAAP__tV73GD0AAAA.",
		httpmock.NewStringResponder(200, "0:0"))

	os.Setenv("TOKEN", "invalid")
	s := State{}
	if err := s.Save(); err == nil || err.Error() != "52:invalid user token" {
		t.Error("Expected '52:invalid user token' error")
	}

	os.Setenv("TOKEN", "mocked")
	if err := s.Save(); err != nil {
		t.Error("Expected '0:0' response")
	}
}
func TestLatestData(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "https://discover.data.vic.gov.au/api/3/action/datastore_search?resource_id=afb52611-6061-4a2b-9110-74c920bede77&q=mock",
		httpmock.NewStringResponder(200, "{}"))

	s := State{Suburb: "mock"}
	data, err := latest_data(s)
	if err != nil {
		t.Error(err)
	}

	if string(data) != "{}" {
		t.Errorf("Incorrect data expecetd '{}' got '%s'", string(data))
	}
}

func TestNotify(t *testing.T) {
	success := "<?xml version='1.0' encoding='UTF-8'?>\n<TwilioResponse><Message><Sid>SMb53f77373a234fbea76e4c4c3b69535f</Sid><DateCreated>Mon, 20 Sep 2021 10:23:34 +0000</DateCreated><DateUpdated>Mon, 20 Sep 2021 10:23:34 +0000</DateUpdated><DateSent/><AccountSid>ACd4cecded1a6fa12370affbff1626bd1a</AccountSid><To>+61400000000</To><From>exposure</From><MessagingServiceSid/><Body>mock_message</Body><Status>queued</Status><NumSegments>1</NumSegments><NumMedia>0</NumMedia><Direction>outbound-api</Direction><ApiVersion>2010-04-01</ApiVersion><Price/><PriceUnit>USD</PriceUnit><ErrorCode/><ErrorMessage/><Uri>/2010-04-01/Accounts/ACd4cecded1a6fa12370affbff1626bd1a/Messages/SMb53f77373a234fbea76e4c4c3b69535f</Uri><SubresourceUris><Media>/2010-04-01/Accounts/ACd4cecded1a6fa12370affbff1626bd1a/Messages/SMb53f77373a234fbea76e4c4c3b69535f/Media</Media></SubresourceUris></Message></TwilioResponse>"
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("POST", "https://api.twilio.com/2010-04-01/Accounts/sid/Messages",
		func(req *http.Request) (res *http.Response, err error) {
			raw, _ := ioutil.ReadAll(req.Body)
			auth := req.Header.Get("Authorization")
			if auth != "Basic c2lkOnRva2Vu" {
				return httpmock.NewStringResponse(401, "<?xml version='1.0' encoding='UTF-8'?>\n<TwilioResponse><RestException><Code>20003</Code><Detail>Your AccountSid or AuthToken was incorrect.</Detail><Message>Authentication Error - invalid username</Message><MoreInfo>https://www.twilio.com/docs/errors/20003</MoreInfo><Status>401</Status></RestException></TwilioResponse>"), nil
			}
			if string(raw) != "Body=mock_message&From=exposure&To=%2B61400000000" {
				return httpmock.NewStringResponse(400, "<?xml version='1.0' encoding='UTF-8'?>\n<TwilioResponse><RestException><Code>21211</Code><Message>The 'To' number 261400000000 is not a valid phone number.</Message><MoreInfo>https://www.twilio.com/docs/errors/21211</MoreInfo><Status>400</Status></RestException></TwilioResponse>"), nil
			}
			return httpmock.NewStringResponse(201, success), nil
		})

	s := State{Token: "token", Sid: "sid", Mobile: "+61400000000"}
	rep, err := notify("mock_message", s)
	if err != nil {
		t.Errorf("TestNotify.err: %s", err.Error())
	}

	if rep != success {
		t.Errorf("TestNotify.rep want '%s' got '%s'", success, rep)
	}
}

func TestFilter(t *testing.T) {
	payload := `{"help": "https://discover.data.vic.gov.au/api/3/action/help_show?name=datastore_search", "success": true, "result": {"include_total": true, "resource_id": "afb52611-6061-4a2b-9110-74c920bede77", "fields": [{"type": "int", "id": "_id"}, {"type": "text", "id": "Suburb"}, {"type": "text", "id": "Site_title"}, {"type": "text", "id": "Site_streetaddress"}, {"type": "text", "id": "Site_state"}, {"type": "text", "id": "Site_postcode"}, {"type": "text", "id": "Exposure_date_dtm"}, {"type": "text", "id": "Exposure_date"}, {"type": "text", "id": "Exposure_time"}, {"type": "text", "id": "Notes"}, {"type": "text", "id": "Added_date_dtm"}, {"type": "text", "id": "Added_date"}, {"type": "text", "id": "Added_time"}, {"type": "text", "id": "Advice_title"}, {"type": "text", "id": "Advice_instruction"}, {"type": "text", "id": "Exposure_time_start_24"}, {"type": "text", "id": "Exposure_time_end_24"}, {"type": "text", "id": "dhid"}], "records_format": "objects", "q": "Doveton", "records": [{"_id":219,"Suburb":"Ballarat","Site_title":"Target Ballarat","Site_streetaddress":"5 Doveton Street","Site_state":"VIC","Site_postcode":"3350","Exposure_date_dtm":"2021-09-12","Exposure_date":"12/09/2021","Exposure_time":"12:55pm - 2:35pm","Notes":"Case attended venue","Added_date_dtm":"2021-09-16","Added_date":"16/09/2021","Added_time":"22:25:00","Advice_title":"Tier 2 - Get tested urgently and isolate until you have a negative result","Advice_instruction":"Anyone who has visited this location during these times should urgently get tested, then isolate until confirmation of a negative result. Continue to monitor for symptoms, get tested again if symptoms appear.","Exposure_time_start_24":"12:55:00","Exposure_time_end_24":"14:35:00","dhid":"B6G5","rank":0.0573088},{"_id":242,"Suburb":"Ballarat","Site_title":"New Generation Clothing Ballarat","Site_streetaddress":"26 Doveton Street South","Site_state":"VIC","Site_postcode":"3350","Exposure_date_dtm":"2021-09-12","Exposure_date":"12/09/2021","Exposure_time":"12:30pm - 1:20pm","Notes":"Case attended venue","Added_date_dtm":"2021-09-16","Added_date":"16/09/2021","Added_time":"19:07:00","Advice_title":"Tier 2 - Get tested urgently and isolate until you have a negative result","Advice_instruction":"Anyone who has visited this location during these times should urgently get tested, then isolate until confirmation of a negative result. Continue to monitor for symptoms, get tested again if symptoms appear.","Exposure_time_start_24":"12:30:00","Exposure_time_end_24":"13:20:00","dhid":"D9U4","rank":0.0573088}], "_links": {"start": "/api/3/action/datastore_search?q=Doveton&resource_id=afb52611-6061-4a2b-9110-74c920bede77", "next": "/api/3/action/datastore_search?q=Doveton&offset=100&resource_id=afb52611-6061-4a2b-9110-74c920bede77"}, "total": 2}}`
	result := filter([]byte(payload))

	if len(result) != 2 {
		t.Errorf("TestFilter.result.len: expected '%d' not '%d", 2, len(result))
	}

	if result[1] != "5 Doveton Street Ballarat VIC 3350" {
		t.Errorf("TestFilter.result[1]: expected '%s' no '%s'", "5 Doveton Street Ballarat VIC 3350", result[1])
	}
}

func TestPrompt(t *testing.T) {
	result := prompt("mock", "mockvalue", nil)
	if result != "mockvalue" {
		t.Errorf("Expected '%s' not '%s'", "mockvalue", result)
	}

	var stdin bytes.Buffer
	stdin.Write([]byte("mocksetvalue\n"))

	result = prompt("mock", "mockvalue", &stdin)
	if result != "mocksetvalue" {
		t.Errorf("Expected '%s' not '%s'", "mocksetvalue", result)
	}
}

func TestConfig(t *testing.T) {
	origfunc := prompt
	defer monkey.Patch(prompt, origfunc)

	monkey.Patch(prompt, func(t string, v string, i io.Reader) string {
		return "mockvalue"
	})

	expected := "{mockvalue mockvalue mockvalue mockvalue []}"
	state := config()
	result := fmt.Sprintf("%v", state)
	if result != expected {
		t.Errorf("Expected '%s' not '%s'", expected, result)
	}

}

func TestCheck(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	var target *State
	monkey.PatchInstanceMethod(reflect.TypeOf(target), "Get", func(s *State) (err error) {
		s.Mobile = "mobile"
		s.Token = "token"
		s.Previous = []string{"2 Removed Road Mockville VIC 0000", "3 Existing Avenue Mockville VIC 0000"}
		s.Sid = "sid"
		s.Suburb = "suburb"
		return
	})

	monkey.Patch(latest_data, func(_ State) (b []byte, err error) {
		b = []byte(`{"result":{"records":[{"Site_streetaddress":"1 Mock Street","Site_state":"VIC","Site_postcode":"0000","Suburb":"Mockville"},{"Site_streetaddress":"3 Existing Avenue","Site_state":"VIC","Site_postcode":"0000","Suburb":"Mockville"}]}}`)
		return
	})

	monkey.Patch(notify, func(m string, s State) (string, error) {
		expected := "New exposure sites:\n1 Mock Street Mockville VIC 0000 (https://www.google.com/maps/place/1+Mock+Street+Mockville+VIC+0000)\n\nRemoved exposure sites:\n2 Removed Road Mockville VIC 0000"

		if m != expected {
			t.Errorf("Expected '%s' not '%s'", expected, m)
		}
		return "", nil
	})

	exp_state := "{sid token mobile suburb [1 Mock Street Mockville VIC 0000 3 Existing Avenue Mockville VIC 0000]}"
	state := check()
	result := fmt.Sprintf("%v", state)
	if result != exp_state {
		t.Errorf("Expected '%s' not '%s'", exp_state, result)
	}

}
