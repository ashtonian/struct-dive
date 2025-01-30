package gowalker

import (
	"reflect"
	"sync"
	"testing"
)

func TestWalk(t *testing.T) {
	type testStruct struct {
		Field string `tag:"field"`
	}
	tests := []struct {
		name    string
		obj     interface{}
		wantErr bool
		options []Option
		fn      UserFunc
	}{
		{
			name:    "valid struct",
			obj:     &testStruct{Field: "test"},
			wantErr: false,
			options: []Option{
				WithUserFunc(func(v reflect.Value, meta FieldMeta) error {
					println(meta.Name, " ", meta.Path)
					return nil
				}),
			},
			fn: func(v reflect.Value, meta FieldMeta) error {
				return nil
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := Walk(tt.obj, tt.options...)
			if (err != nil) != tt.wantErr {
				t.Errorf("Walk() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

type testStruct struct {
	Field1    int
	Field2    string
	SubStruct struct {
		SubField1 float32
		SubField2 bool
		subfield3 bool
	}
}

type TestCase struct {
	Name          string
	Obj           interface{}
	Options       []Option
	ExpectedPaths map[string]bool
}

func TestVisitedFieldsWithDifferentOptions(t *testing.T) {
	testCases := []TestCase{
		{
			Name: "TestMaxDepth",
			Obj: &struct {
				S1 struct {
					S2 struct {
						SF1 int
					}
				}
			}{},
			Options: []Option{
				WithUserFunc(func(v reflect.Value, meta FieldMeta) error {
					println(meta.Name, " ", meta.Path)
					return nil
				}),
				WithTypeFilter(TypeIsOneOf(reflect.TypeOf(0))),
			},
			ExpectedPaths: map[string]bool{},
		},
		// {
		// 	Name: "TestPrivateFields",
		// 	Obj: &struct {
		// 		field1 int // Unexported field.
		// 		Field2 string
		// 	}{},
		// 	Options:       []Option{PrivateFields()},
		// 	ExpectedPaths: map[string]bool{},
		// },
		// {
		// 	Name: "TestOnlySettable",
		// 	Obj: &struct {
		// 		ExportedField         int
		// 		unexportedField       int
		// 		ExportedButUnsettable int `gowalker:"-"`
		// 	}{
		// 		ExportedButUnsettable: 5,
		// 	},
		// 	Options:       []Option{OnlySettable()},
		// 	ExpectedPaths: map[string]bool{},
		// },
		// {
		// 	Name: "TestTagFilter",
		// 	Obj: &struct {
		// 		Field1 int `filter:"true"`
		// 		Field2 string
		// 	}{},
		// 	Options:       []Option{WithTagFilter(TagExists("filter", "true"))},
		// 	ExpectedPaths: map[string]bool{},
		// },
		// {
		// 	Name: "TestTypeFilter",
		// 	Obj: &struct {
		// 		Field1 int
		// 		Field2 string
		// 		Field3 float64
		// 	}{},
		// 	Options: []Option{
		// 		WithTypeFilter(
		// 			TypeIsOneOf(reflect.TypeOf(0), reflect.TypeOf("")),
		// 		),
		// 	},
		// 	ExpectedPaths: map[string]bool{},
		// },
		// {
		// 	Name: "TestIgnoreType",
		// 	Obj: &struct {
		// 		Field1 int
		// 		Field2 string
		// 		Field3 float64
		// 	}{},
		// 	Options:       []Option{WithTypeFilter(IgnoreType(reflect.TypeOf(0.0)))},
		// 	ExpectedPaths: map[string]bool{},
		// },
		// {
		// 	Name: "TestIgnoreTag",
		// 	Obj: &struct {
		// 		Field1 int `filter:"true"`
		// 		Field2 string
		// 	}{},
		// 	Options:       []Option{WithTagFilter(IgnoreTag("filter", "true"))},
		// 	ExpectedPaths: map[string]bool{},
		// },
		// {
		// 	Name: "TestMetaFilter",
		// 	Obj: &struct {
		// 		Field1 int
		// 		Field2 string
		// 	}{},
		// 	Options: []Option{WithMetaFilter(func(meta FieldMeta) bool {
		// 		return meta.Name == "Field1"
		// 	})},
		// 	ExpectedPaths: map[string]bool{},
		// },
		// {
		// 	Name: "TestMultipleFilters",
		// 	Obj: &struct {
		// 		Field1 int `filter:"true"`
		// 		Field2 string
		// 		Field3 float64
		// 	}{},
		// 	Options: []Option{
		// 		WithTypeFilter(TypeIsOneOf(reflect.TypeOf(""))),
		// 		WithTagFilter(TagExists("filter", "true")),
		// 	},
		// 	ExpectedPaths: map[string]bool{},
		// },
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			var visitedFields sync.Map

			fn := func(v reflect.Value, meta FieldMeta) error {
				visitedFields.Store(meta.Path, struct{}{})
				return nil
			}

			tc.Options = append(tc.Options, WithUserFunc(fn))
			_, visited, err := Walk(tc.Obj, tc.Options...)
			if err != nil {
				t.Fatalf("Error during crawling: %v", err)
			}

			for k, v := range visited {
				println(k, " ", v)
			}
			println("=====================================")

			visitedFields.Range(func(key, value interface{}) bool {
				println(key.(string), " ", value)
				path := key.(string)
				if _, ok := tc.ExpectedPaths[path]; !ok {
					t.Errorf("Unexpected field visited: %s", path)
				}
				delete(tc.ExpectedPaths, path)
				return true
			})

			for missingField := range tc.ExpectedPaths {
				t.Errorf("Field was not visited: %s", missingField)
			}
		})
	}
}
