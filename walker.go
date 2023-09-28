package gowalker

import (
	"fmt"
	"reflect"
)

type UserFunc func(v reflect.Value, meta ObjMeta) error
type Option func(*walkSettings)
type TagFilter func(tag reflect.StructTag) bool
type TypeFilter func(tag reflect.Type) bool
type MetaFilter func(meta ObjMeta) bool

type ObjMeta struct {
	Name      string       // Field name
	Type      reflect.Type // Field type
	CanSet    bool         // If the field can be set
	IsPrivate bool         // If the field is unexported
	Path      string       // The field's path
	Parent    *ObjMeta     // Parent field meta (nil if root)
	Children  []*ObjMeta   // Children fields (for nested structs)
}

type walkSettings struct {
	MaxDepth       int
	IncludePrivate bool
	OnlySettable   bool
	TagFilter      TagFilter
	TypeFilter     TypeFilter
	MetaFilter     MetaFilter
}

func defaultSettings() *walkSettings {
	return &walkSettings{
		MaxDepth: 10,
	}
}

func Walk(obj interface{}, fn UserFunc, options ...Option) (*ObjMeta, error) {
	settings := defaultSettings()
	for _, option := range options {
		option(settings)
	}

	visited := make(map[uintptr]bool)
	cache := make(map[string]*ObjMeta)

	t := reflect.TypeOf(obj)
	rootPath := t.Name()
	if rootPath == "" {
		rootPath = t.String()
	}

	return walkRecursive(settings, obj, fn, 0, visited, rootPath, cache)
}

func walkRecursive(settings *walkSettings, obj interface{}, fn UserFunc, depth int, visited map[uintptr]bool, path string, cache map[string]*ObjMeta) (*ObjMeta, error) {
	if depth > settings.MaxDepth {
		return nil, nil
	}

	v := reflect.ValueOf(obj)
	if !v.IsValid() || !v.CanInterface() {
		return nil, nil // invalid value, skip it
	}

	// Use type filters and struct tag filters
	t := v.Type()
	if settings.TypeFilter != nil && !settings.TypeFilter(t) {
		return nil, nil
	}

	if t.Kind() == reflect.Struct {
		if sf, ok := t.FieldByName(path); ok && settings.TagFilter != nil {
			tag := sf.Tag
			if !settings.TagFilter(tag) {
				return nil, nil
			}
		}
	}

	name := t.Name()
	if name == "" {
		name = t.String() // For slice, map, etc.
	}

	// Check and mark visited for pointers and interfaces
	if v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		addr := v.Pointer()
		if visited[addr] {
			return nil, nil // avoid loop
		}
		visited[addr] = true
	}

	if meta, ok := cache[path]; ok {
		return meta, nil
	}

	meta := &ObjMeta{
		Name:      name,
		CanSet:    v.CanSet(),
		Path:      path,
		Type:      t,
		IsPrivate: t.PkgPath() != "",
	}

	if settings.MetaFilter != nil && !settings.MetaFilter(*meta) {
		return nil, nil
	}

	cache[path] = meta

	// Call the user function, ending the recursion
	err := fn(v, *meta)
	if err != nil {
		return nil, err
	}

	// Recursive calls for nested types
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		childMeta, err := walkRecursive(settings, v.Elem().Interface(), fn, depth+1, visited, path, cache)
		if err != nil {
			return nil, err
		}
		if childMeta != nil {
			meta.Children = append(meta.Children, childMeta)
			childMeta.Parent = meta
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)
			fieldType := t.Field(i)
			if fieldType.PkgPath != "" {
				continue
			}
			newPath := path + "." + fieldType.Name
			childMeta, err := walkRecursive(settings, field.Interface(), fn, depth+1, visited, newPath, cache)
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
			childMeta, err := walkRecursive(settings, elem.Interface(), fn, depth+1, visited, newPath, cache)
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
			childMeta, err := walkRecursive(settings, value.Interface(), fn, depth+1, visited, newPath, cache)
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
		s.MaxDepth = depth
	}
}

func PrivateFields() Option {
	return func(s *walkSettings) {
		s.IncludePrivate = true
	}
}

func OnlySettable() Option {
	return func(s *walkSettings) {
		s.OnlySettable = true
	}
}

func WithMetaFilter(fn MetaFilter) Option {
	return func(s *walkSettings) {
		s.MetaFilter = fn
	}
}

func AllMetaFilters(filters ...MetaFilter) MetaFilter {
	return func(meta ObjMeta) bool {
		for _, filter := range filters {
			if !filter(meta) {
				return false
			}
		}
		return true
	}
}

func AnyMetaFilter(filters ...MetaFilter) MetaFilter {
	return func(meta ObjMeta) bool {
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
		s.TagFilter = fn
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
		s.TypeFilter = fn
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
