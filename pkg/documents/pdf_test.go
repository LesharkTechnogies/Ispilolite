package documents

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"ispilolite/internal/models"
)

func TestQuotationPDFAppearance(t *testing.T) {
	pdf := quotationPDF(&models.Quotation{QuotationNumber: "QT-1", TotalAmount: 1250, Currency: "KES", CreatedAt: time.Now()})
	if !bytes.HasPrefix(pdf, []byte("%PDF-1.4")) {
		t.Fatal("quotation output is not a PDF")
	}
	if !strings.Contains(string(pdf), "QT-1") {
		t.Fatal("quotation number missing from PDF")
	}
}
