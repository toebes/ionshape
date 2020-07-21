package main

import (
	"context"
	"fmt"

	"github.com/toebes/go-client/onshape"
)

// processFile Handles an Onshape document
//
func processFile(ctx context.Context, client *onshape.APIClient, parentPath string, element onshape.BTGlobalTreeMagicNodeInfo) error {
	var elementTypeName = map[int]string{
		0: "Part Studio",
		1: "Assembly",
		2: "Drawing",
		3: "Feature Studio",
		4: "BLOB",
		5: "Aplication",
		6: "Table",
		7: "BOM",
	}
	// Get the Document ID and default workspace because the APIs need them to access it.
	// They are used both for generating the URL to access the document and the API for getting the Metadata
	did, found := element.GetIdOk()
	if !found {
		return fmt.Errorf("unable to get default document id")
	}
	defaultWorkspace, found := element.BTGlobalTreeNodeInfo.GetDefaultWorkspaceOk()
	if !found {
		return fmt.Errorf("unable to find default workspace")
	}
	wvid, found := (*defaultWorkspace).GetIdOk()
	if !found {
		return fmt.Errorf("unable to get default workspace id")
	}

	// Pull out the name of the document
	documentName, hasName := element.GetNameOk()
	if !hasName {
		*documentName = "<UNNAMED>"
	}

	// Construct the URL to access the document
	url := fmt.Sprintf("https://cad.onshape.com/documents/%v/w/%v", *did, *wvid)

	// Log the document
	fmt.Printf("+++%v`%v`%v\n", parentPath, *documentName, url)

	// Get the Metadata for the document.  This returns the list of lower level tabs in the document
	// fmt.Printf("Calling: /api/metadata/d/%v/w/%v/e\n", *did, *wvid)
	MetadataNodes, rawResp, err := client.MetadataApi.GetWMVEsMetadata(ctx, *did, "w", *wvid).Depth("5").Execute()
	if err != nil || (rawResp != nil && rawResp.StatusCode >= 300) {
		fmt.Printf("err: %v  -- Response status: %v\n", err, rawResp)
		return err
	}
	// Make sure we have some items to work with
	items, hasItems := MetadataNodes.GetItemsOk()
	if !hasItems {
		return fmt.Errorf("no items found in document")
	}
	// Run through all of the tabs
	for _, subelement := range *items {
		// Figure out what type of tab it is
		elementType, hasElementType := subelement.GetElementTypeOk()
		if !hasElementType {
			return fmt.Errorf("Document Metadata doesn't have an elementType")
		}
		tabType, foundType := elementTypeName[int(*elementType)]
		if !foundType {
			tabType = "UNKNOWN"
		}
		// Get the Properties which corresponds to the list of tabs in the document.
		properties, hasProperties := subelement.GetPropertiesOk()
		if !hasProperties {
			fmt.Printf("###%v\n", subelement)
			return fmt.Errorf("document Metadata item has no properties")
		}

		// Gather the information that we are going to print out about the document
		consolidated, err := GetConsolidatedProperties(*properties)
		if err != nil {
			return err
		}
		fmt.Printf("  -- %v`%v`%v`%v`%v`%v\n", tabType, consolidated.Name, consolidated.Description, consolidated.SKU, consolidated.Vendor, consolidated.Extras)
		// For a part, we want to navigate the individual parts and print out the part information
		if *elementType == 0 {
			parts, hasParts := subelement.GetPartsOk()
			if hasParts {
				partsItems, hasPartsItems := (*parts).GetItemsOk()
				// TODO: Track consistency of the Exclude from BOM bit
				if hasPartsItems && len(*partsItems) > 0 {
					for partCount, part := range *partsItems {
						parttype, hasPartType := part.GetPartTypeOk()
						partProps, hasPartProps := part.GetPropertiesOk()
						if hasPartType && *parttype == "solid" && hasPartProps {
							partConsolidated, err := GetConsolidatedProperties(*partProps)
							if err != nil {
								return err
							}
							// Only print out data for the first 10 parts
							if partCount < 10 {
								fmt.Printf("    - %v`%v`%v`%v`%v\n", partConsolidated.Name, partConsolidated.Description, partConsolidated.SKU, partConsolidated.Vendor, partConsolidated.Extras)
							} else if partCount == 10 {
								fmt.Printf("    - Skipping next %v parts\n", (len(*partsItems) - partCount))
							}
						}
					}
				}
			}
		}
	}
	return err
}
