package formatter

import (
	"fmt"

	"github.com/prebid/prebid-server/v3/endpoints/ctv/vast/model"
)

// ReceiverProfile defines the target receiver profile
type ReceiverProfile string

const (
	// ReceiverGAMSSU is Google Ad Manager Server-Side Unified ad format
	ReceiverGAMSSU ReceiverProfile = "GAM_SSU"
	// ReceiverGeneric is a generic VAST receiver
	ReceiverGeneric ReceiverProfile = "GENERIC"
)

// Config holds formatter configuration
type Config struct {
	Profile        ReceiverProfile
	DefaultVersion string
}

// Formatter formats VAST for specific receiver profiles
type Formatter interface {
	// Format formats VAST according to receiver requirements
	Format(vast *model.VAST) ([]byte, error)
}

// FormatterFactory creates formatters for different profiles
type FormatterFactory struct{}

// NewFormatterFactory creates a new FormatterFactory
func NewFormatterFactory() *FormatterFactory {
	return &FormatterFactory{}
}

// CreateFormatter creates a formatter for the specified profile
func (f *FormatterFactory) CreateFormatter(config Config) Formatter {
	switch config.Profile {
	case ReceiverGAMSSU:
		return NewGAMSSUFormatter(config)
	case ReceiverGeneric:
		return NewGenericFormatter(config)
	default:
		// Default to generic
		return NewGenericFormatter(config)
	}
}

// GenericFormatter implements standard VAST formatting
type GenericFormatter struct {
	config Config
}

// NewGenericFormatter creates a new GenericFormatter
func NewGenericFormatter(config Config) Formatter {
	if config.DefaultVersion == "" {
		config.DefaultVersion = "4.0"
	}
	return &GenericFormatter{config: config}
}

// Format implements Formatter.Format for generic VAST
func (f *GenericFormatter) Format(vast *model.VAST) ([]byte, error) {
	if vast == nil {
		return nil, fmt.Errorf("vast is nil")
	}

	// Ensure version is set
	if vast.Version == "" {
		vast.Version = f.config.DefaultVersion
	}

	// Marshal to XML
	return vast.Marshal()
}

// GAMSSUFormatter implements GAM Server-Side Unified formatting
type GAMSSUFormatter struct {
	config Config
}

// NewGAMSSUFormatter creates a new GAMSSUFormatter
func NewGAMSSUFormatter(config Config) Formatter {
	if config.DefaultVersion == "" {
		config.DefaultVersion = "3.0" // GAM SSU typically uses VAST 3.0
	}
	return &GAMSSUFormatter{config: config}
}

// Format implements Formatter.Format for GAM SSU
func (f *GAMSSUFormatter) Format(vast *model.VAST) ([]byte, error) {
	if vast == nil {
		return nil, fmt.Errorf("vast is nil")
	}

	// Apply GAM SSU specific transformations
	f.applyGAMSSUTransformations(vast)

	// Ensure version is set (GAM SSU prefers 3.0)
	if vast.Version == "" {
		vast.Version = f.config.DefaultVersion
	}

	// Marshal to XML
	return vast.Marshal()
}

// applyGAMSSUTransformations applies GAM SSU specific requirements
func (f *GAMSSUFormatter) applyGAMSSUTransformations(vast *model.VAST) {
	// GAM SSU specific requirements:
	// 1. Ensure Ad IDs are present
	// 2. Ensure impression tracking is set up
	// 3. Validate creative IDs
	// 4. Ensure proper sequence numbering for pods

	for i, ad := range vast.Ad {
		// Ensure Ad ID
		if ad.ID == "" {
			ad.ID = fmt.Sprintf("ad-%d", i+1)
		}

		// Ensure sequence for pods
		if ad.Sequence == 0 && len(vast.Ad) > 1 {
			ad.Sequence = i + 1
		}

		// Process InLine ads
		if ad.InLine != nil {
			f.processInLineForGAM(ad.InLine, ad.ID)
		}

		// Process Wrapper ads
		if ad.Wrapper != nil {
			f.processWrapperForGAM(ad.Wrapper, ad.ID)
		}
	}
}

// processInLineForGAM processes InLine element for GAM SSU
func (f *GAMSSUFormatter) processInLineForGAM(inline *model.InLine, adID string) {
	// Ensure at least one impression
	if len(inline.Impression) == 0 {
		inline.Impression = []model.Impression{
			{ID: fmt.Sprintf("%s-impression", adID)},
		}
	}

	// Ensure AdSystem is set
	if inline.AdSystem == nil {
		inline.AdSystem = &model.AdSystem{
			Value: "Prebid Server",
		}
	}

	// Ensure AdTitle is set
	if inline.AdTitle == "" {
		inline.AdTitle = fmt.Sprintf("Ad %s", adID)
	}

	// Process creatives
	if inline.Creatives != nil {
		for j, creative := range inline.Creatives.Creative {
			// Ensure creative ID
			if creative.ID == "" {
				creative.ID = fmt.Sprintf("%s-creative-%d", adID, j+1)
			}

			// Process Linear creative
			if creative.Linear != nil {
				f.processLinearForGAM(creative.Linear, creative.ID)
			}
		}
	}
}

// processLinearForGAM processes Linear creative for GAM SSU
func (f *GAMSSUFormatter) processLinearForGAM(linear *model.Linear, creativeID string) {
	// Ensure duration is set
	if linear.Duration == "" {
		linear.Duration = "00:00:30" // Default 30 seconds
	}

	// Ensure MediaFiles exist
	if linear.MediaFiles == nil {
		linear.MediaFiles = &model.MediaFiles{}
	}

	// Validate media files have required attributes
	for i, mediaFile := range linear.MediaFiles.MediaFile {
		if mediaFile.ID == "" {
			linear.MediaFiles.MediaFile[i].ID = fmt.Sprintf("%s-media-%d", creativeID, i+1)
		}
		if mediaFile.Delivery == "" {
			linear.MediaFiles.MediaFile[i].Delivery = "progressive"
		}
		if mediaFile.Type == "" {
			linear.MediaFiles.MediaFile[i].Type = "video/mp4"
		}
	}
}

// processWrapperForGAM processes Wrapper element for GAM SSU
func (f *GAMSSUFormatter) processWrapperForGAM(wrapper *model.Wrapper, adID string) {
	// Ensure at least one impression
	if len(wrapper.Impression) == 0 {
		wrapper.Impression = []model.Impression{
			{ID: fmt.Sprintf("%s-impression", adID)},
		}
	}

	// Ensure AdSystem is set
	if wrapper.AdSystem == nil {
		wrapper.AdSystem = &model.AdSystem{
			Value: "Prebid Server",
		}
	}
}

// FormatEmptyVAST creates a formatted empty VAST response
func FormatEmptyVAST(config Config) ([]byte, error) {
	version := config.DefaultVersion
	if version == "" {
		if config.Profile == ReceiverGAMSSU {
			version = "3.0"
		} else {
			version = "4.0"
		}
	}

	vast := model.NewEmptyVAST(version)
	
	factory := NewFormatterFactory()
	formatter := factory.CreateFormatter(config)
	
	return formatter.Format(vast)
}

// ValidateVAST performs basic VAST validation
func ValidateVAST(vast *model.VAST) error {
	if vast == nil {
		return fmt.Errorf("vast is nil")
	}

	if vast.Version == "" {
		return fmt.Errorf("vast version is required")
	}

	// If has ads, validate structure
	for i, ad := range vast.Ad {
		if ad.InLine == nil && ad.Wrapper == nil {
			return fmt.Errorf("ad %d has neither InLine nor Wrapper", i)
		}

		if ad.InLine != nil {
			if err := validateInLine(ad.InLine, i); err != nil {
				return err
			}
		}

		if ad.Wrapper != nil {
			if err := validateWrapper(ad.Wrapper, i); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateInLine validates InLine element
func validateInLine(inline *model.InLine, adIndex int) error {
	if inline.AdSystem == nil {
		return fmt.Errorf("ad %d: AdSystem is required", adIndex)
	}

	if inline.AdTitle == "" {
		return fmt.Errorf("ad %d: AdTitle is required", adIndex)
	}

	if len(inline.Impression) == 0 {
		return fmt.Errorf("ad %d: at least one Impression is required", adIndex)
	}

	if inline.Creatives == nil || len(inline.Creatives.Creative) == 0 {
		return fmt.Errorf("ad %d: at least one Creative is required", adIndex)
	}

	return nil
}

// validateWrapper validates Wrapper element
func validateWrapper(wrapper *model.Wrapper, adIndex int) error {
	if wrapper.AdSystem == nil {
		return fmt.Errorf("ad %d: AdSystem is required in wrapper", adIndex)
	}

	if wrapper.VASTAdTagURI == "" {
		return fmt.Errorf("ad %d: VASTAdTagURI is required in wrapper", adIndex)
	}

	if len(wrapper.Impression) == 0 {
		return fmt.Errorf("ad %d: at least one Impression is required in wrapper", adIndex)
	}

	return nil
}
