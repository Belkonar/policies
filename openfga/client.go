package openfga

import (
	"context"
	"fmt"
	fga "github.com/openfga/go-sdk"
)

type FgaClient struct {
	Configuration *fga.Configuration
}

func (f *FgaClient) CheckRelation(storeId string, user string, relation string, object string) bool {
	client := fga.NewAPIClient(f.Configuration)

	client.SetStoreId(storeId)

	request := fga.CheckRequest{
		TupleKey: fga.TupleKey{
			User:     &user,
			Object:   &object,
			Relation: &relation,
		},
	}

	data, _, err := client.OpenFgaApi.Check(context.Background()).Body(request).Execute()

	if err != nil {
		fmt.Println(err.Error())
		return false
	}

	return data.GetAllowed()
}
