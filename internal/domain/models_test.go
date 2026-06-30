package domain

import (
	"encoding/hex"
	"testing"
	"time"
)

func TestGenerateOrderID_Length(t *testing.T) {
	id := GenerateOrderID()
	if len(id) != 14 {
		t.Errorf("want 14 chars, got %d (%s)", len(id), id)
	}
}

func TestGenerateOrderID_ValidHex(t *testing.T) {
	id := GenerateOrderID()
	if _, err := hex.DecodeString(id); err != nil {
		t.Errorf("not valid hex: %s — %v", id, err)
	}
}

func TestGenerateOrderID_Unique(t *testing.T) {
	seen := make(map[string]bool, 1000)
	for i := 0; i < 1000; i++ {
		id := GenerateOrderID()
		if seen[id] {
			t.Fatalf("duplicate ID at iteration %d: %s", i, id)
		}
		seen[id] = true
	}
}

func TestOrder_BeforeCreate_SetsID(t *testing.T) {
	o := &Order{}
	if err := o.BeforeCreate(nil); err != nil {
		t.Fatal(err)
	}
	if o.ID == "" {
		t.Error("ID not set")
	}
	if len(o.ID) != 14 {
		t.Errorf("want 14 chars, got %d", len(o.ID))
	}
}

func TestOrder_BeforeCreate_PreservesExistingID(t *testing.T) {
	o := &Order{ID: "custom-id-here"}
	if err := o.BeforeCreate(nil); err != nil {
		t.Fatal(err)
	}
	if o.ID != "custom-id-here" {
		t.Errorf("existing ID overwritten: got %s", o.ID)
	}
}

func TestOrder_BeforeCreate_SetsCreatedAt(t *testing.T) {
	before := time.Now().UnixMilli()
	o := &Order{}
	if err := o.BeforeCreate(nil); err != nil {
		t.Fatal(err)
	}
	after := time.Now().UnixMilli()

	if o.CreatedAt < before || o.CreatedAt > after {
		t.Errorf("created_at %d not in range [%d, %d]", o.CreatedAt, before, after)
	}
}

func TestOrder_BeforeCreate_PreservesExistingCreatedAt(t *testing.T) {
	ts := int64(1700000000000)
	o := &Order{CreatedAt: ts}
	if err := o.BeforeCreate(nil); err != nil {
		t.Fatal(err)
	}
	if o.CreatedAt != ts {
		t.Errorf("existing created_at overwritten: got %d", o.CreatedAt)
	}
}

func TestUser_BeforeCreate_SetsFields(t *testing.T) {
	u := &User{}
	if err := u.BeforeCreate(nil); err != nil {
		t.Fatal(err)
	}
	if u.ID == "" {
		t.Error("user ID not set")
	}
	if u.CreatedAt == 0 {
		t.Error("user created_at not set")
	}
	if u.UpdatedAt != u.CreatedAt {
		t.Errorf("updated_at (%d) != created_at (%d)", u.UpdatedAt, u.CreatedAt)
	}
}

func TestUser_BeforeCreate_PreservesExistingID(t *testing.T) {
	u := &User{ID: "user-custom"}
	if err := u.BeforeCreate(nil); err != nil {
		t.Fatal(err)
	}
	if u.ID != "user-custom" {
		t.Errorf("user ID overwritten: got %s", u.ID)
	}
}

func TestProduct_BeforeCreate_SetsFields(t *testing.T) {
	p := &Product{}
	if err := p.BeforeCreate(nil); err != nil {
		t.Fatal(err)
	}
	if p.ID == "" {
		t.Error("product ID not set")
	}
	if p.CreatedAt == 0 {
		t.Error("product created_at not set")
	}
	if p.UpdatedAt != p.CreatedAt {
		t.Errorf("updated_at (%d) != created_at (%d)", p.UpdatedAt, p.CreatedAt)
	}
}

func TestProduct_BeforeCreate_PreservesExistingID(t *testing.T) {
	p := &Product{ID: "prod-custom"}
	if err := p.BeforeCreate(nil); err != nil {
		t.Fatal(err)
	}
	if p.ID != "prod-custom" {
		t.Errorf("product ID overwritten: got %s", p.ID)
	}
}
