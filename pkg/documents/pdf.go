package documents

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"ispilolite/internal/models"
)

type Store struct{ Root string }

func NewStore() *Store {
	root := strings.TrimSpace(os.Getenv("DOCUMENT_STORAGE_ROOT"))
	if root == "" {
		root = "storage/documents"
	}
	return &Store{Root: root}
}

func (s *Store) SaveQuotation(q *models.Quotation) (*models.Document, error) {
	if err := os.MkdirAll(s.Root, 0750); err != nil {
		return nil, err
	}
	name := q.ID + ".pdf"
	path := filepath.Join(s.Root, name)
	if err := os.WriteFile(path, quotationPDF(q), 0600); err != nil {
		return nil, err
	}
	return &models.Document{ID: uuid.NewString(), OwnerID: q.IssuerID, QuotationID: q.ID, StoragePath: path, FileName: "quotation-" + q.QuotationNumber + ".pdf", ContentType: "application/pdf", Visibility: "PRIVATE", CreatedAt: q.CreatedAt}, nil
}

func quotationPDF(q *models.Quotation) []byte {
	text := strings.NewReplacer("\\", "\\\\", "(", "\\(", ")", "\\)", "\n", " ").Replace(fmt.Sprintf("Quotation %s  Total %.2f %s", q.QuotationNumber, q.TotalAmount, q.Currency))
	stream := []byte(fmt.Sprintf("BT /F1 18 Tf 72 720 Td (%s) Tj ET", text))
	objects := []string{"<< /Type /Catalog /Pages 2 0 R >>", "<< /Type /Pages /Kids [3 0 R] /Count 1 >>", "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 4 0 R >> >> /Contents 5 0 R >>", "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>", fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(stream), stream)}
	var out bytes.Buffer
	out.WriteString("%PDF-1.4\n")
	offsets := []int{0}
	for i, object := range objects {
		offsets = append(offsets, out.Len())
		fmt.Fprintf(&out, "%d 0 obj\n%s\nendobj\n", i+1, object)
	}
	xref := out.Len()
	fmt.Fprintf(&out, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for _, offset := range offsets[1:] {
		fmt.Fprintf(&out, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&out, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xref)
	return out.Bytes()
}
