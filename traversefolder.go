package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/toebes/go-client/onshape"
)

// OnshapeDocumentCallback is called to process a document.
type OnshapeDocumentCallback func(ctx context.Context, client *onshape.APIClient, parentPath string, element onshape.BTGlobalTreeMagicNodeInfo) error

// OnshapeFolderCallback is called to process a folder
type OnshapeFolderCallback func(ctx context.Context, client *onshape.APIClient, parentPath string, fid string) error

// OnshapeTraverseFolder traverses the folder hierarchy and performs the actions on it
func OnshapeTraverseFolder(ctx context.Context, client *onshape.APIClient, fid string, docCallback OnshapeDocumentCallback, folderCallBack OnshapeFolderCallback) error {
	// Iterate throught he heirarchy pulling 50 entries at a time.
	offset := int32(0)
	limit := int32(50)
	for {
		var appGlobalTreeNodes onshape.BTGlobalTreeNodesInfo
		var rawResp *http.Response
		var err error
		if len(fid) < 4 {
			appGlobalTreeNodes, rawResp, err = client.GlobalTreeNodesApi.GlobalTreeNodesMagic(ctx, fid).GetPathToRoot(true).Offset(offset).Limit(limit).SortColumn("name").SortOrder("desc").Execute()
		} else {
			appGlobalTreeNodes, rawResp, err = client.GlobalTreeNodesApi.GlobalTreeNodesFolder(ctx, fid).GetPathToRoot(true).Offset(offset).Limit(limit).SortColumn("name").SortOrder("desc").Execute()
		}

		if err != nil {
			return err
		} else if rawResp != nil && rawResp.StatusCode >= 300 {
			return fmt.Errorf("err: Response status: %v", rawResp)
		} else {
			items, hasItems := appGlobalTreeNodes.GetItemsOk()
			// Make sure there is something in the folder to process
			if hasItems {
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
				// Process all of the elements returned for the folder
				for _, element := range *items {
					id, hasID := element.GetIdOk()
					isContainer, hasContainer := element.GetIsContainerOk()
					if hasContainer && *isContainer {
						if hasID {
							foldername, hasname := element.GetNameOk()
							folderpath := parentPath
							if hasname {
								folderpath += " > " + *foldername
							}
							err := folderCallBack(ctx, client, folderpath, *id)
							if err != nil {
								return err
							}
							// } else {
							// 	fmt.Printf("*** Did not get an ID to queue\n")
						}
					} else {
						err := docCallback(ctx, client, parentPath, element)
						if err != nil {
							return err
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
	return nil
}
