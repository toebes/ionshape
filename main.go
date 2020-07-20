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

type folderStack struct {
	queue *list.List
}

func (c *folderStack) Push(value string) {
	c.queue.PushFront(value)
}

func (c *folderStack) Pop() (val string, err error) {
	val, err = c.Front()

	if err == nil {
		ele := c.queue.Front()
		c.queue.Remove(ele)
	}
	return
}

func (c *folderStack) Front() (string, error) {
	if c.queue.Len() > 0 {
		if val, ok := c.queue.Front().Value.(string); ok {
			return val, nil
		}
		return "", fmt.Errorf("Peep Error: Queue Datatype is incorrect")
	}
	return "", fmt.Errorf("Peep Error: Queue is empty")
}

func (c *folderStack) Size() int {
	return c.queue.Len()
}

func (c *folderStack) isEmpty() bool {
	return c.queue.Len() == 0
}

// processFile Handles an Onshape document
// https://cad.onshape.com/api/metadata/d/7508e0d58196a1ff1c86c951/w/3af033a2075cf9db20d26f84/e?configuration=default
// GetWVMetadata(ctx, did, wv, wvid).Execute()
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

	documentName, hasName := element.GetNameOk()
	if !hasName {
		*documentName = "<UNNAMED>"
	}

	url := fmt.Sprintf("https://cad.onshape.com/documents/%v/w/%v", *did, *wvid)
	fmt.Printf("+++%v`%v`%v\n", parentPath, *documentName, url)

	// fmt.Printf("Calling: /api/metadata/d/%v/w/%v/e\n", *did, *wvid)
	MetadataNodes, rawResp, err := client.MetadataApi.GetWMVEsMetadata(ctx, *did, "w", *wvid).Execute()
	if err != nil || (rawResp != nil && rawResp.StatusCode >= 300) {
		fmt.Printf("err: %v  -- Response status: %v\n", err, rawResp)
		return err
	}
	//
	items, hasItems := MetadataNodes.GetItemsOk()
	if !hasItems {
		return fmt.Errorf("no items found in document")
	}
	for _, subelement := range *items {
		elementType, hasElementType := subelement.GetElementTypeOk()
		if !hasElementType {
			return fmt.Errorf("Document Metadata doesn't have an elementType")
		}
		tabType, foundType := elementTypeName[int(*elementType)]
		if !foundType {
			tabType = "UNKNOWN"
		}
		tabName := ""
		tabExtra := ""
		tabExtraSep := ""
		tabDescription := ""
		tabSKU := ""
		tabVendor := ""

		properties, hasProperties := subelement.GetPropertiesOk()
		// fmt.Printf("Result:%v - %v\n", properties, hasProperties)
		if !hasProperties {
			fmt.Printf("###%v\n", subelement)
			return fmt.Errorf("document Metadata item has no properties")
		}
		for _, property := range *properties {
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
