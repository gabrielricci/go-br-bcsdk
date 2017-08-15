package main

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

type CommandResponse struct {
	CommandName  string
	Acknowledged bool
	ResponseCode string
	Parameters   []string
}

func NewCommandResponse(rawResponse []byte) *CommandResponse {
	var params []string

	response := CommandResponse{
		Acknowledged: false,
	}

	if rawResponse[0] == ByteSyn {
		response.Acknowledged = true
	} else {
		return &response
	}

	response.CommandName = string(rawResponse[1:4])

	rawResponse = rawResponse[4:]
	response.ResponseCode = string(rawResponse[0:3])

	rawResponse = rawResponse[3:]
	for rawResponse[0] != ByteEtb {
		paramLen, _ := strconv.Atoi(string(rawResponse[0:3]))
		endPos := 3 + paramLen
		params = append(params, string(rawResponse[3:endPos]))
		rawResponse = rawResponse[endPos:]
	}

	if rawResponse[0] == ByteEtb {
		response.Parameters = params
	}

	return &response
}

func CallEncryptBuffer(device *MPOSDevice) (string, error) {
	response := device.CallMethod("ENB", []string{
		"3230000000000000000000000000000000012345678901234567891",
	})
	if response.ResponseCode != "000" {
		return "", errors.New("Invalid response received")
	}

	return response.Parameters[0], nil
}

func CallGetDUKPT(device *MPOSDevice) (string, error) {
	response := device.CallMethod("GDU", []string{"323"})
	if response.ResponseCode != "000" {
		return "", errors.New("Invalid response received")
	}

	return response.Parameters[0], nil
}

func CallGetInfo(device *MPOSDevice, acquirerCode string) (string, error) {
	response := device.CallMethod("GIN", []string{acquirerCode})
	if response.ResponseCode != "000" {
		return "", errors.New("Invalid response received")
	}

	return response.Parameters[0], nil
}

func CallDisplay(device *MPOSDevice, message string) (bool, error) {
	response := device.CallMethod("DSP", []string{message})
	if response.ResponseCode != "000" {
		return false, errors.New("Invalid response received")
	}

	return true, nil
}

func CallGetTimestamp(device *MPOSDevice) (error, string) {
	acquirerCode := "00"
	response := device.CallMethod("GTS", []string{acquirerCode})

	if response.CommandName == "GTS" {
		return nil, response.Parameters[0]
	}

	return errors.New("Invalid response received"), ""
}

func CallGetCard(device *MPOSDevice, amount float32, tableTimestamp string) (error, map[string]string) {
	cardData := make(map[string]string)

	acquirerCode := "00"
	applicationCode := "01"
	formattedAmount := PadLeft(int(amount*100), 12, "0")
	date := time.Now().Format("060102")
	time := time.Now().Format("150405")

	input := acquirerCode +
		applicationCode +
		formattedAmount +
		date +
		time +
		tableTimestamp +
		"00"

	response := device.CallMethod("GCR", []string{input})

	if response.CommandName != "GCR" || response.ResponseCode != "000" {
		return errors.New("Invalid response received"), cardData
	}

	T := func(in string) string {
		return strings.Trim(in, " ")
	}

	rawData := response.Parameters[0]
	cardData = map[string]string{
		"appPanSequenceNumber":      rawData[254:256],
		"applicationLabel":          T(rawData[256:272]),
		"serviceCode":               rawData[272:275],
		"cardholderName":            T(rawData[275:301]),
		"applicationExpirationDate": rawData[301:307],
		"cardExternalNumberLength":  rawData[307:309],
		"cardExternalNumber":        T(rawData[309:328]),
		"balance":                   rawData[328:336],
		"issuerCountryCode":         rawData[336:339],
		"cardType":                  rawData[0:2],
		"readStatus":                rawData[2:3],
		"applicationType":           rawData[3:5],
		"acquirerCode":              rawData[5:7],
		"aidIndex":                  rawData[7:9],
		"track1Length":              rawData[9:11],
		"track1":                    T(rawData[11:87]),
		"track2Length":              rawData[87:89],
		"track2":                    T(rawData[89:126]),
		"track3Length":              rawData[126:129],
		"track3":                    T(rawData[129:233]),
		"panLength":                 rawData[233:235],
		"pan":                       T(rawData[235:254]),
	}

	return nil, cardData
}
