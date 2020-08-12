package main

import (
	"context"
	"fmt"

	"github.com/toebes/go-client/onshape"
)

// OnshapeSetDocumentDescription sets the description field on a document
func OnshapeSetDocumentDescription(ctx context.Context, client *onshape.APIClient, did string, description string) error {
	docParams := onshape.NewBTDocumentParams()
	docParams.SetDescription(description)
	rawResp, err := client.DocumentsApi.UpdateDocumentAttributes(ctx, did).BTDocumentParams(*docParams).Execute()
	if err == nil && rawResp != nil && rawResp.StatusCode >= 300 {
		err = fmt.Errorf("err: Response status: %v", rawResp)
	}
	return err
}
