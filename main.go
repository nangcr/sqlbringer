package main

import (
	"fmt"
	"github.com/jessevdk/go-flags"
	uuid "github.com/satori/go.uuid"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

func main() {
	var opts struct {
		OutputPath string `short:"o" long:"output" description:"output path" value-name:"OUTPUT" default:"statements"`
	}

	// 读取命令行参数
	args, flagErr := flags.Parse(&opts)
	if flagErr != nil {
		os.Exit(1)
	}
	if len(args) == 0 {
		fmt.Printf("至少需要提供一个目标\n")
		os.Exit(1)
	}

	fmt.Printf("共有目标【%d】个，结果保存到%s\n", len(args), opts.OutputPath)

	wg := sync.WaitGroup{}

	// 循环遍历命令行参数
	for k := range args {
		fmt.Printf("开始处理第%d个目标【%s】\n", k+1, args[k])
		wg.Add(1)
		go func(objectPath string) {
			defer func() {
				if err := recover(); err != nil {
					fmt.Printf("处理目标【%s】异常：%s\n", objectPath, err)
				} else {
					fmt.Printf("处理目标【%s】完成\n", objectPath)
				}
				wg.Done()
			}()
			err := writeToOutputPath(objectPath, opts.OutputPath)
			if err != nil {
				fmt.Printf("处理目标【%s】异常：%s\n", objectPath, err)
			}
		}(args[k])
	}

	wg.Wait()
	fmt.Printf("所有目标处理完成，保存在%s\n", opts.OutputPath)
}

func writeToOutputPath(objectPath, outputPath string) error {
	// 如果参数是一个git仓库，则将其克隆到当前目录
	if strings.HasPrefix(objectPath, "http://") ||
		strings.HasPrefix(objectPath, "https://") ||
		strings.HasPrefix(objectPath, "git@") {
		tempDir := uuid.NewV4().String()
		defer func() {
			err := os.RemoveAll(tempDir)
			if err != nil {
				fmt.Printf("删除临时目录【%s】失败：%s\n", tempDir, err)
			}
		}()
		fmt.Printf("检测到参数【%s】为git仓库，开始克隆\n", objectPath)
		cmd := exec.Command("git", "clone", "--depth=1", objectPath, tempDir)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			fmt.Printf("克隆仓库【%s】失败\n", objectPath)
			return err
		}
		fmt.Printf("克隆完成【%s】\n", objectPath)
		objectPath = tempDir
	}

	// 检查目录是否存在
	_, err := os.Stat(objectPath)
	if err != nil {
		fmt.Printf("目标【%s】不存在\n", objectPath)
		return err
	}

	objectFiles := []string{}
	err = filepath.Walk(objectPath, func(path string, info os.FileInfo, err error) error {
		// 如果遍历到的文件是一个目录，则跳过
		if info.IsDir() {
			return nil
		}
		// 如果遍历到的文件是一个.sql文件，objectFiles
		if filepath.Ext(path) == ".sql" {
			objectFiles = append(objectFiles, path)
		}
		return nil
	})
	// 如果遍历出错，则打印错误信息
	if err != nil {
		fmt.Printf("遍历目标【%s】失败\n", objectPath)
		return err
	}

	// 判断statements目录是否存在，如果不存在，则创建
	_, err = os.Stat(outputPath)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.Mkdir(outputPath, 0755)
			if err != nil {
				fmt.Printf("创建目录【%s】失败\n", outputPath)
				return err
			}
		} else {
			fmt.Printf("检查目录【%s】失败\n", outputPath)
			return err
		}
	}

	wg := sync.WaitGroup{}
	// 循环遍历files切片
	for _, file := range objectFiles {
		wg.Add(1)
		go func(filePath string) {
			defer func() {
				if err := recover(); err != nil {
					fmt.Printf("处理文件【%s】异常：%s\n", filePath, err)
				}
				wg.Done()
			}()
			// 读取文件内容
			content, err := ioutil.ReadFile(filePath)
			if err != nil {
				fmt.Printf("读取文件【%s】失败\n", filePath)
			}

			// 生成一个随机文件名
			filename := uuid.NewV4().String() + ".sql"
			err = ioutil.WriteFile(filepath.Join(outputPath, filename), content, 0644)
			if err != nil {
				fmt.Printf("写入文件【%s】失败\n", filename)
			}
		}(file)
	}
	wg.Wait()
	return nil
}
