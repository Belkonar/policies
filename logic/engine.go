package logic

import (
	"fmt"
	"github.com/antonmedv/expr"
	"github.com/belkonar/policies/models"
)

func checkRelation(principal string, relation string, object string) bool {
	fmt.Println(principal, relation, object)
	return true
}

func FormRequest(request models.EngineRequest) map[string]interface{} {
	return map[string]interface{}{
		"principalId": request.Principal,
		"resourceId":  request.Resource,
		"context":     request.Context,
		"rel": func(s string) bool {
			return checkRelation(request.Principal, s, request.Resource)
		},
		"full": func(s string, object string) bool {
			return checkRelation(request.Principal, s, object)
		},
	}
}

// Execute takes a request and a policy and returns a list of actions that are allowed
func Execute(request map[string]interface{}, policy []models.Policy) ([]string, error) {
	ret := make([]string, 0)

	request["permissions"] = ret

	for _, p := range policy {
		code := p.Rule

		program, err := expr.Compile(code, expr.Env(request), expr.AsBool())

		if err != nil {
			return nil, err
		}

		output, err := expr.Run(program, request)

		if err != nil {
			return nil, err
		}

		if output.(bool) {
			ret = append(ret, p.Action)
		}
	}
	return ret, nil
}
