# gowalker

Gowalker is a Go package enabling in-depth reflection and inspection of objects and their values with various customizable options and filters. Gowalker can walk through nested interfaces, pointers, structs, maps, and slices.

## Installation

```
go get -u github.com/ashtonian/gowalker
```

## Usage

### Basic Usage

Here’s a simple example demonstrating the basic usage of gowalker to trim strings from a struct:

```go
func main() {
    obj := struct{Name: " Example ", Value: " Value "}

    fn := func(v reflect.Value, meta gowalker.ObjMeta) error {
            v.SetString(strings.TrimSpace(v.String()))
        }
        return nil
    }

    gowalker.Walk(&obj, fn, TypeIsOneOf(reflect.String))
}
```

### Options

Gowalker offers options like `MaxDepth`, `PrivateFields`, `OnlySettable`, and `WithTagFilter` to customize the walking process according to your needs.

### Examples

#### Sync Map with Paths

To store all the visited paths in a sync.Map, use the example below:

```go
// show path
func main() {
 obj := MyStruct{Name: "Example", Value: 42}
 var syncMap sync.Map

 _, _ = gowalker.Walk(obj, func(v reflect.Value, meta gowalker.ObjMeta) error {
  syncMap.Store(meta.Path, v.Interface())
  return nil
 })
}
```

// TODO:
* fix tests
* build
* doc options
* doc func
* get val fn example
* viper example