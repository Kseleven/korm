package korm

import (
	"fmt"
	"github.com/jackc/pgx/v5"
	"net"
	"net/netip"
	"reflect"
	"strings"
	"unsafe"
)

type FieldInfo struct {
	Name   string
	Offset uintptr
	Type   reflect.Type
}

func getFieldOffsets(t reflect.Type) []FieldInfo {
	fieldOffsets := make([]FieldInfo, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldOffsets[i] = FieldInfo{
			Name:   field.Name,
			Offset: field.Offset,
			Type:   field.Type,
		}
	}
	return fieldOffsets
}

func scanToStruct(rows pgx.Rows, result interface{}) error {
	val := reflect.ValueOf(result).Elem()
	t := val.Type()
	fieldOffsets := getFieldOffsets(t)

	scanTargets := make([]interface{}, len(fieldOffsets))
	fieldNames := make([]string, 0, len(fieldOffsets))
	values := make([]interface{}, len(fieldOffsets))

	for _, field := range fieldOffsets {
		fieldNames = append(fieldNames, field.Name)
		values[len(fieldNames)-1] = new(interface{})
		scanTargets[len(fieldNames)-1] = values[len(fieldNames)-1]
	}

	if !rows.Next() {
		return fmt.Errorf("no rows found")
	}
	if err := rows.Scan(scanTargets...); err != nil {
		return err
	}

	for i, value := range values {
		fmt.Printf("scanTarget:%v fielName:%v valueType:%v\n",
			*scanTargets[i].(*interface{}), fieldNames[i], *value.(*interface{}))
	}

	// 使用 unsafe.Pointer 赋值
	structPtr := unsafe.Pointer(val.UnsafeAddr())
	for i := range fieldNames {
		fieldInfo := fieldOffsets[i]
		fieldPtr := unsafe.Add(structPtr, fieldInfo.Offset)
		rawValue := *values[i].(*interface{})
		if rawValue == nil {
			continue
		}

		switch fieldInfo.Type.Kind() {
		case reflect.Int:
			*(*int)(fieldPtr) = int(rawValue.(int64))
		case reflect.String:
			*(*string)(fieldPtr) = rawValue.(string)
		case reflect.Slice:
			elemType := fieldInfo.Type.Elem()
			fmt.Printf("elemType:%v rawValue:%v\n", elemType, reflect.TypeOf(rawValue))

			if elemType.Kind() == reflect.Int {
				pgArray, ok := rawValue.([]uint8)
				if !ok {
					return fmt.Errorf("expected []int64 but got %T", rawValue)
				}
				intArray := make([]int, len(pgArray))
				for i, v := range pgArray {
					intArray[i] = int(v)
				}
				*(*[]int)(fieldPtr) = intArray
			}

			if elemType == reflect.TypeOf(net.IP{}) {
				str := string(rawValue.([]uint8))
				fmt.Printf("inetp[]:%s\n", str)
				str = strings.Trim(str, "{}")
				parts := strings.Split(str, ",")
				var ips []net.IP
				for _, part := range parts {
					ip := net.ParseIP(strings.TrimSpace(part)) // 解析 IP
					ips = append(ips, ip)
				}
				*(*[]net.IP)(fieldPtr) = ips
			}

			if elemType == reflect.TypeOf(netip.Prefix{}) {
				bys, ok := rawValue.([]uint8)
				if !ok {
					return fmt.Errorf("expected []string but got %T", rawValue)
				}
				strArray := parseTextArray(string(bys))
				prefixArray := make([]netip.Prefix, len(strArray))
				for i, v := range strArray {
					prefix, err := netip.ParsePrefix(v)
					if err != nil {
						return fmt.Errorf("invalid CIDR format: %s", v)
					}
					prefixArray[i] = prefix
				}
				*(*[]netip.Prefix)(fieldPtr) = prefixArray
			}
		}
	}
	return nil
}

func parseByteaArray(data []uint8) [][]byte {
	var result [][]byte
	if len(data) == 0 {
		return result
	}
	return [][]byte{data}
}

func parseTextArray(data string) []string {
	data = strings.Trim(data, "{}")
	if data == "" {
		return []string{}
	}
	return strings.Split(data, ",")
}
