package main

import (
	"container/list"
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/toebes/go-client/onshape"
)

// folderStack is used to maintain a queue of folders to process
// It is used as a LIFO stack so that we can end up doing an in-order traversal
type folderStack struct {
	queue *list.List
}

// Push puts an entry at the top of the stack
func (c *folderStack) Push(value string) {
	c.queue.PushFront(value)
}

// Pop removes the entry from the top of the stack
func (c *folderStack) Pop() (val string, err error) {
	val, err = c.Front()

	if err == nil {
		ele := c.queue.Front()
		c.queue.Remove(ele)
	}
	return
}

// Front finds the first entry in the stack
func (c *folderStack) Front() (string, error) {
	if c.queue.Len() > 0 {
		if val, ok := c.queue.Front().Value.(string); ok {
			return val, nil
		}
		return "", fmt.Errorf("Peep Error: Queue Datatype is incorrect")
	}
	return "", fmt.Errorf("Peep Error: Queue is empty")
}

// Size tells us how many entries there are
func (c *folderStack) Size() int {
	return c.queue.Len()
}

// isEmpty tells when there is nothing left on the stack
func (c *folderStack) isEmpty() bool {
	return c.queue.Len() == 0
}

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
	MetadataNodes, rawResp, err := client.MetadataApi.GetWMVEsMetadata(ctx, *did, "w", *wvid).Execute()
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
		// Gather the information that we are going to print out about the document
		tabName := ""
		tabExtra := ""
		tabExtraSep := ""
		tabDescription := ""
		tabSKU := ""
		tabVendor := ""

		// Get the Properties which corresponds to the list of tabs in the document.
		properties, hasProperties := subelement.GetPropertiesOk()
		if !hasProperties {
			fmt.Printf("###%v\n", subelement)
			return fmt.Errorf("document Metadata item has no properties")
		}
		// Iterate over all the elements in the document.
		for _, property := range *properties {
			// This is where we have a bit of challenge with the API.  We have to determine the type
			// of the property in order to access the correct polymorhpic structure of data
			metadataType := property.BTMetadataItemsPropertiesInterface.GetValueType()
			propName := "<NONAME>"
			propValue := "<NOVALUE>"
			showProp := false
			switch metadataType {
			case "BOOL":
				propIface := property.BTMetadataItemsPropertiesInterface.(*onshape.BTMetadataCommonBool)
				name, hasName := propIface.GetNameOk()
				if hasName {
					propName = *name
					pval, hasPval := propIface.GetValueOk()
					if hasPval {
						propValue = strconv.FormatBool(*pval)
						showProp = true
					}
				}
			case "CATEGORY":
				propIface := property.BTMetadataItemsPropertiesInterface.(*onshape.BTMetadataCommonCategory)
				name, hasName := propIface.GetNameOk()
				if hasName {
					propName = *name
					_ /*pval*/, hasPval := propIface.GetValueOk()
					if hasPval {
						propValue = "NEED TO HANDLE CATEGORY"
					}
				}
			case "DATE":
				propIface := property.BTMetadataItemsPropertiesInterface.(*onshape.BTMetadataCommonDate)
				name, hasName := propIface.GetNameOk()
				if hasName {
					propName = *name
					pval, hasPval := propIface.GetValueOk()
					if hasPval {
						propValue = (*pval).Format("Mon Jan _2 15:04:05 2006")
						showProp = true
					}
				}
			case "ENUM":
				propIface := property.BTMetadataItemsPropertiesInterface.(*onshape.BTMetadataCommonEnum)
				name, hasName := propIface.GetNameOk()
				if hasName {
					propName = *name
					pval, hasPval := propIface.GetValueOk()
					if hasPval {
						propValue = *pval
						showProp = true
					}
				}
			case "STRING":
				propIface := property.BTMetadataItemsPropertiesInterface.(*onshape.BTMetadataCommonString)
				name, hasName := propIface.GetNameOk()
				if hasName {
					propName = *name
					pval, hasPval := propIface.GetValueOk()
					if hasPval {
						propValue = *pval
						showProp = true
					}
				}
			case "USER":
				propIface := property.BTMetadataItemsPropertiesInterface.(*onshape.BTMetadataCommonUser)
				name, hasName := propIface.GetNameOk()
				if hasName {
					propName = *name
					pval, hasPval := propIface.GetValueOk()
					if hasPval && len(*pval) > 0 {
						userName, hasUserName := ((*pval)[0]).GetNameOk()
						if hasUserName {
							propValue = *userName
							showProp = true
						}
					}
				}
			default:
			}
			if showProp {
				switch propName {
				case "Name":
					tabName = propValue
				// Fields we want to rember if they are set
				case "Exclude from BOM":
					fallthrough
				case "Not revision managed":
					if propValue == "true" {
						tabExtra += tabExtraSep + propName
						tabExtraSep = ", "
					}
				case "Description":
					tabDescription = propValue
				case "Part number":
					tabSKU = propValue
				case "Vendor":
					tabVendor = propValue
					// Fields we can safely ignore
				case "State":
				case "Unit of measure":
				case "Drawn by":
				case "Last changed date":
				case "Date drawn":
				case "Last changed by":
				default:
					tabExtra += tabExtraSep + "[" + propName + "=" + propValue + "]"
					tabExtraSep = ", "
				}

			}
		}
		fmt.Printf("  -- %v`%v`%v`%v`%v`%v\n", tabType, tabName, tabDescription, tabSKU, tabVendor, tabExtra)
	}
	return err
}

func main() {
	config := onshape.NewConfiguration()
	//config.Debug = true

	folderQueue := &folderStack{
		queue: list.New(),
	}
	client := onshape.NewAPIClient(config)
	testSecretKey := os.Getenv("ONSHAPE_API_SECRET_KEY")
	testAccessKey := os.Getenv("ONSHAPE_API_ACCESS_KEY")
	fmt.Printf("Keys: Secret=%v Access=%v\n", testSecretKey, testAccessKey)

	ctx := context.WithValue(context.Background(), onshape.ContextAPIKeys, onshape.APIKeys{SecretKey: testSecretKey, AccessKey: testAccessKey})

	// We will start with the magic tree root for "My Onshape"
	folderQueue.Push("1")
done:
	for folderQueue.Size() > 0 {
		folderid, err := folderQueue.Pop()
		if err != nil {
			fmt.Printf("Something broke with the queue: %v\n", err)
			break
		}

		// Iterate throught he heirarchy pulling 50 entries at a time.
		offset := int32(0)
		limit := int32(50)
		for {
			var appGlobalTreeNodes onshape.BTGlobalTreeNodesInfo
			var rawResp *http.Response
			var err error
			if folderid == "1" {
				appGlobalTreeNodes, rawResp, err = client.GlobalTreeNodesApi.GlobalTreeNodesMagic(ctx, folderid).GetPathToRoot(true).Offset(offset).Limit(limit).SortColumn("name").SortOrder("desc").Execute()
			} else {
				appGlobalTreeNodes, rawResp, err = client.GlobalTreeNodesApi.GlobalTreeNodesFolder(ctx, folderid).GetPathToRoot(true).Offset(offset).Limit(limit).SortColumn("name").SortOrder("desc").Execute()
			}

			if err != nil || (rawResp != nil && rawResp.StatusCode >= 300) {
				fmt.Printf("err: %v  -- Response status: %v\n", err, rawResp)
			} else {
				// First we need to get the path to this item
				pathToRoot, hasPathToRoot := appGlobalTreeNodes.GetPathToRootOk()
				parentPath := ""
				extra := ""
				if hasPathToRoot {
					for _, element := range *pathToRoot {
						name, hasName := element.GetNameOk()
						if !hasName {
							*name = "<NONAME>"
						}
						parentPath = *name + extra + parentPath
						extra = " > "
					}
				}
				items, hasItems := appGlobalTreeNodes.GetItemsOk()
				if hasItems {
					for _, element := range *items {
						id, hasID := element.GetIdOk()
						isContainer, hasContainer := element.GetIsContainerOk()
						// itemType := ""
						if hasContainer && *isContainer {
							// itemType = "/"
							if hasID {
								folderQueue.Push(*id)
							} else {
								fmt.Printf("*** Did not get an ID to queue\n")
							}
						} else {
							err := processFile(ctx, client, parentPath, element)
							if err != nil {
								fmt.Printf("err: %v\n", err)
								continue done
							}
						}
					}
				}
				next, hasNext := appGlobalTreeNodes.GetNextOk()
				if hasNext && *next != "" {
					offset += limit
				} else {
					break
				}
			}
		}
	}
}
