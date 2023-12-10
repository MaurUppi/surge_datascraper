package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/joho/godotenv"
)

func main() {
	// 加载 .env 文件
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// 从 .env 文件中读取 EXECUTION_INTERVAL
	executionInterval, exists := os.LookupEnv("EXECUTION_INTERVAL")
	if !exists {
		log.Fatal("EXECUTION_INTERVAL must be set in .env")
	}

	// 如果 EXECUTION_INTERVAL 赋值为 "0"，则不运行定时任务
	if executionInterval == "0" {
		log.Println("EXECUTION_INTERVAL is set to 0, skipping periodic execution")
		return
	}

	// 解析间隔时间
	interval, err := time.ParseDuration(executionInterval)
	if err != nil {
		log.Fatal("Invalid execution interval: ", err)
	}

	// 创建定时器
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// 定时执行任务
	for range ticker.C {
		executeTask()
	}
}

func executeTask() {
	// 加载 .env 文件
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// 从 .env 文件中读取 STRUCTURE_FILE_PATH
	structureFilePath, exists := os.LookupEnv("STRUCTURE_FILE_PATH")
	if !exists {
		log.Fatal("STRUCTURE_FILE_PATH must be set in .env")
	}

	// 读取并解析 structure.json 文件
	structureData, err := os.ReadFile(structureFilePath)
	if err != nil {
		log.Fatal("Error reading structure file: ", err)
	}
	var structure map[string]string
	if err := json.Unmarshal(structureData, &structure); err != nil {
		log.Fatal("Error parsing structure file: ", err)
	}

	// 读取 Debug 有关变量
	debug, _ := os.LookupEnv("DEBUG")
	var debugLogWriter *bufio.Writer

	if debug == "1" {
		debugLogPath, exists := os.LookupEnv("DEBUG_LOG")
		if !exists {
			log.Fatal("DEBUG_LOG must be set in .env")
		}

		debugLogFile, err := os.OpenFile(debugLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal("Failed to open debug log file: ", err)
		}
		defer debugLogFile.Close()
		debugLogWriter = bufio.NewWriter(debugLogFile)
	}

	// 读取其他环境变量
	endpoint, exists := os.LookupEnv("ENDPOINT")
	if !exists {
		log.Fatal("ENDPOINT must be set in .env")
	}
	xKey, exists := os.LookupEnv("X_KEY")
	if !exists {
		log.Fatal("X_KEY must be set in .env")
	}

	// 创建 HTTP 客户端和请求
	client := &http.Client{}
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("X-key", xKey)

	// 发送请求并获取响应
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	// 解析 JSON 数据
	var jsonData map[string]interface{}
	if err := json.Unmarshal(body, &jsonData); err != nil {
		log.Fatal(err)
	}

	// 扁平化处理 requests 数据
	requests, ok := jsonData["requests"].([]interface{})
	if !ok {
		log.Fatal("Error: 'requests' key is not an array")
	}

	// filterAndFlattenRequest 函数调用
	var filteredRequests []map[string]interface{}
	for _, req := range requests {
		requestMap := req.(map[string]interface{})

		// 确保 id 字段为字符串类型
		if id, exists := requestMap["id"]; exists {
			requestMap["id"] = fmt.Sprintf("%v", id)
		}

		filteredReq := filterAndFlattenRequest(requestMap, structure)
		filteredRequests = append(filteredRequests, filteredReq)
	}

	// InfluxDB 配置
	influxURL, exists := os.LookupEnv("INFLUXDB_URL")
	if !exists {
		log.Fatal("INFLUXDB_URL must be set in .env")
	}
	influxToken, exists := os.LookupEnv("INFLUXDB_TOKEN")
	if !exists {
		log.Fatal("INFLUXDB_TOKEN must be set in .env")
	}
	influxOrg, exists := os.LookupEnv("INFLUXDB_ORG")
	if !exists {
		log.Fatal("INFLUXDB_ORG must be set in .env")
	}
	influxBucket, exists := os.LookupEnv("INFLUXDB_BUCKET")
	if !exists {
		log.Fatal("INFLUXDB_BUCKET must be set in .env")
	}
	influxMeasurement, exists := os.LookupEnv("INFLUXDB_MEASUREMENT")
	if !exists {
		log.Fatal("INFLUXDB_MEASUREMENT must be set in .env")
	}

	logFilePath, exists := os.LookupEnv("Execution_log")
	if !exists {
		log.Fatal("Execution_log must be set in .env")
	}
	// 打开日志文件以记录写入操作
	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal("Failed to open log file: ", err)
	}
	defer logFile.Close()
	logWriter := bufio.NewWriter(logFile)

	// 创建 InfluxDB 客户端
	influxClient := influxdb2.NewClient(influxURL, influxToken)
	writeAPI := influxClient.WriteAPIBlocking(influxOrg, influxBucket)

	allSuccess := true // 用于跟踪是否所有记录都成功写入

	// 遍历并写入数据
	for _, req := range filteredRequests {
		p := influxdb2.NewPointWithMeasurement(influxMeasurement)

		// 初始化变量用于存储转换后的时间
		var pointTime time.Time
		var startDateFound bool

		for k, v := range req {
			if k == "id" {
				// id 作为 tag，并通确保是string类型
				p.AddTag(k, fmt.Sprintf("%v", v))
				// #1 _, _ = fmt.Fprintf(logWriter, "[%s] Tag added: %s=%v\n", time.Now().Format("2006-01-02 15:04:05"), k, v)
			} else if k == "startDate" {
				if startDate, ok := v.(float64); ok {
					seconds := int64(startDate)
					nanoseconds := int64((startDate - float64(seconds)) * 1e9)
					pointTime = time.Unix(seconds, nanoseconds)
					startDateFound = true
					// #2 _, _ = fmt.Fprintf(logWriter, "[%s] startDate processed: %v\n", time.Now().Format("2006-01-02 15:04:05"), pointTime)
				} else {
					logMsg := fmt.Sprintf("[%s] Invalid startDate format for request: %v\n", time.Now().Format("2006-01-02 15:04:05"), req)
					_, _ = fmt.Fprint(logWriter, logMsg)
					log.Print(logMsg)
					allSuccess = false
					break
				}
			} else {
				p.AddField(k, v)
				//#3 _, _ = fmt.Fprintf(logWriter, "[%s] Field added: %s=%v\n", time.Now().Format("2006-01-02 15:04:05"), k, v)
			}
		}

		if !startDateFound || !allSuccess {
			// 如果 startDate 未找到或其他错误发生
			_, _ = fmt.Fprintf(logWriter, "[%s] Error: startDate not found or other error occurred\n", time.Now().Format("2006-01-02 15:04:05"))
			break
		}

		// 在这里记录调试信息
		if debug == "1" {
			_, _ = fmt.Fprintf(debugLogWriter, "[%s] Processed request: %v\n", time.Now().Format("2006-01-02 15:04:05"), req)
			_ = debugLogWriter.Flush()
		}

		if !startDateFound || !allSuccess {
			_, _ = fmt.Fprintf(logWriter, "[%s] Error: startDate not found or other error occurred\n", time.Now().Format("2006-01-02 15:04:05"))
			break
		}

		p.SetTime(pointTime)
		//#4 currentTime := time.Now().Format("2006-01-02 15:04:05")
		if err := writeAPI.WritePoint(context.Background(), p); err != nil {
			//	_, _ = fmt.Fprintf(logWriter, "[%s] Failed to write point: %v\n", currentTime, err)
			allSuccess = false
			break // 恢复Execution.log记录详细，则删除这行，以及uncomment #1-#6.
			//#5} else {
			//#6	_, _ = fmt.Fprintf(logWriter, "[%s] Point written successfully\n", currentTime)
		}
	}

	// 循环结束后，记录整体的成功或失败状态
	if allSuccess {
		_, _ = fmt.Fprintf(logWriter, "[%s] All Points written successfully\n", time.Now().Format("2006-01-02 15:04:05"))
	} else {
		_, _ = fmt.Fprintf(logWriter, "[%s] Not all points were written successfully\n", time.Now().Format("2006-01-02 15:04:05"))
	}

	if err := logWriter.Flush(); err != nil {
		log.Fatal("Failed to flush log file: ", err)
	}

	influxClient.Close()
	fmt.Println("Data has been written to InfluxDB")

}

// 如下是自定义函数
func filterAndFlattenRequest(request map[string]interface{}, structure map[string]string) map[string]interface{} {
	filtered := make(map[string]interface{})
	for key, dataType := range structure {
		if strings.Contains(key, "timingRecords.") {
			handleTimingRecord(request, filtered, key, dataType)
		} else if value, exists := request[key]; exists {
			filtered[key] = value
		} else {
			// 对于在 structure 中定义但在 request 中缺失的字段，赋值为 "Null"
			filtered[key] = "Null"
		}
	}
	return filtered
}

func handleTimingRecord(request map[string]interface{}, filtered map[string]interface{}, key string, dataType string) {
	timingRecords, exists := request["timingRecords"].([]interface{})
	if !exists {
		return
	}

	timingKey := strings.TrimPrefix(key, "timingRecords.")
	formattedTimingKey := "timingRecords" + strings.ReplaceAll(timingKey, " ", "")
	for _, record := range timingRecords {
		if rec, ok := record.(map[string]interface{}); ok {
			if name, ok := rec["name"].(string); ok && name == timingKey {
				// 首先尝试将 durationInMillisecond 作为浮点数处理，并将其转换为整数。
				if duration, ok := rec["durationInMillisecond"].(float64); ok {
					filtered[formattedTimingKey] = int(duration)
					// 如果 durationInMillisecond 已经是整数，则直接使用
				} else if duration, ok := rec["durationInMillisecond"].(int); ok {
					filtered[formattedTimingKey] = duration
				} else {
					// 可能的错误处理或默认值
					filtered[formattedTimingKey] = 0
				}
				return
			}
		}
	}

	// 如果 timingRecords 中没有找到指定的 key，则赋值为 0
	filtered[formattedTimingKey] = 0
}
