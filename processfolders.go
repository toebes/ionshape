package main

import (
	"container/list"
	"context"
	"fmt"

	"github.com/toebes/go-client/onshape"
)

// processFolders traverses the folder hierarchy and performs the actions on it
func processFolders(ctx context.Context, client *onshape.APIClient, workQueue chan workItem, doneQueue chan doneItem) (int, error) {
	order := 0
	// Create a queue of folders to traverse.  Initially we start with te
	folderQueue := &FolderStack{
		queue: list.New(),
	}
	if len(folderIDs) > 0 {
		for _, id := range folderIDs {
			// order++
			folderQueue.Push(id, "Seed")
		}
	} else {
		// We will start with the magic tree root for "My Onshape"
		// order++
		folderQueue.Push("1", "Seed")
	}

	for folderQueue.Size() > 0 {
		folderent, err := folderQueue.Pop()
		if err != nil {
			fmt.Printf("Something broke with the queue: %v\n", err)
			break
		}

		// Put the folder entry into the output print queue so that we can get the path and the url to the path
		folderResult := makefileInfo()
		folderResult.OnshapeURL = fmt.Sprintf("https://cad.onshape.com/documents?nodeId=%v&resourceType=folder", folderent.FolderID)
		folderResult.Path = folderent.FolderPath
		order++
		output := doneItem{order: order, workerID: -1, err: nil, result: folderResult, finished: false}
		doneQueue <- output

		err = OnshapeTraverseFolder(ctx, client, folderent.FolderID,
			func(ctx context.Context, client *onshape.APIClient, parentPath string, element onshape.BTGlobalTreeMagicNodeInfo) error {
				order++
				return queueFile(workQueue, order, parentPath, element)
			}, func(ctx context.Context, client *onshape.APIClient, parentPath string, folderID string) error {
				//order++
				folderQueue.Push(folderID, parentPath)
				return nil
			})
		if err != nil {
			return order, err
		}
	}

	return order, nil
}
