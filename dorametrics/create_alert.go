package dorametrics

import (
	"bytes"
	"encoding/json"
	"net/http"
)

const pagerDutyUrl = "https://api.pagerduty.com/incidents"

type Payload struct {
	Incident Incident `json:"incident"`
}
type Service struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}
type Incident struct {
	Type    string  `json:"type"`
	Title   string  `json:"title"`
	Service Service `json:"service"`
}

func createAlert(service string, namespace string, stage string, username string, token string) error {
	data := Payload{}
	payloadBytes, err := json.Marshal(data)
	if err != nil {
		// TODO: handle error
	}
	body := bytes.NewReader(payloadBytes)

	req, err := http.NewRequest("POST", pagerDutyUrl, body)
	if err != nil {
		// TODO: handle error
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.pagerduty+json;version=2")
	req.Header.Set("From", username)
	req.Header.Set("Authorization", "Token token="+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// TODO: handle error
	}
	defer resp.Body.Close()

	return nil
}
