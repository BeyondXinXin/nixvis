package util

import "os"

// 初始化文件夹
func InitDir() {
	fileList := []string{"./data"}

	for _, file := range fileList {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			err := os.MkdirAll(file, 0755)
			if err != nil {
				panic(err)
			}
		}
	}

	// 检查并创建配置文件 ./data/config.json
	if _, err := os.Stat("./data/config.json"); os.IsNotExist(err) {
		configJson := `{"system": {"logDestination": "stdout"}}`
		err := os.WriteFile("./data/config.json", []byte(configJson), 0644)
		if err != nil {
			panic(err)
		}
	}
}
