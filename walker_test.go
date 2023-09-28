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
			fn: func(v reflect.Value, meta ObjMeta) error {
				return nil
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Walk(tt.obj, tt.fn, tt.options...)
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
				Field1 struct {
					SubField1 struct {
						SubSubField1 int
					}
				}
			}{},
			Options:       []Option{MaxDepth(1)},
			ExpectedPaths: map[string]bool{},
		},
		{
			Name: "TestPrivateFields",
			Obj: &struct {
				field1 int // Unexported field.
				Field2 string
			}{},
			Options:       []Option{PrivateFields()},
			ExpectedPaths: map[string]bool{},
		},
		{
			Name: "TestOnlySettable",
			Obj: &struct {
				ExportedField         int
				unexportedField       int
				ExportedButUnsettable int `gowalker:"-"`
			}{
				ExportedButUnsettable: 5,
			},
			Options:       []Option{OnlySettable()},
			ExpectedPaths: map[string]bool{},
		},
		{
			Name: "TestTagFilter",
			Obj: &struct {
				Field1 int `filter:"true"`
				Field2 string
			}{},
			Options:       []Option{WithTagFilter(TagExists("filter", "true"))},
			ExpectedPaths: map[string]bool{},
		},
		{
			Name: "TestTypeFilter",
			Obj: &struct {
				Field1 int
				Field2 string
				Field3 float64
			}{},
			Options: []Option{
				WithTypeFilter(
					TypeIsOneOf(reflect.TypeOf(0), reflect.TypeOf("")),
				),
			},
			ExpectedPaths: map[string]bool{},
		},
		{
			Name: "TestIgnoreType",
			Obj: &struct {
				Field1 int
				Field2 string
				Field3 float64
			}{},
			Options:       []Option{WithTypeFilter(IgnoreType(reflect.TypeOf(0.0)))},
			ExpectedPaths: map[string]bool{},
		},
		{
			Name: "TestIgnoreTag",
			Obj: &struct {
				Field1 int `filter:"true"`
				Field2 string
			}{},
			Options:       []Option{WithTagFilter(IgnoreTag("filter", "true"))},
			ExpectedPaths: map[string]bool{},
		},
		{
			Name: "TestMetaFilter",
			Obj: &struct {
				Field1 int
				Field2 string
			}{},
			Options: []Option{WithMetaFilter(func(meta ObjMeta) bool {
				return meta.Name == "Field1"
			})},
			ExpectedPaths: map[string]bool{},
		},
		{
			Name: "TestMultipleFilters",
			Obj: &struct {
				Field1 int `filter:"true"`
				Field2 string
				Field3 float64
			}{},
			Options: []Option{
				WithTypeFilter(TypeIsOneOf(reflect.TypeOf(""))),
				WithTagFilter(TagExists("filter", "true")),
			},
			ExpectedPaths: map[string]bool{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			var visitedFields sync.Map

			fn := func(v reflect.Value, meta ObjMeta) error {
				visitedFields.Store(meta.Path, struct{}{})
				return nil
			}

			if _, err := Walk(tc.Obj, fn, tc.Options...); err != nil {
				t.Fatalf("Error during crawling: %v", err)
			}

			visitedFields.Range(func(key, value interface{}) bool {
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
