package formatter

import (
	"strings"
	"testing"

	"github.com/prebid/prebid-server/v3/endpoints/ctv/vast/model"
)

func TestGenericFormatter(t *testing.T) {
	config := Config{
		Profile:        ReceiverGeneric,
		DefaultVersion: "4.0",
	}

	formatter := NewGenericFormatter(config)

	vast := &model.VAST{
		Version: "4.0",
		Ad: []*model.Ad{
			{
				ID: "test-ad",
				InLine: &model.InLine{
					AdSystem: &model.AdSystem{Value: "TestSystem"},
					AdTitle:  "Test Ad",
					Impression: []model.Impression{
						{Value: "http://example.com/impression"},
					},
					Creatives: &model.Creatives{
						Creative: []*model.Creative{
							{
								ID: "creative1",
								Linear: &model.Linear{
									Duration: "00:00:30",
								},
							},
						},
					},
				},
			},
		},
	}

	data, err := formatter.Format(vast)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	xml := string(data)
	if !strings.Contains(xml, `<VAST`) {
		t.Error("Expected VAST element in output")
	}

	if !strings.Contains(xml, `version="4.0"`) {
		t.Error("Expected version 4.0 in output")
	}

	if !strings.Contains(xml, `<Ad id="test-ad">`) {
		t.Error("Expected Ad element with ID in output")
	}
}

func TestGenericFormatter_DefaultVersion(t *testing.T) {
	config := Config{
		Profile: ReceiverGeneric,
		// No default version specified
	}

	formatter := NewGenericFormatter(config)

	vast := &model.VAST{
		// No version set
		Ad: []*model.Ad{
			{
				ID: "test-ad",
				InLine: &model.InLine{
					AdSystem: &model.AdSystem{Value: "Test"},
					AdTitle:  "Test",
					Impression: []model.Impression{{Value: "http://test.com"}},
				},
			},
		},
	}

	data, err := formatter.Format(vast)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	xml := string(data)
	if !strings.Contains(xml, `version="4.0"`) {
		t.Error("Expected default version 4.0")
	}
}

func TestGAMSSUFormatter(t *testing.T) {
	config := Config{
		Profile:        ReceiverGAMSSU,
		DefaultVersion: "3.0",
	}

	formatter := NewGAMSSUFormatter(config)

	vast := &model.VAST{
		Ad: []*model.Ad{
			{
				InLine: &model.InLine{
					Creatives: &model.Creatives{
						Creative: []*model.Creative{
							{
								Linear: &model.Linear{
									MediaFiles: &model.MediaFiles{
										MediaFile: []model.MediaFile{
											{Value: "http://example.com/video.mp4"},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	data, err := formatter.Format(vast)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	xml := string(data)

	// Check GAM SSU defaults were applied
	if !strings.Contains(xml, `version="3.0"`) {
		t.Error("Expected version 3.0 for GAM SSU")
	}

	if !strings.Contains(xml, `<AdSystem>`) {
		t.Error("Expected AdSystem to be added")
	}

	if !strings.Contains(xml, `<AdTitle>`) {
		t.Error("Expected AdTitle to be added")
	}

	if !strings.Contains(xml, `<Impression`) {
		t.Error("Expected Impression to be added")
	}
}

func TestGAMSSUFormatter_PodSequencing(t *testing.T) {
	config := Config{
		Profile: ReceiverGAMSSU,
	}

	formatter := NewGAMSSUFormatter(config)

	// Pod with multiple ads
	vast := &model.VAST{
		Ad: []*model.Ad{
			{
				InLine: &model.InLine{
					Creatives: &model.Creatives{
						Creative: []*model.Creative{
							{Linear: &model.Linear{}},
						},
					},
				},
			},
			{
				InLine: &model.InLine{
					Creatives: &model.Creatives{
						Creative: []*model.Creative{
							{Linear: &model.Linear{}},
						},
					},
				},
			},
			{
				InLine: &model.InLine{
					Creatives: &model.Creatives{
						Creative: []*model.Creative{
							{Linear: &model.Linear{}},
						},
					},
				},
			},
		},
	}

	data, err := formatter.Format(vast)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	// Parse back to check sequences
	parsed, err := model.Parse(data)
	if err != nil {
		t.Fatalf("Failed to parse formatted VAST: %v", err)
	}

	// Check sequences were assigned
	for i, ad := range parsed.Ad {
		expectedSequence := i + 1
		if ad.Sequence != expectedSequence {
			t.Errorf("Ad %d: expected sequence %d, got %d", i, expectedSequence, ad.Sequence)
		}
	}
}

func TestGAMSSUFormatter_EnsuresIDs(t *testing.T) {
	config := Config{
		Profile: ReceiverGAMSSU,
	}

	formatter := NewGAMSSUFormatter(config)

	vast := &model.VAST{
		Ad: []*model.Ad{
			{
				// No ID
				InLine: &model.InLine{
					Creatives: &model.Creatives{
						Creative: []*model.Creative{
							{
								// No ID
								Linear: &model.Linear{
									MediaFiles: &model.MediaFiles{
										MediaFile: []model.MediaFile{
											{
												// No ID
												Value: "http://example.com/video.mp4",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	data, err := formatter.Format(vast)
	if err != nil {
		t.Fatalf("Format failed: %v", err)
	}

	parsed, err := model.Parse(data)
	if err != nil {
		t.Fatalf("Failed to parse formatted VAST: %v", err)
	}

	// Check IDs were assigned
	if parsed.Ad[0].ID == "" {
		t.Error("Expected Ad ID to be assigned")
	}

	if parsed.Ad[0].InLine.Creatives.Creative[0].ID == "" {
		t.Error("Expected Creative ID to be assigned")
	}

	if parsed.Ad[0].InLine.Creatives.Creative[0].Linear.MediaFiles.MediaFile[0].ID == "" {
		t.Error("Expected MediaFile ID to be assigned")
	}
}

func TestFormatterFactory(t *testing.T) {
	factory := NewFormatterFactory()

	// Test GAM SSU
	gamConfig := Config{Profile: ReceiverGAMSSU}
	gamFormatter := factory.CreateFormatter(gamConfig)
	if _, ok := gamFormatter.(*GAMSSUFormatter); !ok {
		t.Error("Expected GAMSSUFormatter")
	}

	// Test Generic
	genericConfig := Config{Profile: ReceiverGeneric}
	genericFormatter := factory.CreateFormatter(genericConfig)
	if _, ok := genericFormatter.(*GenericFormatter); !ok {
		t.Error("Expected GenericFormatter")
	}

	// Test unknown (should default to Generic)
	unknownConfig := Config{Profile: "UNKNOWN"}
	unknownFormatter := factory.CreateFormatter(unknownConfig)
	if _, ok := unknownFormatter.(*GenericFormatter); !ok {
		t.Error("Expected GenericFormatter for unknown profile")
	}
}

func TestFormatEmptyVAST(t *testing.T) {
	config := Config{
		Profile:        ReceiverGAMSSU,
		DefaultVersion: "3.0",
	}

	data, err := FormatEmptyVAST(config)
	if err != nil {
		t.Fatalf("FormatEmptyVAST failed: %v", err)
	}

	xml := string(data)
	if !strings.Contains(xml, `<VAST`) {
		t.Error("Expected VAST element")
	}

	if !strings.Contains(xml, `version="3.0"`) {
		t.Error("Expected version 3.0")
	}

	// Should have no ads
	if strings.Contains(xml, `<Ad`) {
		t.Error("Expected no Ad elements in empty VAST")
	}
}

func TestValidateVAST(t *testing.T) {
	tests := []struct {
		name      string
		vast      *model.VAST
		expectErr bool
	}{
		{
			name:      "Nil VAST",
			vast:      nil,
			expectErr: true,
		},
		{
			name: "Missing version",
			vast: &model.VAST{
				Ad: []*model.Ad{},
			},
			expectErr: true,
		},
		{
			name: "Valid empty VAST",
			vast: &model.VAST{
				Version: "4.0",
				Ad:      []*model.Ad{},
			},
			expectErr: false,
		},
		{
			name: "Valid VAST with ad",
			vast: &model.VAST{
				Version: "4.0",
				Ad: []*model.Ad{
					{
						ID: "ad1",
						InLine: &model.InLine{
							AdSystem:   &model.AdSystem{Value: "Test"},
							AdTitle:    "Test Ad",
							Impression: []model.Impression{{Value: "http://test.com"}},
							Creatives: &model.Creatives{
								Creative: []*model.Creative{
									{
										ID: "creative1",
										Linear: &model.Linear{
											Duration: "00:00:30",
										},
									},
								},
							},
						},
					},
				},
			},
			expectErr: false,
		},
		{
			name: "Ad with neither InLine nor Wrapper",
			vast: &model.VAST{
				Version: "4.0",
				Ad: []*model.Ad{
					{ID: "ad1"},
				},
			},
			expectErr: true,
		},
		{
			name: "InLine missing AdSystem",
			vast: &model.VAST{
				Version: "4.0",
				Ad: []*model.Ad{
					{
						ID: "ad1",
						InLine: &model.InLine{
							AdTitle:    "Test",
							Impression: []model.Impression{{Value: "http://test.com"}},
							Creatives: &model.Creatives{
								Creative: []*model.Creative{{Linear: &model.Linear{}}},
							},
						},
					},
				},
			},
			expectErr: true,
		},
		{
			name: "InLine missing AdTitle",
			vast: &model.VAST{
				Version: "4.0",
				Ad: []*model.Ad{
					{
						ID: "ad1",
						InLine: &model.InLine{
							AdSystem:   &model.AdSystem{Value: "Test"},
							Impression: []model.Impression{{Value: "http://test.com"}},
							Creatives: &model.Creatives{
								Creative: []*model.Creative{{Linear: &model.Linear{}}},
							},
						},
					},
				},
			},
			expectErr: true,
		},
		{
			name: "InLine missing Impression",
			vast: &model.VAST{
				Version: "4.0",
				Ad: []*model.Ad{
					{
						ID: "ad1",
						InLine: &model.InLine{
							AdSystem: &model.AdSystem{Value: "Test"},
							AdTitle:  "Test",
							Creatives: &model.Creatives{
								Creative: []*model.Creative{{Linear: &model.Linear{}}},
							},
						},
					},
				},
			},
			expectErr: true,
		},
		{
			name: "InLine missing Creatives",
			vast: &model.VAST{
				Version: "4.0",
				Ad: []*model.Ad{
					{
						ID: "ad1",
						InLine: &model.InLine{
							AdSystem:   &model.AdSystem{Value: "Test"},
							AdTitle:    "Test",
							Impression: []model.Impression{{Value: "http://test.com"}},
						},
					},
				},
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateVAST(tt.vast)
			if tt.expectErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestFormatter_NilVAST(t *testing.T) {
	config := Config{Profile: ReceiverGeneric}
	formatter := NewGenericFormatter(config)

	_, err := formatter.Format(nil)
	if err == nil {
		t.Error("Expected error for nil VAST")
	}
}
