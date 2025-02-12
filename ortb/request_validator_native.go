package ortb

import (
	"encoding/json"
	"fmt"

	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/native1"
	nativeRequests "github.com/prebid/openrtb/v20/native1/request"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

// fillAndValidateNative validates the request, and assigns the Asset IDs as recommended by the Native v1.2 spec.
func fillAndValidateNative(n *openrtb2.Native, impIndex int) error {
	if n == nil {
		return nil
	}

	if len(n.Request) == 0 {
		return fmt.Errorf("request.imp[%d].native missing required property \"request\"", impIndex)
	}
	var nativePayload nativeRequests.Request
	if err := jsonutil.UnmarshalValid(json.RawMessage(n.Request), &nativePayload); err != nil {
		return err
	}

	if err := validateNativeContextTypes(nativePayload.Context, nativePayload.ContextSubType, impIndex); err != nil {
		return err
	}
	if err := validateNativePlacementType(nativePayload.PlcmtType, impIndex); err != nil {
		return err
	}
	if err := fillAndValidateNativeAssets(nativePayload.Assets, impIndex); err != nil {
		return err
	}
	if err := validateNativeEventTrackers(nativePayload.EventTrackers, impIndex); err != nil {
		return err
	}

	serialized, err := jsonutil.Marshal(nativePayload)
	if err != nil {
		return err
	}
	n.Request = string(serialized)
	return nil
}

func validateNativeContextTypes(cType native1.ContextType, cSubtype native1.ContextSubType, impIndex int) error {
	if cType == 0 {
		// Context is only recommended, so none is a valid type.
		return nil
	}
	if cType < native1.ContextTypeContent || (cType > native1.ContextTypeProduct && cType < openrtb_ext.NativeExchangeSpecificLowerBound) {
		return fmt.Errorf("request.imp[%d].native.request.context is invalid. See https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=39", impIndex)
	}
	if cSubtype < 0 {
		return fmt.Errorf("request.imp[%d].native.request.contextsubtype value can't be less than 0. See https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=39", impIndex)
	}
	if cSubtype == 0 {
		return nil
	}
	if cSubtype >= native1.ContextSubTypeGeneral && cSubtype <= native1.ContextSubTypeUserGenerated {
		if cType != native1.ContextTypeContent && cType < openrtb_ext.NativeExchangeSpecificLowerBound {
			return fmt.Errorf("request.imp[%d].native.request.context is %d, but contextsubtype is %d. This is an invalid combination. See https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=39", impIndex, cType, cSubtype)
		}
		return nil
	}
	if cSubtype >= native1.ContextSubTypeSocial && cSubtype <= native1.ContextSubTypeChat {
		if cType != native1.ContextTypeSocial && cType < openrtb_ext.NativeExchangeSpecificLowerBound {
			return fmt.Errorf("request.imp[%d].native.request.context is %d, but contextsubtype is %d. This is an invalid combination. See https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=39", impIndex, cType, cSubtype)
		}
		return nil
	}
	if cSubtype >= native1.ContextSubTypeSelling && cSubtype <= native1.ContextSubTypeProductReview {
		if cType != native1.ContextTypeProduct && cType < openrtb_ext.NativeExchangeSpecificLowerBound {
			return fmt.Errorf("request.imp[%d].native.request.context is %d, but contextsubtype is %d. This is an invalid combination. See https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=39", impIndex, cType, cSubtype)
		}
		return nil
	}
	if cSubtype >= openrtb_ext.NativeExchangeSpecificLowerBound {
		return nil
	}

	return fmt.Errorf("request.imp[%d].native.request.contextsubtype is invalid. See https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=39", impIndex)
}

func validateNativePlacementType(pt native1.PlacementType, impIndex int) error {
	if pt == 0 {
		// Placement Type is only recommended, not required.
		return nil
	}
	if pt < native1.PlacementTypeFeed || (pt > native1.PlacementTypeRecommendationWidget && pt < openrtb_ext.NativeExchangeSpecificLowerBound) {
		return fmt.Errorf("request.imp[%d].native.request.plcmttype is invalid. See https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=40", impIndex)
	}
	return nil
}

func fillAndValidateNativeAssets(assets []nativeRequests.Asset, impIndex int) error {
	if len(assets) < 1 {
		return fmt.Errorf("request.imp[%d].native.request.assets must be an array containing at least one object", impIndex)
	}

	assetIDs := make(map[int64]struct{}, len(assets))

	// If none of the asset IDs are defined by the caller, then prebid server should assign its own unique IDs. But
	// if the caller did assign its own asset IDs, then prebid server will respect those IDs
	assignAssetIDs := true
	for i := 0; i < len(assets); i++ {
		assignAssetIDs = assignAssetIDs && (assets[i].ID == 0)
	}

	for i := 0; i < len(assets); i++ {
		if err := validateNativeAsset(assets[i], impIndex, i); err != nil {
			return err
		}

		if assignAssetIDs {
			assets[i].ID = int64(i)
			continue
		}

		// Each asset should have a unique ID thats assigned by the caller
		if _, ok := assetIDs[assets[i].ID]; ok {
			return fmt.Errorf("request.imp[%d].native.request.assets[%d].id is already being used by another asset. Each asset ID must be unique.", impIndex, i)
		}

		assetIDs[assets[i].ID] = struct{}{}
	}

	return nil
}

func validateNativeAsset(asset nativeRequests.Asset, impIndex int, assetIndex int) error {
	assetErr := "request.imp[%d].native.request.assets[%d] must define exactly one of {title, img, video, data}"
	foundType := false

	if asset.Title != nil {
		foundType = true
		if err := validateNativeAssetTitle(asset.Title, impIndex, assetIndex); err != nil {
			return err
		}
	}

	if asset.Img != nil {
		if foundType {
			return fmt.Errorf(assetErr, impIndex, assetIndex)
		}
		foundType = true
		if err := validateNativeAssetImage(asset.Img, impIndex, assetIndex); err != nil {
			return err
		}
	}

	if asset.Video != nil {
		if foundType {
			return fmt.Errorf(assetErr, impIndex, assetIndex)
		}
		foundType = true
		if err := validateNativeAssetVideo(asset.Video, impIndex, assetIndex); err != nil {
			return err
		}
	}

	if asset.Data != nil {
		if foundType {
			return fmt.Errorf(assetErr, impIndex, assetIndex)
		}
		foundType = true
		if err := validateNativeAssetData(asset.Data, impIndex, assetIndex); err != nil {
			return err
		}
	}

	if !foundType {
		return fmt.Errorf(assetErr, impIndex, assetIndex)
	}

	return nil
}

func validateNativeEventTrackers(trackers []nativeRequests.EventTracker, impIndex int) error {
	for i := 0; i < len(trackers); i++ {
		if err := validateNativeEventTracker(trackers[i], impIndex, i); err != nil {
			return err
		}
	}
	return nil
}

func validateNativeAssetTitle(title *nativeRequests.Title, impIndex int, assetIndex int) error {
	if title.Len < 1 {
		return fmt.Errorf("request.imp[%d].native.request.assets[%d].title.len must be a positive number", impIndex, assetIndex)
	}
	return nil
}

func validateNativeEventTracker(tracker nativeRequests.EventTracker, impIndex int, eventIndex int) error {
	if tracker.Event < native1.EventTypeImpression || (tracker.Event > native1.EventTypeViewableVideo50 && tracker.Event < openrtb_ext.NativeExchangeSpecificLowerBound) {
		return fmt.Errorf("request.imp[%d].native.request.eventtrackers[%d].event is invalid. See section 7.6: https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=43", impIndex, eventIndex)
	}
	if len(tracker.Methods) < 1 {
		return fmt.Errorf("request.imp[%d].native.request.eventtrackers[%d].method is required. See section 7.7: https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=43", impIndex, eventIndex)
	}
	for methodIndex, method := range tracker.Methods {
		if method < native1.EventTrackingMethodImage || (method > native1.EventTrackingMethodJS && method < openrtb_ext.NativeExchangeSpecificLowerBound) {
			return fmt.Errorf("request.imp[%d].native.request.eventtrackers[%d].methods[%d] is invalid. See section 7.7: https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=43", impIndex, eventIndex, methodIndex)
		}
	}

	return nil
}

func validateNativeAssetImage(img *nativeRequests.Image, impIndex int, assetIndex int) error {
	if img.W < 0 {
		return fmt.Errorf("request.imp[%d].native.request.assets[%d].img.w must be a positive integer", impIndex, assetIndex)
	}
	if img.H < 0 {
		return fmt.Errorf("request.imp[%d].native.request.assets[%d].img.h must be a positive integer", impIndex, assetIndex)
	}
	if img.WMin < 0 {
		return fmt.Errorf("request.imp[%d].native.request.assets[%d].img.wmin must be a positive integer", impIndex, assetIndex)
	}
	if img.HMin < 0 {
		return fmt.Errorf("request.imp[%d].native.request.assets[%d].img.hmin must be a positive integer", impIndex, assetIndex)
	}
	return nil
}

func validateNativeAssetVideo(video *nativeRequests.Video, impIndex int, assetIndex int) error {
	if len(video.MIMEs) < 1 {
		return fmt.Errorf("request.imp[%d].native.request.assets[%d].video.mimes must be an array with at least one MIME type", impIndex, assetIndex)
	}
	if video.MinDuration < 1 {
		return fmt.Errorf("request.imp[%d].native.request.assets[%d].video.minduration must be a positive integer", impIndex, assetIndex)
	}
	if video.MaxDuration < 1 {
		return fmt.Errorf("request.imp[%d].native.request.assets[%d].video.maxduration must be a positive integer", impIndex, assetIndex)
	}
	if err := validateNativeVideoProtocols(video.Protocols, impIndex, assetIndex); err != nil {
		return err
	}

	return nil
}

func validateNativeAssetData(data *nativeRequests.Data, impIndex int, assetIndex int) error {
	if data.Type < native1.DataAssetTypeSponsored || (data.Type > native1.DataAssetTypeCTAText && data.Type < 500) {
		return fmt.Errorf("request.imp[%d].native.request.assets[%d].data.type is invalid. See section 7.4: https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=40", impIndex, assetIndex)
	}

	return nil
}

func validateNativeVideoProtocols(protocols []adcom1.MediaCreativeSubtype, impIndex int, assetIndex int) error {
	if len(protocols) < 1 {
		return fmt.Errorf("request.imp[%d].native.request.assets[%d].video.protocols must be an array with at least one element", impIndex, assetIndex)
	}
	for i := 0; i < len(protocols); i++ {
		if err := validateNativeVideoProtocol(protocols[i], impIndex, assetIndex, i); err != nil {
			return err
		}
	}
	return nil
}

func validateNativeVideoProtocol(protocol adcom1.MediaCreativeSubtype, impIndex int, assetIndex int, protocolIndex int) error {
	if protocol < adcom1.CreativeVAST10 || protocol > adcom1.CreativeDAAST10Wrapper {
		return fmt.Errorf("request.imp[%d].native.request.assets[%d].video.protocols[%d] is invalid. See Section 5.8: https://www.iab.com/wp-content/uploads/2016/03/OpenRTB-API-Specification-Version-2-5-FINAL.pdf#page=52", impIndex, assetIndex, protocolIndex)
	}
	return nil
}
