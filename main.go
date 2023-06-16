package main

import (
	"fmt"
	"github.com/belkonar/policies/logic"
	"github.com/belkonar/policies/models"
)

func test(str string) {
	fmt.Println(str)
}

func main() {
	//r := chi.NewRouter()
	//
	//r.Get("/", func(w http.ResponseWriter, r *http.Request) {
	//	_, _ = w.Write([]byte("welcome"))
	//})
	//
	//r.Get("/{name}", func(w http.ResponseWriter, r *http.Request) {
	//	name := chi.URLParam(r, "name")
	//	_, _ = w.Write([]byte(name))
	//})
	//
	//fmt.Println("Server running on port 3000")
	//
	//_ = http.ListenAndServe(":3000", r)

	policy := models.Policy{
		Action:       "read",
		ResourceType: "document",
		Rule:         `principalId == "user:bob" && rel('bob')`,
	}

	request := models.EngineRequest{
		Principal: "user:bob",
		Resource:  "document:123",
		Context:   map[string]interface{}{},
	}

	perms, err := logic.Execute(logic.FormRequest(request), []models.Policy{policy})

	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(perms)
}
