package adapter

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os/exec"
	"time"

	"github.com/shopspring/decimal"

	"github.com/smpebble/actus-fvm/internal/model"
)

// ISO20022Client generates ISO 20022 messages (built-in, no Claude API).
type ISO20022Client struct {
	institutionBIC  string
	institutionName string
}

// NewISO20022Client creates a new ISO 20022 client.
func NewISO20022Client(bic, name string) *ISO20022Client {
	return &ISO20022Client{
		institutionBIC:  bic,
		institutionName: name,
	}
}

// GeneratePacs008 generates a pacs.008 FIToFICustomerCreditTransfer message.
func (c *ISO20022Client) GeneratePacs008(settlement *model.Settlement) model.ISO20022Message {
	msg := pacs008Document{
		XMLName: xml.Name{Local: "Document"},
		Xmlns:   "urn:iso:std:iso:20022:tech:xsd:pacs.008.001.10",
		FIToFICstmrCdtTrf: &fiToFICstmrCdtTrf{
			GrpHdr: groupHeader{
				MsgId:    fmt.Sprintf("PACS008-%s", settlement.ID[:8]),
				CreDtTm:  time.Now().Format("2006-01-02T15:04:05"),
				NbOfTxs:  "1",
				CtrlSum:  settlement.CashAmount.Amount.StringFixed(2),
				InstgAgt: agentID{FinInstnId: finInstnID{BIC: c.institutionBIC}},
			},
			CdtTrfTxInf: creditTransferInfo{
				PmtId:          paymentID{InstrId: settlement.ID[:8], EndToEndId: settlement.ID},
				IntrBkSttlmAmt: amountWithCcy{Ccy: string(settlement.CashAmount.Currency), Value: settlement.CashAmount.Amount.StringFixed(2)},
				IntrBkSttlmDt:  settlement.SettlementDate.Format("2006-01-02"),
				Dbtr:           partyID{Nm: settlement.Deliverer},
				Cdtr:           partyID{Nm: settlement.Receiver},
			},
		},
	}

	xmlBytes, _ := xml.MarshalIndent(msg, "", "  ")
	xmlStr := string(xmlBytes)
	status, details := validateXML(xmlStr)

	return model.ISO20022Message{
		MessageType:       "pacs.008.001.10",
		Description:       "FI to FI Customer Credit Transfer",
		XMLContent:        xmlStr,
		ValidationStatus:  status,
		ValidationDetails: details,
	}
}

func validateXML(xmlContent string) (string, []string) {
	cmd := exec.Command("python", "scripts/validate_iso.py")
	cmd.Stdin = bytes.NewReader([]byte(xmlContent))

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return "Error checking validation", []string{err.Error()}
	}

	var result struct {
		Valid   bool     `json:"valid"`
		Message string   `json:"message"`
		Details []string `json:"details"`
		Error   string   `json:"error"`
	}

	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		return "Parse Error", []string{"Failed to read validator output"}
	}

	if !result.Valid {
		if result.Error != "" {
			return "ValidationError", []string{result.Error}
		}
		return "Invalid", result.Details
	}

	return "Valid", result.Details
}

// GenerateSese023 generates a sese.023 SecuritiesSettlementTransactionInstruction.
func (c *ISO20022Client) GenerateSese023(settlement *model.Settlement, isin string) model.ISO20022Message {
	msg := sese023Document{
		XMLName: xml.Name{Local: "Document"},
		Xmlns:   "urn:iso:std:iso:20022:tech:xsd:sese.023.001.11",
		SctiesSttlmTxInstr: &sctiesSttlmTxInstr{
			TxId: fmt.Sprintf("SESE023-%s", settlement.ID[:8]),
			SttlmTpAndAdtlParams: sttlmType{
				SctiesMvmntTp: "DELI",
				Pmt:           "APMT",
			},
			FinInstrmId: finInstrmID{ISIN: isin},
			SttlmAmt:    amountWithCcy{Ccy: string(settlement.CashAmount.Currency), Value: settlement.CashAmount.Amount.StringFixed(2)},
			SttlmDt:     settlement.SettlementDate.Format("2006-01-02"),
			DlvrgPty:    partyID{Nm: settlement.Deliverer},
			RcvgPty:     partyID{Nm: settlement.Receiver},
		},
	}

	xmlBytes, _ := xml.MarshalIndent(msg, "", "  ")
	xmlStr := string(xmlBytes)
	status, details := validateXML(xmlStr)

	return model.ISO20022Message{
		MessageType:       "sese.023.001.11",
		Description:       "Securities Settlement Transaction Instruction",
		XMLContent:        xmlStr,
		ValidationStatus:  status,
		ValidationDetails: details,
	}
}

// GenerateCamt054 generates a camt.054 BankToCustomerDebitCreditNotification.
func (c *ISO20022Client) GenerateCamt054(accountName string, amount decimal.Decimal, currency model.Currency, date time.Time, description string) model.ISO20022Message {
	msg := camt054Document{
		XMLName: xml.Name{Local: "Document"},
		Xmlns:   "urn:iso:std:iso:20022:tech:xsd:camt.054.001.08",
		BkToCstmrDbtCdtNtfctn: &bkToCstmrDbtCdtNtfctn{
			GrpHdr: groupHeader{
				MsgId:   fmt.Sprintf("CAMT054-%s", time.Now().Format("20060102150405")),
				CreDtTm: time.Now().Format("2006-01-02T15:04:05"),
			},
			Ntfctn: notification{
				Id:      accountName,
				CreDtTm: time.Now().Format("2006-01-02T15:04:05"),
				Acct:    accountRef{Id: accountName},
				Ntry: notificationEntry{
					Amt:          amountWithCcy{Ccy: string(currency), Value: amount.StringFixed(2)},
					CdtDbtInd:    "CRDT",
					Sts:          "BOOK",
					BookgDt:      date.Format("2006-01-02"),
					AddtlNtryInf: description,
				},
			},
		},
	}

	xmlBytes, _ := xml.MarshalIndent(msg, "", "  ")
	xmlStr := string(xmlBytes)
	status, details := validateXML(xmlStr)

	return model.ISO20022Message{
		MessageType:       "camt.054.001.08",
		Description:       "Bank To Customer Debit Credit Notification",
		XMLContent:        xmlStr,
		ValidationStatus:  status,
		ValidationDetails: details,
	}
}

// XML structure types for ISO 20022 messages

type pacs008Document struct {
	XMLName           xml.Name           `xml:"Document"`
	Xmlns             string             `xml:"xmlns,attr"`
	FIToFICstmrCdtTrf *fiToFICstmrCdtTrf `xml:"FIToFICstmrCdtTrf"`
}

type fiToFICstmrCdtTrf struct {
	GrpHdr      groupHeader        `xml:"GrpHdr"`
	CdtTrfTxInf creditTransferInfo `xml:"CdtTrfTxInf"`
}

type groupHeader struct {
	MsgId    string  `xml:"MsgId"`
	CreDtTm  string  `xml:"CreDtTm"`
	NbOfTxs  string  `xml:"NbOfTxs,omitempty"`
	CtrlSum  string  `xml:"CtrlSum,omitempty"`
	InstgAgt agentID `xml:"InstgAgt,omitempty"`
}

type agentID struct {
	FinInstnId finInstnID `xml:"FinInstnId"`
}

type finInstnID struct {
	BIC string `xml:"BIC"`
}

type creditTransferInfo struct {
	PmtId          paymentID     `xml:"PmtId"`
	IntrBkSttlmAmt amountWithCcy `xml:"IntrBkSttlmAmt"`
	IntrBkSttlmDt  string        `xml:"IntrBkSttlmDt"`
	Dbtr           partyID       `xml:"Dbtr"`
	Cdtr           partyID       `xml:"Cdtr"`
}

type paymentID struct {
	InstrId    string `xml:"InstrId"`
	EndToEndId string `xml:"EndToEndId"`
}

type amountWithCcy struct {
	Ccy   string `xml:"Ccy,attr"`
	Value string `xml:",chardata"`
}

type partyID struct {
	Nm string `xml:"Nm"`
}

type sese023Document struct {
	XMLName            xml.Name            `xml:"Document"`
	Xmlns              string              `xml:"xmlns,attr"`
	SctiesSttlmTxInstr *sctiesSttlmTxInstr `xml:"SctiesSttlmTxInstr"`
}

type sctiesSttlmTxInstr struct {
	TxId                 string        `xml:"TxId"`
	SttlmTpAndAdtlParams sttlmType     `xml:"SttlmTpAndAdtlParams"`
	FinInstrmId          finInstrmID   `xml:"FinInstrmId"`
	SttlmAmt             amountWithCcy `xml:"SttlmAmt"`
	SttlmDt              string        `xml:"SttlmDt"`
	DlvrgPty             partyID       `xml:"DlvrgPty"`
	RcvgPty              partyID       `xml:"RcvgPty"`
}

type sttlmType struct {
	SctiesMvmntTp string `xml:"SctiesMvmntTp"`
	Pmt           string `xml:"Pmt"`
}

type finInstrmID struct {
	ISIN string `xml:"ISIN"`
}

type camt054Document struct {
	XMLName               xml.Name               `xml:"Document"`
	Xmlns                 string                 `xml:"xmlns,attr"`
	BkToCstmrDbtCdtNtfctn *bkToCstmrDbtCdtNtfctn `xml:"BkToCstmrDbtCdtNtfctn"`
}

type bkToCstmrDbtCdtNtfctn struct {
	GrpHdr groupHeader  `xml:"GrpHdr"`
	Ntfctn notification `xml:"Ntfctn"`
}

type notification struct {
	Id      string            `xml:"Id"`
	CreDtTm string            `xml:"CreDtTm"`
	Acct    accountRef        `xml:"Acct"`
	Ntry    notificationEntry `xml:"Ntry"`
}

type accountRef struct {
	Id string `xml:"Id"`
}

type notificationEntry struct {
	Amt          amountWithCcy `xml:"Amt"`
	CdtDbtInd    string        `xml:"CdtDbtInd"`
	Sts          string        `xml:"Sts"`
	BookgDt      string        `xml:"BookgDt"`
	AddtlNtryInf string        `xml:"AddtlNtryInf"`
}
