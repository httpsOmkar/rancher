package schema

import (
	"github.com/rancher/wrangler/pkg/schemas/validation"

	"github.com/rancher/steve/pkg/schemaserver/httperror"
	"github.com/rancher/steve/pkg/schemaserver/store/empty"
	"github.com/rancher/steve/pkg/schemaserver/types"
	"github.com/rancher/wrangler/pkg/schemas/definition"
)

type Store struct {
	empty.Store
}

func NewSchemaStore() types.Store {
	return &Store{}
}

func toAPIObject(schema *types.APISchema) types.APIObject {
	return types.APIObject{
		Type:   "schema",
		ID:     schema.ID,
		Object: schema,
	}
}

func (s *Store) ByID(apiOp *types.APIRequest, schema *types.APISchema, id string) (types.APIObject, error) {
	schema = apiOp.Schemas.LookupSchema(id)
	if schema == nil {
		return types.APIObject{}, httperror.NewAPIError(validation.NotFound, "no such schema")
	}
	return toAPIObject(schema), nil
}

func (s *Store) List(apiOp *types.APIRequest, schema *types.APISchema) (types.APIObjectList, error) {
	schemaMap := apiOp.Schemas.Schemas
	schemas := types.APIObjectList{}

	included := map[string]bool{}
	for _, schema := range schemaMap {
		if included[schema.ID] {
			continue
		}

		if apiOp.AccessControl.CanList(apiOp, schema) == nil || apiOp.AccessControl.CanGet(apiOp, schema) == nil {
			schemas = s.addSchema(apiOp, schema, schemaMap, schemas, included)
		}
	}

	return schemas, nil
}

func (s *Store) addSchema(apiOp *types.APIRequest, schema *types.APISchema, schemaMap map[string]*types.APISchema, schemas types.APIObjectList, included map[string]bool) types.APIObjectList {
	included[schema.ID] = true
	schemas = s.traverseAndAdd(apiOp, schema, schemaMap, schemas, included)
	schemas.Objects = append(schemas.Objects, toAPIObject(schema))
	return schemas
}

func (s *Store) traverseAndAdd(apiOp *types.APIRequest, schema *types.APISchema, schemaMap map[string]*types.APISchema, schemas types.APIObjectList, included map[string]bool) types.APIObjectList {
	for _, field := range schema.ResourceFields {
		t := ""
		subType := field.Type
		for subType != t {
			t = subType
			subType = definition.SubType(t)
		}

		if refSchema, ok := schemaMap[t]; ok && !included[t] {
			schemas = s.addSchema(apiOp, refSchema, schemaMap, schemas, included)
		}
	}

	for _, action := range schema.ResourceActions {
		for _, t := range []string{action.Output, action.Input} {
			if t == "" {
				continue
			}

			if refSchema, ok := schemaMap[t]; ok && !included[t] {
				schemas = s.addSchema(apiOp, refSchema, schemaMap, schemas, included)
			}
		}
	}

	for _, action := range schema.CollectionActions {
		for _, t := range []string{action.Output, action.Input} {
			if t == "" {
				continue
			}

			if refSchema, ok := schemaMap[t]; ok && !included[t] {
				schemas = s.addSchema(apiOp, refSchema, schemaMap, schemas, included)
			}
		}
	}

	return schemas
}
