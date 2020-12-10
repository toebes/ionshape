package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/toebes/go-client/onshape"
)

// fileInfo is what is passed from the fileThread to the outputThread for writing
type fileInfo struct {
	Path       string
	OnshapeURL string
	Name       uniqueString
	SKU        uniqueString
	Vendor     uniqueString
	VendorURL  uniqueString
	Checks     string
}

// AddCheck appands to the checks string
func (f *fileInfo) AddCheck(format string, parms ...interface{}) {
	msg := fmt.Sprintf(format, parms...)
	if f.Checks == "" {
		f.Checks = msg
	} else {
		f.Checks += ", " + msg
	}
}

// makefileInfo Creates an empty fileInfo structure
func makefileInfo() fileInfo {
	fi := fileInfo{Name: uniqueString{}, SKU: uniqueString{}, Vendor: uniqueString{}, VendorURL: uniqueString{}}
	return fi
}

// OutputThread is responsible for printing out all the information found
func outputThread(numWorkers int, logfile string, doneQueue chan doneItem, allDone chan bool) {
	running := numWorkers
	baseEntry := 1
	linenum := 0
	lastbase := ""

	outfile, err := os.Create(logfile)
	if err != nil {
		allDone <- true
		log.Fatal(err)
		return
	}
	fmt.Fprintf(outfile, "%v`%v`%v`%v`%v`%v`%v`%v\n",
		"Order",
		"Path",
		"Name",
		"SKU",
		"Vendor",
		"VendorURL",
		"OnshapeURL",
		"Notes")

	orderQueue := make([]*doneItem, 0, 25)
	for {
		output := <-doneQueue
		if output.finished {
			// 			// fmt.Printf("--End: %v\n", output.workerID)
			running--
			if running <= 0 {
				break
			}
		} else {
			// Put the new entry into the queue in the appropriate place.
			if output.order < baseEntry {
				// Something is very wrong.  We have gotten an item out of order.  Just dump it out
				fmt.Printf("***Late Out of order: %v but %v is the base\n", output.order, baseEntry)
			} else {
				idx := output.order - baseEntry
				for len(orderQueue) <= idx {
					orderQueue = append(orderQueue, nil)
				}
				orderQueue[idx] = &output
			}
			// Now see if we have any entries in the queue to output
			toprint := 0
			for toprint < len(orderQueue) && orderQueue[toprint] != nil {
				ent := *orderQueue[toprint]
				if ent.result.Path != lastbase {
					// We don't have to output a separator if it is a directory entry that came from the main thread.
					if ent.workerID != -1 {
						linenum++
						fmt.Fprintf(outfile, "%v`%v\n", linenum, ent.result.Path)
					}
					lastbase = ent.result.Path
				}
				// We have the data, so dump it out
				fmt.Printf("++Output %v(%v): %v`%v`%v`%v`%v`%v`%v\n", ent.order, ent.workerID,
					ent.result.Path,
					ent.result.Name.get(),
					ent.result.SKU.get(),
					ent.result.Vendor.get(),
					ent.result.VendorURL.get(),
					ent.result.OnshapeURL,
					ent.result.Checks)
				linenum++
				fmt.Fprintf(outfile, "%v`%v`%v`%v`%v`%v`%v`%v\n", linenum,
					ent.result.Path,
					ent.result.Name.get(),
					ent.result.SKU.get(),
					ent.result.Vendor.get(),
					ent.result.VendorURL.get(),
					ent.result.OnshapeURL,
					ent.result.Checks)

				toprint++
			}
			if toprint > 0 {
				orderQueue = orderQueue[toprint:]
				baseEntry += toprint
			}
			//		fmt.Printf("--%v(%v): %v\n", output.order, output.workerID, output.result)
		}
	}

	allDone <- true
}

func canFixName(testname string, basename string) bool {
	// Ok the description doesn't match, see if we can fix it automatically
	// "- 2 Pack"
	// "- 25 Pack "
	// "- 4 Pack"
	// "- 4 Pack "
	// "2 Pack"
	fixes := []string{"- 2 Pack", "- 25 Pack ", "- 4 Pack", "- 4 Pack ", "2 Pack", "2 Pack ", "- 2 Pack "}
	for _, fix := range fixes {
		try := strings.Replace(testname, fix, "", -1)
		if strings.EqualFold(try, basename) {
			return true
		}
	}
	return false
}

// queueFile puts a work item on the queue to be processed by one of the fileThreads
func queueFile(workQueue chan workItem, order int, parentPath string, element onshape.BTGlobalTreeMagicNodeInfo) error {
	workQueue <- workItem{order: order, parentPath: parentPath, element: element, finished: false}
	return nil
}

// fileThread is the go thread that takes the file requests that have been queued and processes the file.
// When it is done, the output will be put onto the outputQueue to be handled by a different thread.
// We need to do this because the API can take quite a bit of time to respond with the request
func fileThread(ctx context.Context, client *onshape.APIClient, workerID int, workQueue chan workItem, doneQueue chan doneItem) {
	for {
		// Wait for something to do.  This will either be a file to process or a signal that the queue is done
		// and we are to exit
		request := <-workQueue
		// Are we done?
		if request.finished {
			// If so, then tell the output task that we are ending so they can count us out
			output := doneItem{order: request.order, workerID: workerID, err: nil, result: makefileInfo(), finished: true}
			doneQueue <- output
			break
		}
		// Somethign to do! Let the processFile routine do all the work to gather our result
		result, err := processFile(ctx, client, request.parentPath, request.element)
		if err != nil {
			fmt.Printf("===ERROR (%v):%v/%v\n", err, result.Name, result.SKU)
		}
		output := doneItem{order: request.order, workerID: workerID, err: err, result: result, finished: false}
		doneQueue <- output
	}
}

// processFile Handles an Onshape document
//
func processFile(ctx context.Context, client *onshape.APIClient, parentPath string, element onshape.BTGlobalTreeMagicNodeInfo) (fileInfo, error) {
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
	result := makefileInfo()
	result.Path = parentPath

	// Get the Document ID and default workspace because the APIs need them to access it.
	// They are used both for generating the URL to access the document and the API for getting the Metadata
	did, found := element.GetIdOk()
	if !found {
		return result, fmt.Errorf("unable to get default document id")
	}
	defaultWorkspace, found := element.BTGlobalTreeNodeInfo.GetDefaultWorkspaceOk()
	if !found {
		return result, fmt.Errorf("unable to find default workspace")
	}
	wvid, found := (*defaultWorkspace).GetIdOk()
	if !found {
		return result, fmt.Errorf("unable to get default workspace id")
	}

	foundPiece := false

	// Pull out the name of the document
	documentName, hasName := element.GetNameOk()
	if !hasName {
		*documentName = "<UNNAMED>"
	}
	result.Name.set(*documentName, "MainDocument")

	description, hasit := element.GetDescriptionOk()
	if hasit && *description != "" {
		// We need to parse out the description to confirm that it matches the name and
		// also find the URL for the product
		// In the most likely scenario, the document name SHOULD be the first part of the description followed by a carriage return and then the product URL
		pieces := strings.Split(*description, "\n")
		if len(pieces) != 2 {
			if !strings.Contains(*description, "[OBSOLETE]") && !strings.Contains(*description, "[DISCONTINUED]") {
				result.AddCheck(" Description doesn't have a single carriage return '%v'", strings.ReplaceAll(*description, "\n", "\\n"))
			}
		} else {
			if strings.EqualFold(pieces[0], *documentName) {
				result.VendorURL.set(pieces[1], "Main_Description")
			} else {
				if canFixName(pieces[0], *documentName) {
					err := OnshapeSetDocumentDescription(ctx, client, *did, *documentName+"\n"+pieces[1])
					if err != nil {
						return result, err
					}
				} else {
					if !strings.Contains(*documentName, "(Configurable)") {
						result.AddCheck(" Description '%v' does not match main name", pieces[0])
					}
				}
			}
		}
	}

	// Construct the URL to access the document
	result.OnshapeURL = fmt.Sprintf("https://cad.onshape.com/documents/%v/w/%v", *did, *wvid)
	result.Path = parentPath

	// Get the Metadata for the document.  This returns the list of lower level tabs in the document
	// fmt.Printf("Calling: /api/metadata/d/%v/w/%v/e\n", *did, *wvid)

	var MetadataNodes onshape.BTMetadataInfo
	var rawResp *http.Response
	var err error
	for delay := 0; delay < 50; delay++ {
		MetadataNodes, rawResp, err = client.MetadataApi.GetWMVEsMetadata(ctx, *did, "w", *wvid).Depth("5").Execute()
		// If we are rate limited, implement a backoff strategy
		if err == nil || err.Error() != "429 " {
			break
		}
		fmt.Printf(".......Rate Limited.. Sleeping\n")
		time.Sleep(time.Duration(delay*50) * time.Millisecond)
	}
	if err != nil {
		fmt.Printf("GetWMVEsMetadata error: %v getting %v/w/%v\n", err.Error(), *did, *wvid)
		return result, err
	} else if rawResp != nil && rawResp.StatusCode >= 300 {
		err = fmt.Errorf("err: Response status: %v", rawResp)
		return result, err
	}

	// Make sure we have some items to work with
	items, hasItems := MetadataNodes.GetItemsOk()
	if !hasItems {
		return result, fmt.Errorf("no items found in document")
	}
	// Run through all of the tabs
	for _, subelement := range *items {
		// Figure out what type of tab it is
		elementType, hasElementType := subelement.GetElementTypeOk()
		if !hasElementType {
			return result, fmt.Errorf("Document Metadata doesn't have an elementType")
		}
		tabType, foundType := elementTypeName[int(*elementType)]
		if !foundType {
			tabType = "UNKNOWN"
		}
		// Get the Properties which corresponds to the list of tabs in the document.
		properties, hasProperties := subelement.GetPropertiesOk()
		if !hasProperties {
			fmt.Printf("###%v\n", subelement)
			return result, fmt.Errorf("document Metadata item has no properties")
		}

		// Gather the information that we are going to print out about the tab of the document
		consolidated, err := GetConsolidatedProperties(*properties)
		if err != nil {
			return result, err
		}
		switch tabType {
		case "Part Studio":
			eid, hasEid := subelement.GetElementIdOk()
			if !hasEid {
				result.AddCheck("Element ID is missing")
			}
			parts, hasParts := subelement.GetPartsOk()
			if !strings.EqualFold(*documentName, consolidated.Name) &&
				canFixName(consolidated.Name, *documentName) {
				// We have to rename the part studio to be the proper name
				consolidated.Name = *documentName
			}
			// For a part studio, either the name is something like "parts" or it is the same name as the document
			// If it is the same name as the document, there should be a single part and it should contain the information
			//    about the vendor, not be excluded from BOM
			// if it is not the same name as the document, it should be named PARTS, PARTS DO NOT USE or some such nonesense.
			//    For a part, we want to navigate the individual parts and print out the part information.  It should contain
			//    a derived part called "DO NOT USE PARTS" and all of the parts should have the EXCLUDE FROM BOM flag set.
			if strings.EqualFold(*documentName, consolidated.Name) {
				if foundPiece {
					result.AddCheck("Extra Main Part Studio")
				}
				foundPiece = true
				// Check to make sure that there is only a single part
				if !hasParts {
					result.AddCheck("Part Studio is Empty")
				} else {
					partsItems, hasPartsItems := (*parts).GetItemsOk()
					// TODO: Track consistency of the Exclude from BOM bit
					if hasPartsItems && len(*partsItems) > 0 {
						if len(*partsItems) > 1 {
							result.AddCheck("Part Studio has more than one item")
						}
						for _, part := range *partsItems {
							pid, hasPid := part.GetPartIdOk()
							if !hasPid {
								result.AddCheck("Part ID is missing")
							}
							parttype, hasPartType := part.GetPartTypeOk()
							partProps, hasPartProps := part.GetPropertiesOk()
							href, hasHref := part.GetHrefOk()
							if hasPartType && *parttype == "solid" && hasPartProps {
								partConsolidated, err := GetConsolidatedProperties(*partProps)
								if err != nil {
									return result, err
								}
								result.Name.set(partConsolidated.Name, "PartName")
								result.VendorURL.set(partConsolidated.Description, "PartDescription")
								result.SKU.set(partConsolidated.SKU, "PartSku")
								result.Vendor.set(partConsolidated.Vendor, "PartSku")

								// See if we need to fix the Vendor in this case
								if strings.EqualFold(partConsolidated.Vendor, fixvendor) && partConsolidated.Vendor != fixvendor && hasHref {
									err := SetPartMetadata(ctx, client, *did, "w", *wvid, *eid, *pid, *href, *partProps, "Vendor", fixvendor)
									if err != nil {
										return result, err
									}
								}

								if partConsolidated.ExcludeFromBOM {
									result.AddCheck("Main part is excluded from BOM")
								}
							}
						}
					} else {
						result.AddCheck("Part Studio is missing items")
					}
				}
			} else {
				// Not the main part, so check the name to see if it is something we like
				if !strings.EqualFold(consolidated.Name, "PARTS") &&
					!strings.EqualFold(consolidated.Name, "PARTS DO NOT USE") &&
					!strings.EqualFold(consolidated.Name, "PARTS - DO NOT USE") &&
					!strings.EqualFold(consolidated.Name, "DO NOT USE PARTS") {
					result.AddCheck("Bad Part Studio:\"%v\"", consolidated.Name)
				}
				if hasParts {
					partsItems, hasPartsItems := (*parts).GetItemsOk()
					// TODO: Track consistency of the Exclude from BOM bit
					if hasPartsItems && len(*partsItems) > 0 {
						reportedExclude := false
						foundDoNotUse := false
						for _, part := range *partsItems {
							parttype, hasPartType := part.GetPartTypeOk()
							partProps, hasPartProps := part.GetPropertiesOk()
							if hasPartType && *parttype == "solid" && hasPartProps {
								partConsolidated, err := GetConsolidatedProperties(*partProps)
								if err != nil {
									return result, err
								}

								if !partConsolidated.ExcludeFromBOM {
									if !reportedExclude {
										result.AddCheck("Part not excluded from BOM:\"%v\"", partConsolidated.Name)
									}
									reportedExclude = true
								}
								if strings.EqualFold(partConsolidated.Name, "DO NOT USE PARTS") ||
									strings.EqualFold(partConsolidated.Name, "DO NOT USE THESE PARTS") {
									foundDoNotUse = true
								}
							}
						}
						if !foundDoNotUse {
							result.AddCheck("Do not Use ICON not found")
						}
					}

				} else {
					result.AddCheck(" ExtraPart:")
				}
			}

		case "Assembly":
			// For a legacy assembly we can simply ignore it.
			if strings.Contains(strings.ToUpper(consolidated.Name), "LEGACY") ||
				strings.Contains(strings.ToUpper(consolidated.Name), "DO NOT USE") ||
				strings.Contains(strings.ToUpper(consolidated.Name), "OBSOLETE") ||
				strings.Contains(strings.ToUpper(consolidated.Name), "OLD ASSEMBLY:") {
				// result.AddCheck(" LegacyAsm:\"%v\"", consolidated.Name)
			} else if *documentName == consolidated.Name {
				// This is the assembly intended for the part, so check the SKU, Vendor and Description
				result.SKU.set(consolidated.SKU, "AssemblyPart#")
				result.VendorURL.set(consolidated.Description, "AssemblyDesc")
				result.Vendor.set(consolidated.Vendor, "Assembly")
				// See if we need to fix the Vendor in this case
				if strings.EqualFold(consolidated.Vendor, fixvendor) && consolidated.Vendor != fixvendor {
					href, hasHref := subelement.GetHrefOk()
					if hasHref {
						err := SetMetadata(ctx, client, *did, "w", *wvid, *href, *properties, "Vendor", fixvendor)
						if err != nil {
							fmt.Printf("SetMetadata Assembly error\n")
							return result, err
						}
					}
				}
				foundPiece = true
			} else {
				// The name doesn't match, so just note it in the checks
				result.AddCheck(" ExtraAssembly:\"%v\"", consolidated.Name)

			}

		case "BLOB":
		default:
			// We can ignore the tab
		}

	}
	if !foundPiece {
		result.AddCheck(" NoMainPieceFound")
	}
	return result, err
}
