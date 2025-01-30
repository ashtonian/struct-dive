package gowalker

import (
	"fmt"
	"reflect"
	"sync"
)

type UserFunc func(v reflect.Value, meta FieldMeta) error
type Option func(*walkSettings)
type TagFilter func(tag reflect.StructTag) bool
type TypeFilter func(tag reflect.Type) bool
type MetaFilter func(meta FieldMeta) bool

type FieldMeta struct {
	Name      string
	Type      reflect.Type
	CanSet    bool
	IsPrivate bool
	Path      string
	Parent    *FieldMeta
	Children  []*FieldMeta
}

type walkSettings struct {
	maxDepth       int
	includePrivate bool
	onlySettable   bool
	tagFilter      TagFilter
	typeFilter     TypeFilter
	metaFilter     MetaFilter
	userFuncs      []UserFunc
}

func defaultSettings() *walkSettings {
	return &walkSettings{
		maxDepth:       10,
		includePrivate: true,
		onlySettable:   false,
		tagFilter:      nil,
		typeFilter:     nil,
		metaFilter:     nil,
	}
}

func Walk(obj interface{}, options ...Option) (*FieldMeta, map[string]*FieldMeta, error) {
	settings := defaultSettings()
	for _, option := range options {
		option(settings)
	}

	visited := make(map[uintptr]bool)

	t := reflect.TypeOf(obj)
	rootPath := t.Name()
	if rootPath == "" {
		rootPath = t.String()
	}

	flatMap := sync.Map{}

	fieldMap, err := walkRecursive(settings, obj, 0, visited, rootPath, &flatMap)
	if err != nil {
		return nil, nil, err
	}

	m := make(map[string]*FieldMeta)
	flatMap.Range(func(key, value interface{}) bool {
		m[key.(string)] = value.(*FieldMeta)
		return true
	})

	return fieldMap, m, nil

}

func walkRecursive(
	settings *walkSettings,
	obj interface{},
	depth int,
	visited map[uintptr]bool,
	path string,
	flatMap *sync.Map,
) (*FieldMeta, error) {

	if depth > settings.maxDepth {
		return nil, nil
	}

	v := reflect.ValueOf(obj)

	if !v.IsValid() || !v.CanInterface() {
		return nil, nil
	}

	t := v.Type()

	if v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		addr := v.Pointer()
		if addr != 0 {
			if visited[addr] {
				return nil, nil
			}
			visited[addr] = true
		}
	}

	name := t.Name()
	if name == "" {

		name = t.String()
	}

	meta := &FieldMeta{
		Name:      name,
		CanSet:    v.CanSet(),
		Path:      path,
		Type:      t,
		IsPrivate: t.PkgPath() != "",
	}
	flatMap.Store(path, meta)

	for _, fn := range settings.userFuncs {
		if err := fn(v, *meta); err != nil {
			return nil, err
		}
	}

	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:

		if !v.IsNil() {
			childPath := path + ".*"
			childMeta, err := walkRecursive(settings, v.Elem().Interface(), depth+1, visited, childPath, flatMap)
			if err != nil {
				return nil, err
			}
			if childMeta != nil {
				meta.Children = append(meta.Children, childMeta)
				childMeta.Parent = meta
			}
		}

	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			fieldVal := v.Field(i)
			fieldType := t.Field(i)

			if settings.tagFilter != nil && !settings.tagFilter(fieldType.Tag) {
				continue
			}

			if !settings.includePrivate && fieldType.PkgPath != "" {
				continue
			}

			if settings.onlySettable && !fieldVal.CanSet() {
				continue
			}

			newPath := path + "." + fieldType.Name
			childMeta, err := walkRecursive(settings, fieldVal.Interface(), depth+1, visited, newPath, flatMap)
			if err != nil {
				return nil, err
			}
			if childMeta != nil {
				meta.Children = append(meta.Children, childMeta)
				childMeta.Parent = meta
			}
		}

	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			elem := v.Index(i)
			newPath := fmt.Sprintf("%s[%d]", path, i)
			childMeta, err := walkRecursive(settings, elem.Interface(), depth+1, visited, newPath, flatMap)
			if err != nil {
				return nil, err
			}
			if childMeta != nil {
				meta.Children = append(meta.Children, childMeta)
				childMeta.Parent = meta
			}
		}

	case reflect.Map:
		for _, key := range v.MapKeys() {
			value := v.MapIndex(key)
			newPath := fmt.Sprintf("%s[%v]", path, key.Interface())
			childMeta, err := walkRecursive(settings, value.Interface(), depth+1, visited, newPath, flatMap)
			if err != nil {
				return nil, err
			}
			if childMeta != nil {
				meta.Children = append(meta.Children, childMeta)
				childMeta.Parent = meta
			}
		}
	}

	return meta, nil
}

func MaxDepth(depth int) Option {
	return func(s *walkSettings) {
		s.maxDepth = depth
	}
}

func PrivateFields() Option {
	return func(s *walkSettings) {
		s.includePrivate = true
	}
}

func OnlySettable() Option {
	return func(s *walkSettings) {
		s.onlySettable = true
	}
}

func WithMetaFilter(fn MetaFilter) Option {
	return func(s *walkSettings) {
		s.metaFilter = fn
	}
}

func AllMetaFilters(filters ...MetaFilter) MetaFilter {
	return func(meta FieldMeta) bool {
		for _, filter := range filters {
			if !filter(meta) {
				return false
			}
		}
		return true
	}
}

func AnyMetaFilter(filters ...MetaFilter) MetaFilter {
	return func(meta FieldMeta) bool {
		for _, filter := range filters {
			if filter(meta) {
				return true
			}
		}
		return false
	}
}

func WithTagFilter(fn TagFilter) Option {
	return func(s *walkSettings) {
		s.tagFilter = fn
	}
}

func TagExists(tagKey string, values ...string) TagFilter {
	return func(tag reflect.StructTag) bool {
		tagValue, exists := tag.Lookup(tagKey)
		if !exists {
			return false
		}
		if len(values) == 0 {
			return true
		}
		for _, value := range values {
			if tagValue == value {
				return true
			}
		}
		return false
	}
}

func IgnoreTag(tagKey string, values ...string) TagFilter {
	return func(tag reflect.StructTag) bool {
		tagValue, exists := tag.Lookup(tagKey)
		if !exists {
			return true
		}
		if len(values) == 0 {
			return false
		}
		for _, value := range values {
			if tagValue == value {
				return false
			}
		}
		return true
	}
}

func AllTagFilters(filters ...TagFilter) TagFilter {
	return func(tag reflect.StructTag) bool {
		for _, filter := range filters {
			if !filter(tag) {
				return false
			}
		}
		return true
	}
}

func AnyTagFilter(filters ...TagFilter) TagFilter {
	return func(tag reflect.StructTag) bool {
		for _, filter := range filters {
			if filter(tag) {
				return true
			}
		}
		return false
	}
}

func WithTypeFilter(fn TypeFilter) Option {
	return func(s *walkSettings) {
		s.typeFilter = fn
	}
}

func IgnoreType(disallowedTypes ...reflect.Type) TypeFilter {
	return func(t reflect.Type) bool {
		for _, disallowedType := range disallowedTypes {
			if t == disallowedType {
				return false
			}
		}
		return true
	}
}

func TypeIsOneOf(allowedTypes ...reflect.Type) TypeFilter {
	return func(t reflect.Type) bool {
		for _, allowedType := range allowedTypes {
			if t == allowedType {
				return true
			}
		}
		return false
	}
}

func WithUserFunc(fn UserFunc) Option {
	return func(s *walkSettings) {
		s.userFuncs = append(s.userFuncs, fn)
	}
}
