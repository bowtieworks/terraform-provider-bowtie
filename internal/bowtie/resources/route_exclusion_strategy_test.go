package resources

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestApplyStrategyValidation(t *testing.T) {
	if _, diags := applyStrategyToClient(strategyPercentageUserMatch, types.Int64Null()); !diags.HasError() {
		t.Fatal("percentage strategy without apply_strategy_percentage should error")
	}

	if _, diags := applyStrategyToClient(strategyAlways, types.Int64Value(25)); !diags.HasError() {
		t.Fatal("non-percentage strategy with apply_strategy_percentage should error")
	}

	got, diags := applyStrategyToClient(strategyPercentageDeviceMatch, types.Int64Value(25))
	if diags.HasError() {
		t.Fatalf("percentage strategy should be valid: %v", diags)
	}
	if got.Type != "percentage-device-match" || got.Value == nil || *got.Value != 25 {
		t.Fatalf("unexpected strategy conversion: %+v", got)
	}
}
