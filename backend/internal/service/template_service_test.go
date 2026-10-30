package service

import (
	"context"
	"testing"
)

func TestTemplateServiceUsesCodeCatalogWithoutDatabase(t *testing.T) {
	list, err := NewTemplateService().List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != "vertical" {
		t.Fatal("unexpected template catalog")
	}
}
