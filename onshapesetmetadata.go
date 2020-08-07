package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/toebes/go-client/onshape"
)

//java.util.concurrent.ExecutionException: com.belmonttech.service.exception.BTBadRequestException
///metadata/d/:did/[wv]/:wv/e/:eid/p/:pid

//
// SetPartMetadata allows updating a metadata property
func SetPartMetadata(ctx context.Context, client *onshape.APIClient, did string, wv string, wvid string, eid string, pid string, href string, partProps []onshape.BTMetadataItemsProperties, field string, value interface{}) error {
	body, err := GenMetadataSetBody(partProps, field, value)
	if err == nil {
		// Needs to be:
		//   {
		//   	"items": [{
		// 			"href": "https://cad.onshape.com/api/metadata/d/b1a86c597f35c9390a3a56d1/w/058cc895a12a6ac5080df4a8/e/8f4658a621977488c28508bd?configuration=default",
		// 			"properties":[{
		// 				"propertyId": "57f3fb8efa3416c06701d612",
		// 				"value": "goBILDA"}
		// 	  			]
		// 	  		}]
		//   }
		items := map[string]interface{}{"href": href, "properties": []interface{}{body}}
		propBody := map[string]interface{}{"items": []interface{}{items}}
		jsonBody, err := json.Marshal(propBody)
		fmt.Printf("JsonBody: %v\n", string(jsonBody))
		if err == nil {
			MetadataNodes, rawResp, err := client.MetadataApi.UpdateWVEPMetadata(ctx, did, wv, wvid, eid, pid, "").Body(string(jsonBody)).Execute()
			if err == nil && rawResp != nil && rawResp.StatusCode >= 300 {
				err = fmt.Errorf("err: Response status: %v", rawResp)
			}
			fmt.Printf("Result: %v\n", MetadataNodes)
		}
	}
	// But currently is:
	//  {"propertyID":"57f3fb8efa3416c06701d612","value":"goBILDA"}
	return err

}

// SetMetadata allows updating a metadata property
func SetMetadata(ctx context.Context, client *onshape.APIClient, did string, wv string, wvid string, href string, partProps []onshape.BTMetadataItemsProperties, field string, value interface{}) error {
	body, err := GenMetadataSetBody(partProps, field, value)
	if err == nil {
		// the body actually has to be an array
		items := map[string]interface{}{"href": href, "properties": []interface{}{body}}
		propBody := map[string]interface{}{"items": []interface{}{items}}
		jsonBody, err := json.Marshal(propBody)
		fmt.Printf("JsonBody: %v\n", string(jsonBody))
		if err == nil {
			MetadataNodes, rawResp, err := client.MetadataApi.UpdateWVMetadata(ctx, did, wv, wvid).Body(string(jsonBody)).Execute()
			if err == nil && rawResp != nil && rawResp.StatusCode >= 300 {
				err = fmt.Errorf("err: Response status: %v", rawResp)
			}
			fmt.Printf("Result: %v\n", MetadataNodes)
		}
	}
	return err

}

// GenMetadataSetBody generates
func GenMetadataSetBody(partProps []onshape.BTMetadataItemsProperties, field string, value interface{}) (result interface{}, err error) {
	// We need to know the propertyID

	// {
	// 	"items": [{
	// 	"href": "https://cad.onshape.com/api/metadata/d/b1a86c597f35c9390a3a56d1/w/058cc895a12a6ac5080df4a8/e/8f4658a621977488c28508bd?configuration=default",
	// 	"properties":[{
	// 	"propertyId": "57f3fb8efa3416c06701d612",
	// 	"value": "goBILDA"}
	// 	  ]
	// 	  }]
	//   }

	result = nil
	err = fmt.Errorf("Unable to find propertyID for field '%v'", field)
	// Iterate over all the elements in the document.
	for _, metadataItem := range partProps {
		// We need to cast the type to a common type in order to get the name
		metadataType := metadataItem.BTMetadataItemsPropertiesInterface.GetValueType()
		var name *string
		var propertyID *string
		name = nil
		propertyID = nil
		hasName := false
		hasPropertyID := false

		switch metadataType {
		case "BOOL":
			propIface := metadataItem.BTMetadataItemsPropertiesInterface.(*onshape.BTMetadataCommonBool)
			name, hasName = propIface.GetNameOk()
			propertyID, hasPropertyID = propIface.GetPropertyIdOk()

		case "CATEGORY":
			propIface := metadataItem.BTMetadataItemsPropertiesInterface.(*onshape.BTMetadataCommonCategory)
			name, hasName = propIface.GetNameOk()
			propertyID, hasPropertyID = propIface.GetPropertyIdOk()

		case "COMPUTED":
			propIface := metadataItem.BTMetadataItemsPropertiesInterface.(*onshape.BTMetadataCommonComputed)
			name, hasName = propIface.GetNameOk()
			propertyID, hasPropertyID = propIface.GetPropertyIdOk()

		case "DATE":
			propIface := metadataItem.BTMetadataItemsPropertiesInterface.(*onshape.BTMetadataCommonDate)
			name, hasName = propIface.GetNameOk()
			propertyID, hasPropertyID = propIface.GetPropertyIdOk()

		case "ENUM":
			propIface := metadataItem.BTMetadataItemsPropertiesInterface.(*onshape.BTMetadataCommonEnum)
			name, hasName = propIface.GetNameOk()
			propertyID, hasPropertyID = propIface.GetPropertyIdOk()

		case "OBJECT":
			propIface := metadataItem.BTMetadataItemsPropertiesInterface.(*onshape.BTMetadataCommonObject)
			name, hasName = propIface.GetNameOk()
			propertyID, hasPropertyID = propIface.GetPropertyIdOk()

		case "STRING":
			propIface := metadataItem.BTMetadataItemsPropertiesInterface.(*onshape.BTMetadataCommonString)
			name, hasName = propIface.GetNameOk()
			propertyID, hasPropertyID = propIface.GetPropertyIdOk()

		case "USER":
			propIface := metadataItem.BTMetadataItemsPropertiesInterface.(*onshape.BTMetadataCommonUser)
			name, hasName = propIface.GetNameOk()
			propertyID, hasPropertyID = propIface.GetPropertyIdOk()

		default:
		}

		if hasName && hasPropertyID && *name == field {
			// Ok they have the field that they want to
			err = nil
			result = map[string]interface{}{"propertyId": *propertyID, "value": value}
			return
		}
	}
	// We didn't find it, so skip out with the default error
	return
}
