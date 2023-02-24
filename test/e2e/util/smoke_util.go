package util

import (
	"bufio"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"k8s.io/utils/exec"
)

const label = "app.kubernetes.io/instance"
const name = "mycluster"
const namespace = "default"

func GetFiles(path string) ([]string, error) {
	var result []string
	e := filepath.Walk(path, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			log.Println(err)
			return err
		}
		if !fi.IsDir() {
			if strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") {
				result = append(result, path)
			}
		}
		return nil
	})
	if e != nil {
		log.Println(e)
		return result, e
	}
	return result, nil
}

func GetFolders(path string) ([]string, error) {
	var result []string
	e := filepath.Walk(path, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			log.Println(err)
			return err
		}
		if fi.IsDir() {
			result = append(result, path)
		}
		return nil
	})
	if e != nil {
		log.Println(e)
		return result, e
	}
	return result, nil
}

func CheckClusterStatus() bool {
	cmd := "kubectl get cluster " + name + " -n " + namespace + " | grep " + name + " | awk '{print $5}'"
	log.Println(cmd)
	clusterStatus := ExecCommand(cmd)
	log.Println("clusterStatus is " + clusterStatus)
	return strings.TrimSpace(clusterStatus) == "Running"
}

func CheckPodStatus() map[string]bool {
	var podStatusResult = make(map[string]bool)
	cmd := "kubectl get pod -n " + namespace + " -l '" + label + "=" + name + "'| grep " + name + " | awk '{print $1}'"
	log.Println(cmd)
	arr := ExecCommandReadline(cmd)
	log.Println(arr)
	if len(arr) > 0 {
		for _, podName := range arr {
			command := "kubectl get pod " + podName + " -n " + namespace + " | grep " + podName + " | awk '{print $3}'"
			log.Println(command)
			podStatus := ExecCommand(command)
			log.Println("podStatus is " + podStatus)
			if strings.TrimSpace(podStatus) == "Running" {
				podStatusResult[podName] = true
			} else {
				podStatusResult[podName] = false
			}
		}
	}
	return podStatusResult
}

func OpsYaml(file string, ops string) bool {
	cmd := "kubectl " + ops + " -f " + file
	log.Println(cmd)
	b := ExecuteCommand(cmd)
	return b
}

func ExecuteCommand(command string) bool {
	exeFlag := true
	cmd := exec.New().Command("bash", "-c", command)
	// Create a fetch command output pipe
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("Error:can not obtain stdout pipe for command:%s\n", err)
		exeFlag = false
	}
	// Create a fetch command err output pipe
	stderr, err1 := cmd.StderrPipe()
	if err1 != nil {
		log.Printf("Error:can not obtain stderr pipe for command:%s\n", err1)
		exeFlag = false
	}
	// Run command
	if err := cmd.Start(); err != nil {
		log.Printf("Error:The command is err:%s\n", err)
		exeFlag = false
	}
	// Use buffered readers
	go asyncLog(stdout)
	go asyncLog(stderr)
	if err := cmd.Wait(); err != nil {
		log.Printf("wait:%s\n", err.Error())
		exeFlag = false
	}
	return exeFlag
}

func asyncLog(reader io.Reader) {
	cache := ""
	buf := make([]byte, 1024)
	for {
		num, err := reader.Read(buf)
		if err != nil && errors.Is(err, io.EOF) {
			return
		}
		if num > 0 {
			b := buf[:num]
			s := strings.Split(string(b), "\n")
			line := strings.Join(s[:len(s)-1], "\n")
			log.Printf("%s%s\n", cache, line)
			cache = s[len(s)-1]
		}
		if errors.Is(err, io.EOF) {
			break
		}
	}
}

func ExecCommandReadline(strCommand string) []string {
	var arr []string
	cmd := exec.New().Command("/bin/bash", "-c", strCommand)
	stdout, _ := cmd.StdoutPipe()
	if err := cmd.Start(); err != nil {
		log.Printf("Execute failed when Start:%s\n\n", err.Error())
	}
	reader := bufio.NewReader(stdout)
	for {
		line, err := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if err != nil || io.EOF == err {
			break
		}
		arr = append(arr, line)
	}
	if err := cmd.Wait(); err != nil {
		log.Printf("wait:%s\n\n", err.Error())
	}
	return arr
}

func ExecCommand(strCommand string) string {
	cmd := exec.New().Command("/bin/bash", "-c", strCommand)
	stdout, _ := cmd.StdoutPipe()
	if err := cmd.Start(); err != nil {
		log.Printf("Execute failed when Start:%s\n", err.Error())
		return ""
	}
	outBytes, _ := io.ReadAll(stdout)
	err := stdout.Close()
	if err != nil {
		return ""
	}
	if err := cmd.Wait(); err != nil {
		log.Printf("Execute failed when Wait:%s\n", err.Error())
		return ""
	}
	return string(outBytes)
}

func WaitTime() {
	wg := sync.WaitGroup{}
	wg.Add(int(400000000))
	for i := 0; i < int(400000000); i++ {
		go func(i int) {
			defer wg.Done()
		}(i)
	}
	wg.Wait()
}

func GetClusterCreateYaml(files []string) string {
	for _, file := range files {
		if strings.Contains(file, "00") {
			return file
		}
	}
	return ""
}
