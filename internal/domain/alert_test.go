package domain

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestNewAlertConfig_Direction(t *testing.T) {
	t.Run("UP direction when target > current", func(t *testing.T) {
		alert := NewAlertConfig("BTC", decimal.NewFromInt(50000), decimal.NewFromInt(45000), "UPBIT", false)
		if alert.Direction != "UP" {
			t.Errorf("Expected UP, got %s", alert.Direction)
		}
	})

	t.Run("DOWN direction when target < current", func(t *testing.T) {
		alert := NewAlertConfig("BTC", decimal.NewFromInt(40000), decimal.NewFromInt(45000), "UPBIT", false)
		if alert.Direction != "DOWN" {
			t.Errorf("Expected DOWN, got %s", alert.Direction)
		}
	})

	t.Run("UP direction when target = current", func(t *testing.T) {
		alert := NewAlertConfig("BTC", decimal.NewFromInt(45000), decimal.NewFromInt(45000), "UPBIT", false)
		if alert.Direction != "UP" {
			t.Errorf("Expected UP for equal prices, got %s", alert.Direction)
		}
	})
}

func TestAlertConfig_CheckCondition(t *testing.T) {
	t.Run("UP alert triggers at target", func(t *testing.T) {
		alert := NewAlertConfig("BTC", decimal.NewFromInt(50000), decimal.NewFromInt(45000), "UPBIT", false)
		if !alert.CheckCondition(decimal.NewFromInt(50000)) {
			t.Error("Should trigger at target price")
		}
	})

	t.Run("UP alert triggers above target", func(t *testing.T) {
		alert := NewAlertConfig("BTC", decimal.NewFromInt(50000), decimal.NewFromInt(45000), "UPBIT", false)
		if !alert.CheckCondition(decimal.NewFromInt(51000)) {
			t.Error("Should trigger above target price")
		}
	})

	t.Run("UP alert does not trigger below target", func(t *testing.T) {
		alert := NewAlertConfig("BTC", decimal.NewFromInt(50000), decimal.NewFromInt(45000), "UPBIT", false)
		if alert.CheckCondition(decimal.NewFromInt(49000)) {
			t.Error("Should not trigger below target price")
		}
	})

	t.Run("DOWN alert triggers at target", func(t *testing.T) {
		alert := NewAlertConfig("BTC", decimal.NewFromInt(40000), decimal.NewFromInt(45000), "UPBIT", false)
		if !alert.CheckCondition(decimal.NewFromInt(40000)) {
			t.Error("Should trigger at target price")
		}
	})

	t.Run("Inactive alert does not trigger", func(t *testing.T) {
		alert := NewAlertConfig("BTC", decimal.NewFromInt(50000), decimal.NewFromInt(45000), "UPBIT", false)
		alert.SetActive(false)
		if alert.CheckCondition(decimal.NewFromInt(55000)) {
			t.Error("Inactive alert should not trigger")
		}
	})
}

