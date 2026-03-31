package models

import (
	"reflect"
	"testing"
)

func TestAdminModel(t *testing.T) {
	admin := &Admin{
		Username:     "test",
		IsSuperAdmin: true,
	}

	if admin.Username != "test" {
		t.Errorf("Expected username test, got %s", admin.Username)
	}

	if !admin.IsSuperAdmin {
		t.Error("Expected IsSuperAdmin to be true")
	}

	// Just a simple check to ensure tags exist on models
	field, ok := reflect.TypeOf(admin).Elem().FieldByName("Username")
	if !ok {
		t.Error("Expected Admin to have Username field")
	}
	
	jsonTag := field.Tag.Get("json")
	if jsonTag != "username" {
		t.Errorf("Expected json tag 'username', got '%s'", jsonTag)
	}
}

func TestCoreModel(t *testing.T) {
	core := &Core{
		Name: "singbox",
	}

	if core.Name != "singbox" {
		t.Errorf("Expected core name singbox, got %s", core.Name)
	}
}
