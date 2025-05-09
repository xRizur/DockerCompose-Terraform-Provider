package main

import (
	"gopkg.in/yaml.v2"
)

func getMap(data map[string]interface{}, key string) map[string]interface{} {
	if val, ok := data[key]; ok && val != nil {
		return val.(map[string]interface{})
	}
	return map[string]interface{}{}
}

func getString(data map[string]interface{}, key string, defaultValue string) string {
	if val, ok := data[key]; ok && val != nil {
		return val.(string)
	}
	return defaultValue
}

func getInt(data map[string]interface{}, key string, defaultValue int) int {
	if val, ok := data[key]; ok && val != nil {
		return val.(int)
	}
	return defaultValue
}

func getStringList(data map[string]interface{}, key string) []string {
	if val, ok := data[key]; ok && val != nil {
		rawList := val.([]interface{})
		strList := make([]string, len(rawList))
		for i, item := range rawList {
			strList[i] = item.(string)
		}
		return strList
	}
	return []string{}
}

func getStringMap(data map[string]interface{}, key string) map[string]string {
	if val, ok := data[key]; ok && val != nil {
		rawMap := val.(map[string]interface{})
		strMap := make(map[string]string, len(rawMap))
		for k, v := range rawMap {
			strMap[k] = v.(string)
		}
		return strMap
	}
	return map[string]string{}
}

func myToYaml(v interface{}) string {
	b, err := yaml.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}
