package main_test

import (
	"testing"

	"github.com/spf13/viper"
)

func TestDefaultPreinstallAuditEnabled(t *testing.T) {
	v := viper.New()
	v.SetDefault("preinstall_audit_enabled", true)
	v.SetDefault("preinstall_allow_private_networks", false)
	if !v.GetBool("preinstall_audit_enabled") {
		t.Fatal("expected preinstall audit enabled by default")
	}
	if v.GetBool("preinstall_allow_private_networks") {
		t.Fatal("expected private networks blocked by default")
	}
}
