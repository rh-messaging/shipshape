package qdrmanagement

import (
	"encoding/json"
	entities2 "github.com/rh-messaging/shipshape/pkg/apps/qdrouterd/qdrmanagement/entities"
	"github.com/rh-messaging/shipshape/pkg/framework"
	"reflect"
	"time"
)

const (
	timeout time.Duration = 60 * time.Second
)

var (
	queryCommand = []string{"qdmanage", "query", "--type"}
)

// QdmanageQuery executes a "qdmanager query" command on the provided pod, returning
// a slice of entities of the provided "entity" type.
func QdmanageQuery(c framework.ContextData, pod string, entity entities2.Entity, fn func(entities2.Entity) bool) ([]entities2.Entity, error) {
	// Preparing command to execute
	command := append(queryCommand, entity.GetEntityId())
	kubeExec := framework.NewKubectlExecCommand(c, pod, timeout, command...)
	jsonString, err := kubeExec.Exec()
	if err != nil {
		return nil, err
	}

	// Using reflection to get a slice instance of the concrete type
	vo := reflect.TypeOf(entity)
	v := reflect.SliceOf(vo)
	nv := reflect.New(v)
	//fmt.Printf("v    - %T - %v\n", v, v)
	//fmt.Printf("nv   - %T - %v\n", nv, nv)

	// Unmarshalling to a slice of the concrete Entity type provided via "entity" instance
	err = json.Unmarshal([]byte(jsonString), nv.Interface())
	if err != nil {
		//fmt.Printf("ERROR: %v\n", err)
		return nil, err
	}

	// Adding each parsed concrete Entity to the parsedEntities
	parsedEntities := []entities2.Entity{}
	for i := 0; i < nv.Elem().Len(); i++ {
		candidate := nv.Elem().Index(i).Interface().(entities2.Entity)

		// If no filter function provided, just add
		if fn == nil {
			parsedEntities = append(parsedEntities, candidate)
			continue
		}

		// Otherwhise invoke to determine whether to include
		if fn(candidate) {
			parsedEntities = append(parsedEntities, candidate)
		}
	}

	return parsedEntities, err
}
