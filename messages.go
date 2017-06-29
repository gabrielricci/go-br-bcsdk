package main

type AuthorisationRequestMessage struct {
	PAN            string
	CVV            string
	Track2         string
	ExpiryDate     string
	EncryptedPAN   string
	PANKSN         string
	CardholderName string
	InitiatorTxId  string
	Amount         float32
}

type AuthorisationResponseMessage struct {
	Response       string
	ResponseReason string
}
