package handlers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/zap"

	"github.com/myroutine/backend/internal/api/middleware"
)

// oversizedImportBody builds a syntactically valid JSON body whose CSV content
// is larger than the import cap, so only the size limit can reject it.
func oversizedImportBody() *bytes.Reader {
	content := strings.Repeat("a", maxImportBodyBytes+1024)
	return bytes.NewReader([]byte(`{"filename":"x.csv","content":"` + content + `"}`))
}

func TestImportPreviewRejectsOversizedBody(t *testing.T) {
	h := NewImportHandler(nil, nil, zap.NewNop())

	req := httptest.NewRequest(http.MethodPost, "/finance/imports/preview", oversizedImportBody())
	rec := httptest.NewRecorder()
	h.Preview(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, esperado 413", rec.Code)
	}
}

func TestImportCreateRejectsOversizedBody(t *testing.T) {
	h := NewImportHandler(nil, nil, zap.NewNop())

	req := httptest.NewRequest(http.MethodPost, "/finance/imports", oversizedImportBody())
	req = req.WithContext(context.WithValue(req.Context(), middleware.UserIDKey, "00000000-0000-0000-0000-000000000000"))
	rec := httptest.NewRecorder()
	h.Create(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, esperado 413", rec.Code)
	}
}

// A body just under the cap still goes through the size gate.
func TestImportPreviewAcceptsBodyUnderCap(t *testing.T) {
	h := NewImportHandler(nil, nil, zap.NewNop())

	body := `{"content":"Data;Valor;Histórico\n12/08/2026;-45,90;IFOOD"}`
	req := httptest.NewRequest(http.MethodPost, "/finance/imports/preview", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.Preview(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, esperado 200 (%s)", rec.Code, rec.Body.String())
	}
}
