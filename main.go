package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/allegro/bigcache/v3"
	"github.com/belkonar/policies/logic"
	"github.com/belkonar/policies/models"
	"github.com/belkonar/policies/openfga"
	"github.com/go-chi/chi/v5"
	fga "github.com/openfga/go-sdk"
	etcd "go.etcd.io/etcd/client/v3"
	"io"
	"net/http"
	"time"
)

func main() {
	r := chi.NewRouter()

	cacheConfig := bigcache.DefaultConfig(365 * 24 * time.Hour)
	cacheConfig.CleanWindow = -1

	cache, _ := bigcache.New(context.Background(), cacheConfig)

	configuration, err := fga.NewConfiguration(fga.Configuration{
		ApiScheme: "http",
		ApiHost:   "localhost:8080",
	})

	fgaClient := openfga.FgaClient{
		Configuration: configuration,
	}

	engine := logic.Engine{
		Cache: cache,
		EtcdConfig: etcd.Config{
			Endpoints:   []string{"localhost:2379"},
			DialTimeout: 5 * time.Second,
		},
		Fga: &fgaClient,
	}

	engine.InitialLoad()

	go engine.Watcher()

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("OK"))
	})

	r.Route("/namespace/{namespace}", func(r chi.Router) {
		r.Route("/doc/{document}", func(r chi.Router) {
			r.Put("/", func(w http.ResponseWriter, r *http.Request) {
				namespaceId := chi.URLParam(r, "namespace")
				documentId := chi.URLParam(r, "document")

				bodyData, err := io.ReadAll(r.Body)

				if err != nil {
					_, _ = w.Write([]byte(err.Error()))
					return
				}

				document := models.Document{}

				err = json.Unmarshal(bodyData, &document)

				if err != nil {
					_, _ = w.Write([]byte(err.Error()))
					return
				}

				document.Key = documentId
				document.NamespaceId = namespaceId

				err = engine.SaveDocument(document)
			})
		})

		r.Put("/", func(w http.ResponseWriter, r *http.Request) {
			namespaceId := chi.URLParam(r, "namespace")

			bodyData, err := io.ReadAll(r.Body)

			if err != nil {
				_, _ = w.Write([]byte(err.Error()))
			}

			namespace := models.Namespace{}

			err = json.Unmarshal(bodyData, &namespace)

			if err != nil {
				_, _ = w.Write([]byte(err.Error()))
				return
			}

			namespace.Id = namespaceId

			err = engine.SaveNamespace(namespace)

			if err != nil {
				_, _ = w.Write([]byte(err.Error()))
				return
			}
		})

		r.Post("/refresh", func(w http.ResponseWriter, r *http.Request) {
			namespaceId := chi.URLParam(r, "namespace")

			err := engine.RefreshPolicyCache(namespaceId)

			if err != nil {
				_, _ = w.Write([]byte(err.Error()))
				return
			}

			_, _ = w.Write([]byte("OK"))
		})

		r.Post("/", func(w http.ResponseWriter, r *http.Request) {
			namespaceId := chi.URLParam(r, "namespace")

			bodyData, err := io.ReadAll(r.Body)

			if err != nil {
				_, _ = w.Write([]byte(err.Error()))
				return
			}

			request := models.GetPermissionsRequest{}

			err = json.Unmarshal(bodyData, &request)

			request.NamespaceId = namespaceId

			fmt.Println(request)

			if err != nil {
				_, _ = w.Write([]byte(err.Error()))
				return
			}

			response, err := engine.ProcessEngineRequest(request)

			if err != nil {
				_, _ = w.Write([]byte(err.Error()))
				return
			}

			responseData, err := json.Marshal(response)

			if err != nil {
				_, _ = w.Write([]byte(err.Error()))
				return
			}

			w.Header().Set("Content-Type", "application/json")

			_, _ = w.Write(responseData)
		})
	})

	fmt.Println("Server running on port 3030")

	err = http.ListenAndServe(":3030", r)
	if err != nil {
		panic(err)
	}
}
