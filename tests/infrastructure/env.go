package infrastructure

import "os"

// GetWithDefault получение переменной окружения с дефолтным значением
func GetWithDefault(key, defaultValue string) string {
	value, ok := os.LookupEnv(key)
	if ok {
		return value
	}
	return defaultValue
}