package handlers

import (
	"encoding/json"
	"testing"

	"github.com/sakid00/massmaker-be/internal/service"
)

func TestInquiryJSONEmptyBudgetDoesNotDecode(t *testing.T) {
	t.Parallel()
	raw := []byte(`{
		"projectName":"Booth","productName":"Stiker","categorySlug":"sticker",
		"quantity":100,"designCount":1,"wantSample":false,
		"requestedReadyOn":"2026-10-10","requestedShipOn":"2026-10-20",
		"budgetMinIdr":"","budgetMaxIdr":"",
		"attachments":[{"url":"https://cdn.example/inquiries/u/a.jpg","contentType":"image/jpeg","originalFilename":"a.jpg","byteSize":1200}]
	}`)
	var rawIn service.CreateInquiryInput
	if err := json.Unmarshal(raw, &rawIn); err == nil {
		t.Fatal("empty string budget should fail strict json")
	}
	var in service.CreateInquiryInput
	if err := decodeInquiryFrom(raw, &in); err != nil {
		t.Fatal(err)
	}
	if in.BudgetMinIDR != nil || in.BudgetMaxIDR != nil {
		t.Fatalf("expected null budgets, got %v %v", in.BudgetMinIDR, in.BudgetMaxIDR)
	}
	if in.Quantity != 100 {
		t.Fatalf("quantity %d", in.Quantity)
	}
}

func TestInquiryJSONStringQuantity(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"projectName":"Booth","productName":"Stiker","categorySlug":"sticker","quantity":"24","designCount":"1","budgetMinIdr":"15000","attachments":[]}`)
	var in service.CreateInquiryInput
	if err := decodeInquiryFrom(raw, &in); err != nil {
		t.Fatal(err)
	}
	if in.Quantity != 24 || in.DesignCount != 1 || in.BudgetMinIDR == nil || *in.BudgetMinIDR != 15000 {
		t.Fatalf("%+v", in)
	}
}

func TestCoerceInquiryJSONRejectsGarbage(t *testing.T) {
	t.Parallel()
	if _, err := coerceInquiryJSON([]byte("not-json")); err == nil {
		t.Fatal("expected error")
	}
}
