package main

import (
	"container/list"
	"context"
	"fmt"
	"net/http"

	"github.com/toebes/go-client/onshape"
)

// processFolders traverses the folder hierarchy and performs the actions on it
func processFolders(ctx context.Context, client *onshape.APIClient) error {
	folderQueue := &folderStack{
		queue: list.New(),
	}
	if len(folderIDs) > 0 {
		for _, id := range folderIDs {
			folderQueue.Push(id)
		}
	} else {
		// We will start with the magic tree root for "My Onshape"
		folderQueue.Push("1")
	}
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
	return nil
}
