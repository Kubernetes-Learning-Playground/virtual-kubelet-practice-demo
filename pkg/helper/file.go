package helper

import (
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"
	"k8s.io/klog"
)

// YamlFile2Struct 读取文件内容 且反序列为struct
func YamlFile2Struct(path string, obj interface{}) error {
	b, err := GetFileContent(path)
	if err != nil {
		klog.Error("开启文件错误：", err)
		return err
	}
	err = yaml.Unmarshal(b, obj)
	if err != nil {
		klog.Error("解析yaml文件错误：", err)
		return err
	}
	return nil
}

// GetFileContent 文件读取函数
func GetFileContent(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	b, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return b, nil
}
