package zip

import (
	"fmt"
	"reflect"
)

var logAll = false

func logSizeOf(args ...interface{}) {
	if !logAll {
		return
	}
	fmt.Println(args...)
}

func sizeOf(v interface{}, visited map[interface{}]struct{}) int64 {
	if visited == nil {
		visited = make(map[interface{}]struct{})
	}

	return sizeOf2(reflect.ValueOf(v), visited)
}

func sizeOf2(s reflect.Value, visited map[interface{}]struct{}) int64 {
	return int64(s.Type().Size()) + internalSizeOf(s, visited)
}

// sizeOf reworked from https://stackoverflow.com/a/51432438
func internalSizeOf(s reflect.Value, visited map[interface{}]struct{}) int64 {
	var size int64

	switch s.Kind() {
	case reflect.Slice:
		logSizeOf("Slice:", size)
		for i := 0; i < s.Len(); i++ {
			extra := sizeOf(s.Index(i).Interface(), visited)
			logSizeOf("Slice", i, ":", extra)
			size += extra
		}

	case reflect.Map:
		keys := s.MapKeys()
		size += int64(float64(len(keys)) * 10.79) // approximation from https://golang.org/src/runtime/hashmap.go
		logSizeOf("Map:", size)
		for i := range keys {
			keySize := sizeOf(keys[i].Interface(), visited)
			valueSize := sizeOf(s.MapIndex(keys[i]).Interface(), visited)
			logSizeOf("MapKey", i, ":", keySize, valueSize)
			size += keySize + valueSize
		}

	case reflect.String:
		if _, ok := visited[s.String()]; ok {
			break
		}
		visited[s.String()] = struct{}{}
		size += int64(s.Len())
		logSizeOf("String", size)

	case reflect.Struct:
		logSizeOf("Struct:", size, s.Type().Name())
		for i := 0; i < s.NumField(); i++ {
			//FieldByName("headerOffset").Int()
			extra := internalSizeOf(s.Field(i), visited)
			logSizeOf("Struct Field", i, ":", s.Type().Field(i).Name, extra)
			size += extra
		}

		// case reflect.Ptr:
		// 	logSizeOf("Pointer:", size)
		// 	s = reflect.Indirect(s)
		// 	return sizeOf2(s, visited)
	}
	return size
}
