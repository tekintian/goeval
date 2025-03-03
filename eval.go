package goeval

import (
	"errors"
	"fmt"
	"go/format"
	"math/rand"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

const (
	letterBytes = "abcdefghijklmnopqrstuvwxyz"
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

var (
	dirSeparator = "/"
	tempDir      = os.TempDir()
	src          = rand.NewSource(time.Now().UnixNano())
)

// 参考： https://colobu.com/2018/09/02/generate-random-string-in-Go/
func RandString(n int) string {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}
	return string(b)
}

func Eval(defineCode string, code string, imports ...string) (re []byte, err error) {
	var (
		tmp = `package main

%s

%s

func main() {
%s
}
`
		importStr string
		fullCode string
	 	newTmpDir = tempDir + dirSeparator + RandString(8)
	)
	
	//检查是否包含危险操作,如果包含,直接返回失败
	safeReg, _ := regexp.Compile(`(\bos\.|\brm\b|\bremove\b|\bdelete\b)`)
	if safeReg.MatchString(code) {
		dangerStr := safeReg.FindAllString(code, -1) // 查找所有匹配的危险字符
		return nil, errors.New(fmt.Sprintf("当前代码包含危险字符: %v,\n已被系统阻止执行!", strings.Join(dangerStr, " ; ")))
	}

	if 0 < len(imports) {
		importStr = "import ("
		for _, item := range imports {
			if blankInd := strings.Index(item, " "); -1 < blankInd {
				importStr += fmt.Sprintf("\n %s \"%s\"", item[:blankInd], item[blankInd+1:])
			} else {
				importStr += fmt.Sprintf("\n\"%s\"", item)
			}
		}
		importStr += "\n)"
	}
	fullCode = fmt.Sprintf(tmp, importStr, defineCode, code)

	var codeBytes = []byte(fullCode)
	// 格式化输出的代码
	if formatCode, err := format.Source(codeBytes); nil == err {
		// 格式化失败，就还是用 content 吧
		codeBytes = formatCode
	}

	// 创建目录
	if err = os.Mkdir(newTmpDir, os.ModePerm); nil != err {
		return
	}
	defer os.RemoveAll(newTmpDir)
	// 创建文件
	tmpFile, err := os.Create(newTmpDir + dirSeparator + "main.go")
	if err != nil {
		return re, err
	}
	defer os.Remove(tmpFile.Name())
	// 代码写入文件
	tmpFile.Write(codeBytes)
	tmpFile.Close()
	// 运行代码
	cmd := exec.Command("go", "run", tmpFile.Name())
	res, err := cmd.CombinedOutput()
	return res, err
}
