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

	// TODO (pbm) test more prices
	// interval := 0.01
	// price := 0.00
	// for price < 20.01 {
	// 	fmt.Println(price)

	// 	priceMap := GetPriceBucketString(price)
	// 	fmt.Println(priceMap)

	// 	price += interval
	// }
}
