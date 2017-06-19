package pbs

import (
	"testing"
)

func TestGetPriceBucketString(t *testing.T) {
	price := 1.87
	priceMap := GetPriceBucketString(price)

	if priceMap["low"] != "1.50" {
		t.Error("Expected 1.50")
	}
	if priceMap["med"] != "1.80" {
		t.Error("Expected 1.80")
	}
	if priceMap["high"] != "1.87" {
		t.Error("Expected 1.87")
	}
	if priceMap["auto"] != "1.85" {
		t.Error("Expected 1.85")
	}
	if priceMap["dense"] != "1.87" {
		t.Error("Expected 1.87")
	}

	// test a cpm above the max in low price bucket
	price = 5.72
	priceMap = GetPriceBucketString(price)

	if priceMap["low"] != "5.00" {
		t.Error("Expected 5.00")
	}
	if priceMap["med"] != "5.70" {
		t.Error("Expected 5.70")
	}
	if priceMap["high"] != "5.72" {
		t.Error("Expected 5.72")
	}
	if priceMap["auto"] != "5.70" {
		t.Error("Expected 5.70")
	}
	if priceMap["dense"] != "5.70" {
		t.Error("Expected 5.70")
	}
}
