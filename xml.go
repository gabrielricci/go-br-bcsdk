package main

import (
	"strconv"
	"strings"
	"time"

	"github.com/beevik/etree"
)

func NewXML(body []byte) (*etree.Document, error) {
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(body); err != nil {
		return doc, err
	}

	return doc, nil
}

func CreateAuthorisationRequestXML(message *AuthorisationRequestMessage) ([]byte, error) {
	// required data
	pan := message.PAN
	obfuscatedPAN := pan[0:6] + strings.Repeat("*", len(pan)-6)
	obfuscatedTrack2 := obfuscatedPAN + message.Track2[len(pan):]
	amountInCents := int(message.Amount * 100)
	datetime := time.Now().Format(time.RFC3339)[0:19]

	// xml generation start
	doc := etree.NewDocument()
	doc.CreateProcInst("xml", `version="1.0" encoding="UTF-8"`)

	root := doc.CreateElement("Document")
	root.CreateAttr("xmlns", "urn:AcceptorAuthorisationRequestV02.1")

	authReqRoot := root.CreateElement("AccptrAuthstnReq")

	header := authReqRoot.CreateElement("Hdr")
	msgFunction := header.CreateElement("MsgFctn")
	msgFunction.SetText("AUTQ")

	protocolVersion := header.CreateElement("PrtcolVrsn")
	protocolVersion.SetText("2.0")

	authReq := authReqRoot.CreateElement("AuthstnReq")

	environment := authReq.CreateElement("Envt")
	context := authReq.CreateElement("Cntxt")
	transaction := authReq.CreateElement("Tx")

	// Environment
	merchant := environment.CreateElement("Mrchnt").CreateElement("Id").CreateElement("Id")
	merchant.SetText("bla")

	card := environment.CreateElement("Card")
	plainCardData := card.CreateElement("PlainCardData")
	plainCardData.CreateElement("PAN").SetText(obfuscatedPAN)
	plainCardData.CreateElement("XpryDt").SetText(message.ExpiryDate)
	plainCardData.CreateElement("CardSeqNb")
	plainCardData.CreateElement("CardSctyCd").CreateElement("CSCVal").SetText(message.CVV)
	trackData := plainCardData.CreateElement("TrckData")
	trackData.CreateElement("TrckNb").SetText("2")
	trackData.CreateElement("TrckVal").SetText(obfuscatedTrack2)
	encCardData := card.CreateElement("EncryptedCardData")
	encPAN := encCardData.CreateElement("PAN")
	encPAN.CreateElement("EncryptedPAN").SetText(message.EncryptedPAN)
	encPAN.CreateElement("PANKSN").SetText(message.PANKSN)
	cardHolder := environment.CreateElement("Crdhldr")
	cardHolder.CreateElement("Nm").SetText(message.CardholderName)
	cardHolder.CreateElement("Authntcn").CreateElement("AuthntcnMtd").SetText("PPSG")

	// Context
	paymentContext := context.CreateElement("PmtCntxt")
	paymentContext.CreateElement("CardDataNtryMd").SetText("MGST")
	paymentContext.CreateElement("FllbckInd").SetText("false")

	// Transaction
	transaction.CreateElement("InitrTxId").SetText(message.InitiatorTxId)
	transaction.CreateElement("TxCaptr").SetText("true")
	transactionId := transaction.CreateElement("TxId")
	transactionId.CreateElement("TxDtTm").SetText(datetime)
	transactionId.CreateElement("TxRef").SetText(message.InitiatorTxId)
	transactionDetails := transaction.CreateElement("TxDtls")
	transactionDetails.CreateElement("Ccy").SetText("0986")
	transactionDetails.CreateElement("TtlAmt").SetText(strconv.Itoa(amountInCents))
	transactionDetails.CreateElement("AcctTp").SetText("CRDT")
	recuringTransaction := transactionDetails.CreateElement("RcrngTx")
	recuringTransaction.CreateElement("InstlmtTp").SetText("NONE")
	recuringTransaction.CreateElement("TtlNbOfPmts").SetText("0")

	doc.Indent(4)
	return doc.WriteToBytes()
}
