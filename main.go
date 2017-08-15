package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/currantlabs/ble"
	"github.com/currantlabs/ble/darwin"
	"github.com/google/uuid"
)

const serviceUuid = "49535343fe7d4ae58fa99fafd205e455"

const scanTimeout = 15 * time.Second

var done = make(chan struct{})

func main() {
	d, err := darwin.NewDevice()
	if err != nil {
		log.Fatalf("Failed to create a BLE client: %s\n", err)
		return
	}

	ble.SetDefaultDevice(d)

	filter := func(a ble.Advertisement) bool {
		return strings.ToUpper(a.LocalName()) == strings.ToUpper("PAX-77253402")
	}

	fmt.Printf("Searching for the device (timeout: %s)\n", scanTimeout)
	ctx := ble.WithSigHandler(context.WithTimeout(context.Background(), scanTimeout))
	cln, err := ble.Connect(ctx, filter)
	if err != nil {
		log.Fatalf("can't connect : %s", err)
	}

	done := make(chan struct{})

	connectedDevice := NewMPOSDevice(cln)

	fmt.Println("Device ready!")

	notificationHandler := func(notification *CommandResponse) {
		if strings.Contains(notification.Parameters[0], "SELECTED") {
			fmt.Println("Please select credit or debit in the device...")
		}
	}

	connectedDevice.SubscribeToNotifications(notificationHandler)

	connectedDevice.CallMethod("OPN", []string{})

	input := ""
	fmt.Println("Do you want to try out this thing and place a transaction? (yes/no)")
	fmt.Scanln(&input)

	if input == "yes" {
		rawAmount := ""
		fmt.Println("Lets get to it then. Please enter the transaction amount: ")
		fmt.Scanln(&rawAmount)
		amount, _ := strconv.ParseFloat(rawAmount, 32)

		fmt.Println("Now please insert your card in the reader...")

		_, tableTimestamp := CallGetTimestamp(connectedDevice)
		_, cardData := CallGetCard(connectedDevice, float32(amount), tableTimestamp)

		cvv := ""
		fmt.Println("Please enter your CVV: ")
		fmt.Scanln(&cvv)

		expiryDate, _ := time.Parse(
			"060102",
			cardData["applicationExpirationDate"],
		)

		xmlRequest, _ := CreateAuthorisationRequestXML(
			&AuthorisationRequestMessage{
				PAN:            cardData["pan"],
				CVV:            cvv,
				Track2:         cardData["track2"],
				ExpiryDate:     expiryDate.Format("2006-01"),
				EncryptedPAN:   "69c74ee0ff66a7fc",     // TODO: change that
				PANKSN:         "FFFFED1AD0000060000B", // TODO: change that,
				CardholderName: cardData["cardholderName"],
				InitiatorTxId:  uuid.New().String(),
				Amount:         float32(amount),
			},
		)

		fmt.Println("Processing...")
		CallDisplay(connectedDevice, "Processing...")

		response, _ := PostRequest(xmlRequest)

		if response.Response == SumUpResponseApproved {
			fmt.Println("Congrats, transaction approved!")
			CallDisplay(connectedDevice, "Approved")
		} else {
			fmt.Println("Oh crap, seems like you don't have enough money!!")
			CallDisplay(connectedDevice, "Declined: "+response.ResponseReason)
		}

		time.Sleep(2 * time.Second)

		fmt.Println("You can have your card back now")
		CallDisplay(connectedDevice, "Remove your card")

		time.Sleep(10 * time.Second)
	}

	connectedDevice.CallMethod("CLO", []string{"SumUp"})

	// Disconnect the connection. (On OS X, this might take a while.)
	fmt.Printf("Disconnecting [ %s ]... (this might take up to few seconds on OS X)\n", cln.Address())
	cln.CancelConnection()

	<-done
}
