package web

import (
	"embed"
	"html/template"
	"io/fs"
	"net/http"
)

//go:embed assets/static
var staticFiles embed.FS

//go:embed assets/templates
var templatesFiles embed.FS

// GetStaticFS 返回静态文件的文件系统
func GetStaticFS() (http.FileSystem, error) {
	sub, err := fs.Sub(staticFiles, "assets/static")
	if err != nil {
		return nil, err
	}
	return http.FS(sub), nil
}

// LoadTemplates 加载并返回解析好的模板
func LoadTemplates() (*template.Template, error) {
	// 创建模板并添加函数
	tmpl := template.New("")

	// 获取模板子文件系统
	templateFS, err := fs.Sub(templatesFiles, "assets/templates")
	if err != nil {
		return nil, err
	}

	// 读取并解析所有模板
	entries, err := fs.ReadDir(templateFS, ".")
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()
		content, err := fs.ReadFile(templateFS, filename)
		if err != nil {
			return nil, err
		}

		_, err = tmpl.New(filename).Parse(string(content))
		if err != nil {
			return nil, err
		}
	}

	return tmpl, nil
}
