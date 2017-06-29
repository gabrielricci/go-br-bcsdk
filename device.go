package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/currantlabs/ble"
	"github.com/howeyc/crc16"
)

const (
	ByteNull = 0x00
	ByteAck  = 0x06
	ByteNak  = 0x15
	ByteSyn  = 0x16
	ByteEtb  = 0x17

	CRCPolynom = 0x1021
)

type MPOSDevice struct {
	writerCharacteristic *ble.Characteristic
	readerCharacteristic *ble.Characteristic

	bleClient ble.Client

	responseChannel chan []byte
}

func bufferResponseFromDevice(input *chan []byte, output *chan []byte) {
	var buf bytes.Buffer
	var crcLeft int

	for { // needs to read from channel ad eternum
		data := <-*input
		for _, b := range data {
			buf.WriteByte(b)

			if crcLeft > 0 {
				crcLeft--
				if crcLeft == 0 {
					*output <- buf.Bytes()
					buf.Reset()
				}

				continue
			}

			if b == ByteNak {
				*output <- buf.Bytes()
				buf.Reset()
				continue
			}

			if b == ByteEtb {
				crcLeft = 2
			}
		}
	}

	fmt.Println("exiting")
}

func NewMPOSDevice(cln ble.Client) *MPOSDevice {
	p, err := cln.DiscoverProfile(true)
	if err != nil {
		log.Fatalf("can't discover profile: %s", err)
	}

	var reader *ble.Characteristic
	var writer *ble.Characteristic

	for _, s := range p.Services {
		if strings.ToUpper(s.UUID.String()) != strings.ToUpper(serviceUuid) {
			continue
		}

		for _, c := range s.Characteristics {
			if (c.Property & ble.CharWrite) != 0 {
				writer = c
			} else {
				reader = c
			}
		}
	}

	rawResponseChannel := make(chan []byte)
	bufferedResponseChannel := make(chan []byte)

	notificationHandler := func(req []byte) {
		rawResponseChannel <- req
	}

	go bufferResponseFromDevice(&rawResponseChannel, &bufferedResponseChannel)

	if err := cln.Subscribe(reader, true, notificationHandler); err != nil {
		fmt.Printf("Failed to subscribe characteristic, err: %s\n", err)
	}

	device := MPOSDevice{
		writerCharacteristic: writer,
		readerCharacteristic: reader,
		bleClient:            cln,
		responseChannel:      bufferedResponseChannel,
	}

	return &device
}

func (device *MPOSDevice) CallMethod(methodName string, params []string) byte {
	var request bytes.Buffer
	//var response bytes.Buffer

	request.WriteByte(ByteSyn)
	request.Write([]byte(methodName))

	for idx, param := range params {
		paddedParamSize := PadLeft(len(param), 3, "0")
		fmt.Println("Param", idx, "has", paddedParamSize, "bytes (padded)")
		request.Write([]byte(paddedParamSize))
		request.Write([]byte(param))
	}

	request.WriteByte(ByteEtb)

	crc := crc16.Checksum(request.Bytes()[1:], crc16.MakeBitsReversedTable(CRCPolynom))
	crcBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(crcBytes, crc)
	request.Write(crcBytes)

	fmt.Printf("sent % X | %q\n", request.Bytes(), request.Bytes())

	device.bleClient.WriteCharacteristic(device.writerCharacteristic, request.Bytes(), true)

	select {
	case bArray := <-device.responseChannel:
		fmt.Printf("got % X | %q\n", bArray, bArray)
		fmt.Println(device.parseResponse(methodName, bArray))

	case <-time.After(60 * time.Second):
		fmt.Println("Method timeout")
		break
	}

	return ByteNull
}

func (device *MPOSDevice) parseResponse(name string, rawResponse []byte) *CommandResponse {
	var params []string

	response := CommandResponse{
		CommandName:  name,
		Acknowledged: false,
	}

	if rawResponse[0] == ByteAck && rawResponse[1] == ByteSyn {
		response.Acknowledged = true
	} else {
		return &response
	}

	rawResponse = rawResponse[2:]
	if name != string(rawResponse[0:3]) {
		response.Acknowledged = false
		return &response
	}

	rawResponse = rawResponse[3:]
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
