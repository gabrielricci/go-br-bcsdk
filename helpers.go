package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
)

func PadLeft(input, length int, padding string) string {
	return fmt.Sprintf("%"+padding+strconv.Itoa(length)+"d", input)
}

func PostRequest(requestBody []byte) (*AuthorisationResponseMessage, error) {
	bodyReader := bytes.NewReader(requestBody)

	client := &http.Client{}
	req, _ := http.NewRequest("POST", "https://txg-beta.sam-app.ro/v1/cloudwalk-payments/", bodyReader)
	// req, _ := http.NewRequest("POST", "http://localhost:8080/v1/cloudwalk-payments/", bodyReader)
	req.Header.Set("Content-Type", "text/xml")
	req.Header.Set("Serial-Number", "1234567890123456")

	res, err := client.Do(req)
	if err != nil {
		return &AuthorisationResponseMessage{}, err
	}

	if res.StatusCode != 200 {
		return &AuthorisationResponseMessage{}, errors.New("Invalid http responde code")
	}

	defer res.Body.Close()

	responseBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return &AuthorisationResponseMessage{}, err
	}

	doc, _ := NewXML(responseBody)
	// doc.WriteTo(os.Stdout)

	responseMessage := AuthorisationResponseMessage{
		Response:       doc.FindElements("//Rspn")[0].Text(),
		ResponseReason: doc.FindElements("//RspnRsn")[0].Text(),
	}

	return &responseMessage, nil
}
