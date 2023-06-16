package logic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/allegro/bigcache/v3"
	"github.com/antonmedv/expr"
	"github.com/belkonar/policies/models"
	"github.com/belkonar/policies/openfga"
	etcd "go.etcd.io/etcd/client/v3"
	"strings"
)

type Engine struct {
	Cache      *bigcache.BigCache
	EtcdConfig etcd.Config
	Fga        *openfga.FgaClient
}

func (e *Engine) MakeRequest(permissionsRequest models.GetPermissionsRequest) map[string]any {
	if permissionsRequest.Context == nil {
		permissionsRequest.Context = make(map[string]any)
	}

	namespace, err := e.GetNamespace(permissionsRequest.NamespaceId)

	if err != nil {
		return nil
	}

	local := permissionsRequest.Context

	local["principalId"] = permissionsRequest.PrincipalId
	local["resourceId"] = permissionsRequest.ResourceId
	local["storeId"] = namespace.FgaStore

	return local
}

// GetNamespace Make this a background cached item to put storeIds in cache
func (e *Engine) GetNamespace(namespaceId string) (models.Namespace, error) {
	namespace := models.Namespace{}

	cli, err := etcd.New(e.EtcdConfig)
	if err != nil {
		return namespace, err
	}
	defer cli.Close()

	resp, err := cli.Get(context.Background(), fmt.Sprintf("/namespace/%s", namespaceId))

	if err != nil {
		return namespace, err
	}

	err = json.Unmarshal(resp.Kvs[0].Value, &namespace)

	return namespace, nil
}

func (e *Engine) Watcher() {
	cli, err := etcd.New(e.EtcdConfig)
	if err != nil {
		// handle error!
		fmt.Println(err)
	}
	defer cli.Close()

	rch := cli.Watch(context.Background(), "/docs/", etcd.WithPrefix())

	for resp := range rch {
		for _, ev := range resp.Events {
			namespace := strings.Split(string(ev.Kv.Key), "/")[2]
			err := e.RefreshPolicyCache(namespace)

			if err != nil {
				fmt.Println(err)
			}
		}
	}

	println("Closing watcher")
}

func (e *Engine) InitialLoad() {
	cli, err := etcd.New(e.EtcdConfig)
	if err != nil {
		// handle error!
		fmt.Println(err)
	}
	defer cli.Close()

	resp, err := cli.Get(context.Background(), "/namespace/", etcd.WithPrefix())

	if err != nil {
		// handle error!
		fmt.Println(err)
	}

	for _, kv := range resp.Kvs {
		namespace := models.Namespace{}
		err = json.Unmarshal(kv.Value, &namespace)

		if err != nil {
			fmt.Println(err)
		}

		err = e.RefreshPolicyCache(namespace.Id)

		if err != nil {
			fmt.Println(err)
		}
	}
}

func (e *Engine) SaveNamespace(namespace models.Namespace) error {
	cli, err := etcd.New(e.EtcdConfig)
	if err != nil {
		// handle error!
		return err
	}
	defer cli.Close()

	namespaceData, _ := json.Marshal(namespace)

	_, err = cli.Put(context.Background(), fmt.Sprintf("/namespace/%s", namespace.Id), string(namespaceData))

	if err != nil {
		// handle error!
		return err
	}

	return nil
}

func (e *Engine) SaveDocument(document models.Document) error {
	cli, err := etcd.New(e.EtcdConfig)
	if err != nil {
		// handle error!
		return err
	}
	defer cli.Close()

	documentData, _ := json.Marshal(document)

	_, err = cli.Put(context.Background(), fmt.Sprintf("/docs/%s/%s", document.NamespaceId, document.Key), string(documentData))

	if err != nil {
		// handle error!
		return err
	}

	return nil
}

func (e *Engine) RefreshPolicyCache(namespace string) error {

	prefix := fmt.Sprintf("/docs/%s/", namespace)

	cli, err := etcd.New(e.EtcdConfig)
	if err != nil {
		// handle error!
		fmt.Println(err)
	}
	defer cli.Close()

	resp, err := cli.Get(context.Background(), prefix, etcd.WithPrefix())

	if err != nil {
		return err
	}

	grouper := make(map[string][]models.Policy)

	for _, kv := range resp.Kvs {
		policy := models.Document{}

		err := json.Unmarshal(kv.Value, &policy)

		if err != nil {
			return err
		}

		for _, p := range policy.Policies {
			if _, ok := grouper[p.ResourceType]; !ok {
				grouper[p.ResourceType] = make([]models.Policy, 0)
			}

			grouper[p.ResourceType] = append(grouper[p.ResourceType], p)
		}
	}

	for k, v := range grouper {
		policyData, _ := json.Marshal(v)

		cacheKey := fmt.Sprintf("%s/%s", namespace, k)

		_ = e.Cache.Set(cacheKey, policyData)
	}

	return nil
}

// Execute takes a request and a policy and returns a list of actions that are allowed
func (e *Engine) Execute(request map[string]any, policy []models.Policy) ([]string, error) {
	storeId := ""
	if request["storeId"] != nil {
		storeId = request["storeId"].(string)
	} else {
		return nil, errors.New("storeId is required")
	}

	request["rel"] = func(s string) bool {
		return e.Fga.CheckRelation(storeId, request["principalId"].(string), s, request["resourceId"].(string))
	}

	request["full"] = func(s string, object string) bool {
		return e.Fga.CheckRelation(storeId, request["principalId"].(string), s, object)
	}

	perms := make([]string, 0)

	request["perms"] = perms

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
			perms = append(perms, p.Action)
		}
	}
	return perms, nil
}

func (e *Engine) ProcessEngineRequest(request models.GetPermissionsRequest) ([]string, error) {
	resourceType := strings.Split(request.ResourceId, ":")[0]

	cacheKey := fmt.Sprintf("%s/%s", request.NamespaceId, resourceType)

	cachedData, err := e.Cache.Get(cacheKey)

	if request.Policies == nil {
		request.Policies = make([]models.Policy, 0)
	}

	policies := request.Policies

	if err == nil {
		cachedPolicies := make([]models.Policy, 0)
		err = json.Unmarshal(cachedData, &cachedPolicies)

		fmt.Println(cachedPolicies)

		if err != nil {
			return nil, err
		}

		policies = append(policies, cachedPolicies...)
	} else {
		fmt.Println(err)
	}

	perms, err := e.Execute(e.MakeRequest(request), policies)

	if err != nil {
		return nil, err
	}

	return perms, nil
}
