package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/currantlabs/ble"
	"github.com/currantlabs/ble/darwin"
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
		return strings.ToUpper(a.LocalName()) == strings.ToUpper("PAX-77101763")
	}

	fmt.Printf("Scanning for %s...\n", scanTimeout)
	ctx := ble.WithSigHandler(context.WithTimeout(context.Background(), scanTimeout))
	cln, err := ble.Connect(ctx, filter)
	if err != nil {
		log.Fatalf("can't connect : %s", err)
	}

	done := make(chan struct{})

	connectedDevice := NewMPOSDevice(cln)
	connectedDevice.CallMethod("OPN", []string{})
	// connectedDevice.CallMethod("DSP", []string{"SumUp"})
	// connectedDevice.CallMethod("GTS", []string{"00"})

	connectedDevice.CallMethod("GCR", []string{"0000000000000100170629093700010620160200"})
	connectedDevice.CallMethod("CLO", []string{"SumUp"})

	time.Sleep(10 * time.Second)

	// Disconnect the connection. (On OS X, this might take a while.)
	fmt.Printf("Disconnecting [ %s ]... (this might take up to few seconds on OS X)\n", cln.Address())
	cln.CancelConnection()

	<-done
}
