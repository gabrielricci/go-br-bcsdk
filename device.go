package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
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

type notifreceiver func(*CommandResponse)

type MPOSDevice struct {
	bleClient            ble.Client
	readerCharacteristic *ble.Characteristic
	writerCharacteristic *ble.Characteristic

	responseChannel      chan *CommandResponse
	notificationChannel  chan *CommandResponse
	notificationHandlers []notifreceiver
}

func bufferResponseFromDevice(input chan []byte,
	responseChannel chan *CommandResponse,
	notificationChannel chan *CommandResponse,
) {
	var buf bytes.Buffer
	var crcLeft int

	for { // needs to read from channel ad eternum
		data := <-input
		for _, b := range data {
			buf.WriteByte(b)

			if crcLeft > 0 {
				crcLeft--
				if crcLeft == 0 {
					// fmt.Printf("got % X | %q\n", buf.Bytes(), buf.Bytes())
					response := NewCommandResponse(buf.Bytes())

					if response.CommandName == "NTM" {
						notificationChannel <- response
					} else {
						responseChannel <- response
					}

					buf.Reset()
				}

				continue
			}

			if b == ByteAck {
				buf.Reset()
				continue
			}

			if b == ByteNak {
				responseChannel <- &CommandResponse{Acknowledged: false}
				buf.Reset()
				continue
			}

			if b == ByteEtb {
				crcLeft = 2
			}
		}
	}
}

func notifySubscribers(input chan *CommandResponse, subscribers *[]notifreceiver) {
	for {
		command := <-input
		if command.CommandName != "NTM" {
			continue
		}

		for _, subscriber := range *subscribers {
			subscriber(command)
		}
	}
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
	bufferedResponseChannel := make(chan *CommandResponse)
	bufferedNotificationChannel := make(chan *CommandResponse)

	readerHandler := func(req []byte) {
		rawResponseChannel <- req
	}

	go bufferResponseFromDevice(rawResponseChannel,
		bufferedResponseChannel,
		bufferedNotificationChannel)

	if err := cln.Subscribe(reader, true, readerHandler); err != nil {
		fmt.Printf("Failed to subscribe characteristic, err: %s\n", err)
	}

	device := MPOSDevice{
		writerCharacteristic: writer,
		readerCharacteristic: reader,
		bleClient:            cln,
		responseChannel:      bufferedResponseChannel,
		notificationChannel:  bufferedNotificationChannel,
	}

	go notifySubscribers(bufferedNotificationChannel, &device.notificationHandlers)

	return &device
}

func (device *MPOSDevice) CallMethod(methodName string, params []string) *CommandResponse {
	var request bytes.Buffer
	//var response bytes.Buffer

	request.WriteByte(ByteSyn)
	request.Write([]byte(methodName))

	for _, param := range params {
		paddedParamSize := PadLeft(len(param), 3, "0")
		request.Write([]byte(paddedParamSize))
		request.Write([]byte(param))
	}

	request.WriteByte(ByteEtb)

	crc := crc16.Checksum(request.Bytes()[1:], crc16.MakeBitsReversedTable(CRCPolynom))
	crcBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(crcBytes, crc)
	request.Write(crcBytes)

	// fmt.Printf("sent % X | %q\n", request.Bytes(), request.Bytes())

	device.bleClient.WriteCharacteristic(device.writerCharacteristic, request.Bytes(), true)

	select {
	case response := <-device.responseChannel:
		return response

	case <-time.After(60 * time.Second):
		fmt.Println("Method timeout")
		break
	}

	return nil
}

func (device *MPOSDevice) SubscribeToNotifications(handler notifreceiver) {
	device.notificationHandlers = append(device.notificationHandlers, handler)
}
