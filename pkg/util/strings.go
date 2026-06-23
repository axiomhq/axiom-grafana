package util

func CheckString(i interface{}) string {
	if str, ok := i.(string); ok {
		return str
	}
	return ""
}
