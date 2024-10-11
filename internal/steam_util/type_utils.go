package steam_util

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

	// TODO port the python vdf parser instead of using this
	// since we already had to port the binary vdf parser from there
	"github.com/andygrunwald/vdf"
)

// Types prefixed with "Vdf" are minimal representations of the structure of a particular vdf/acf file.
// The members should be limited to what we actually care to use from these structures.
// We read in map[string]interface{} for these and then "unmarshal" it into these structs.

type VdfLoginUsers struct {
	Users map[uint64]SteamUser
}
type SteamUser struct {
	SteamID64   uint64
	AccountId   string
	AccountName string
	PersonaName string
	MostRecent  int
	Timestamp   int
}

type VdfLibraryFolders struct {
	Libraryfolders map[string]Libraryfolder
}
type Libraryfolder struct {
	Path string
}

type VdfAppManifest struct {
	AppState struct {
		ScheduledAutoUpdate int
		StateFlags          int
		UserConfig          struct {
			Language string
			BetaKey  string
		}
	}
}

type VdfConfig struct {
	InstallConfigStore struct {
		Software struct {
			Valve struct {
				Steam struct {
					CompatToolMapping map[uint32]CompatToolMapping
				}
			}
		}
	}
}
type CompatToolMapping struct {
	Name string
}

type VdfAppInfo struct {
	Data struct {
		AppInfo struct {
			Config struct {
				Launch map[string]Launch
			}
		}
	}
}
type Launch struct {
	Config struct {
		BetaKey string
		OsList  string
	}
	Executable string
}

type VdfLocalConfig struct {
	UserLocalConfigStore struct {
		Software struct {
			Valve struct {
				Steam struct {
					Apps map[uint32]AppLocalConfig
				}
			}
		}
	}
}
type AppLocalConfig struct {
	LaunchOptions string
}

func initVdfStructFromFile(vdfFilePath string, result interface{}) error {
	vdfFile, err := os.Open(vdfFilePath)
	defer vdfFile.Close()
	if err != nil {
		return errors.New(fmt.Sprintf("Couldn't open %s:\n %v", vdfFilePath, err))
	}
	parsedVdfFile, err := vdf.NewParser(vdfFile).Parse()
	if err != nil {
		return errors.New(fmt.Sprintf("Couldn't parse %s:\n %v", vdfFilePath, err))
	}
	err = populateStructFromMap(parsedVdfFile, result)
	return err
}

func populateStructFromMap(dataUnknown interface{}, result interface{}) error {
	v := reflect.ValueOf(result).Elem()
	t := v.Type()
	fieldMap := make(map[string]string)

	// Build a map of field names in the struct, mapping lowercase names to actual field names.
	for i := 0; i < v.NumField(); i++ {
		fieldMap[strings.ToLower(t.Field(i).Name)] = t.Field(i).Name
	}

	var data map[string]interface{}
	var ok bool
	if data, ok = dataUnknown.(map[string]interface{}); !ok {
		return errors.New("data can't be map[string]interface{}")
	}

	for key, value := range data {
		lookupKey := strings.ToLower(key)
		structFieldName, found := fieldMap[lookupKey]
		if !found {
			continue
		}

		field := v.FieldByName(structFieldName)
		if !field.IsValid() {
			continue
		}
		if !field.CanSet() {
			fmt.Printf("Cannot set field: %s\n", structFieldName)
			continue
		}

		fieldType := field.Type()

		// Handle nested structs by recursively calling the function if the field is a struct.
		if field.Kind() == reflect.Struct {
			if nestedMap, ok := value.(map[string]interface{}); ok {
				nestedStructPtr := reflect.New(field.Type())
				err := populateStructFromMap(nestedMap, nestedStructPtr.Interface())
				if err != nil {
					return err
				}
				field.Set(nestedStructPtr.Elem())
				continue
			}
		}

		// Handle lists of structs.
		if field.Kind() == reflect.Slice && fieldType.Elem().Kind() == reflect.Struct {
			if sliceData, ok := value.([]interface{}); ok {
				slice := reflect.MakeSlice(fieldType, len(sliceData), len(sliceData))
				for i, item := range sliceData {
					if itemMap, ok := item.(map[string]interface{}); ok {
						nestedStructPtr := reflect.New(fieldType.Elem())
						err := populateStructFromMap(itemMap, nestedStructPtr.Interface())
						if err != nil {
							return err
						}
						slice.Index(i).Set(nestedStructPtr.Elem())
					}
				}
				field.Set(slice)
				continue
			}
		}

		// Handle maps with integer types as keys.
		if field.Kind() == reflect.Map && fieldType.Key().Kind() >= reflect.Int && fieldType.Key().Kind() <= reflect.Uint64 {
			if mapData, ok := value.(map[string]interface{}); ok {
				newMap := reflect.MakeMap(fieldType)
				for mapKey, mapValue := range mapData {
					// Convert string key to the appropriate integer type
					uintKey, err := strconv.ParseUint(mapKey, 10, 64)
					if err != nil {
						fmt.Printf("Cannot convert string to integer for key %s in field %s: %s\n", mapKey, structFieldName, err)
						continue
					}

					// Create the integer key based on the map's key type
					intKey := reflect.ValueOf(uintKey).Convert(fieldType.Key())

					// Set the value in the map
					if nestedMap, ok := mapValue.(map[string]interface{}); ok {
						nestedStructPtr := reflect.New(fieldType.Elem())
						err := populateStructFromMap(nestedMap, nestedStructPtr.Interface())
						if err != nil {
							return err
						}
						newMap.SetMapIndex(intKey, nestedStructPtr.Elem())
					} else {
						newMap.SetMapIndex(intKey, reflect.ValueOf(mapValue))
					}
				}
				field.Set(newMap)
				continue
			}
		}

		// Handle maps of string keys to custom struct values.
		if field.Kind() == reflect.Map && fieldType.Key().Kind() == reflect.String && fieldType.Elem().Kind() == reflect.Struct {
			if mapData, ok := value.(map[string]interface{}); ok {
				newMap := reflect.MakeMap(fieldType)
				for mapKey, mapValue := range mapData {
					if nestedMap, ok := mapValue.(map[string]interface{}); ok {
						nestedStructPtr := reflect.New(fieldType.Elem())
						err := populateStructFromMap(nestedMap, nestedStructPtr.Interface())
						if err != nil {
							return err
						}
						newMap.SetMapIndex(reflect.ValueOf(mapKey), nestedStructPtr.Elem())
					} else {
						fmt.Printf("Cannot convert value to map[string]%s for field %s\n", fieldType.Elem(), structFieldName)
						continue
					}
				}
				field.Set(newMap)
				continue
			}
		}

		// Handle conversion of other types (int, string, etc.)
		if fieldType.Kind() == reflect.Int && reflect.TypeOf(value).Kind() == reflect.String {
			intValue, err := strconv.Atoi(value.(string))
			if err != nil {
				fmt.Printf("Cannot convert string to int for field %s: %s\n", structFieldName, err)
				continue
			}
			field.SetInt(int64(intValue))
			continue
		}

		val := reflect.ValueOf(value)
		if fieldType != val.Type() {
			if val.Type().ConvertibleTo(fieldType) {
				val = val.Convert(fieldType)
			} else {
				fmt.Printf("Cannot convert %s to %s for field %s\n", val.Type(), fieldType, structFieldName)
				continue
			}
		}
		field.Set(val)
	}
	return nil
}
