package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	// "sort"
)

var Slaves map[string]string

type BgJson struct {
	Path string `json:"path"`
	Name string `json:"name"`
}

type ChunkJson struct {
	Content string `json:"content"`
	FileID  int    `json:"file_id"`
	ChunkID int    `json:"chunk_id"`
}

type Slave struct {
	IP         string `json:"ip"`
	Slave_id   int    `json:"slave"`
	Slave_name string `json:"slave_name"`
}

func Config() map[string]string {

	json_file, err := os.Open("./config.json")
	if err != nil {
		fmt.Println("Error in opening file")
		return nil
	}

	json_data, err := io.ReadAll(json_file)
	if err != nil {
		fmt.Println("Error in reading json content")
		return nil
	}

	var slaves []Slave
	json.Unmarshal(json_data, &slaves)
	slaves_mp := make(map[string]string, len(slaves))
	for i := 0; i < len(slaves); i++ {
		slaves_mp[slaves[i].Slave_name] = slaves[i].IP
	}

	return slaves_mp

}

func goShutdown(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "Only Post Methods are allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error in reading message", http.StatusBadRequest)
		return
	}

	defer r.Body.Close()

	msg := string(body)
	var response string
	slave_ip, ok := Slaves[msg]

	if !ok || slave_ip == "" {
		fmt.Println("Can't Found The Name: ", msg)
		response = "Can't Found The Name: " + msg
		http.Error(w, response, http.StatusBadRequest)
		return
	}

	fmt.Println("Received Message: ", msg)
	response = "Message Received The Name: " + msg + "\n"
	w.Write([]byte(response))
	url := "http://" + slave_ip + ":9070" + "/Shutdown"
	resp, err := http.Post(url, "application/json", nil)
	if err != nil {
		fmt.Println("Error making request:", err)
		return
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response:", err)
		return
	}
	fmt.Println("Status Code:", resp.StatusCode)
	fmt.Println("Response from server:", string(responseBody))
	response = "The Pc Name: " + msg + " With IP: " + slave_ip + "\n"
	w.Write([]byte(response))
}

func goFileChunk(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()

	// 1️⃣ اقرأ مسار الملف
	path, _ := io.ReadAll(r.Body)
	file, err := os.Open(strings.TrimSpace(string(path)))
	if err != nil {
		http.Error(w, "Can't open file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// 2️⃣ جهز قائمة الـ Slaves
	slaves := []string{}
	for _, addr := range Slaves {
		slaves = append(slaves, addr)
	}
	if len(slaves) == 0 {
		http.Error(w, "No slaves", http.StatusServiceUnavailable)
		return
	}

	// 3️⃣ channel للنتائج
	results := make(chan bool, 1000)

	// 4️⃣ دالة الإرسال: جرب الـ slaves بالترتيب لحد ما واحد ينجح
	sendToSlave := func(chunk ChunkJson, startIdx int) bool {
		data, _ := json.Marshal(chunk)

		for i := 0; i < len(slaves); i++ {
			addr := slaves[(startIdx+i)%len(slaves)] // دور على الـ slaves
			resp, err := http.Post(
				"http://"+addr+":9070/FileChunk", // ✅ endpoint الصح
				"application/json",
				bytes.NewBuffer(data),
			)
			if err == nil && resp.StatusCode == http.StatusOK {
				resp.Body.Close()
				return true // ✅ نجح
			}
			if resp != nil {
				resp.Body.Close()
			}
		}
		return false // ❌ كلهم فشلوا
	}

	// 5️⃣ اقرأ الملف وابعت كل سطر في goroutine
	chunkID := 0
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		chunk := ChunkJson{Content: line, FileID: 1, ChunkID: chunkID}
		startIdx := chunkID % len(slaves) // الدور على الـ slaves

		go func(c ChunkJson, idx int) {
			results <- sendToSlave(c, idx) // ابعت النتيجة في الـ channel
		}(chunk, startIdx)

		chunkID++
	}
	total := chunkID

	// 6️⃣ عدّ النتائج
	success := 0
	for i := 0; i < total; i++ {
		if <-results {
			success++
		} // اقرأ من الـ channel
	}

	// 7️⃣ اكتب الرد
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{
		"total":   total,
		"success": success,
		"failed":  total - success,
	})
}

func goBackground(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "Only Post Methods are allowed", http.StatusMethodNotAllowed)
		return
	}

	var bgjson BgJson
	err := json.NewDecoder(r.Body).Decode(&bgjson)
	if err != nil {
		http.Error(w, "Invalid Json", http.StatusBadRequest)
		return
	}

	fmt.Println("Received Successfully:")

	file_path := bgjson.Path
	file, err := os.Open(file_path)
	if err != nil {
		http.Error(w, "Can't Open The File", http.StatusBadRequest)
		return
	}
	defer file.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", file_path)
	if err != nil {
		http.Error(w, "Can't Create Form File", http.StatusBadRequest)
		return
	}
	io.Copy(part, file)
	writer.Close()

	var response string
	slave_ip, ok := Slaves[bgjson.Name]

	if !ok || slave_ip == "" {
		fmt.Println("Can't Found The Name: ", bgjson.Name)
		response = "Can't Found The Name: " + bgjson.Name
		http.Error(w, response, http.StatusBadRequest)
		return
	}

	response = "Message Received The Name: " + bgjson.Name + "\n"
	url := "http://" + slave_ip + ":9070" + "/Background"
	req, err := http.NewRequest("POST", url, &body)
	if err != nil {
		http.Error(w, "Can't Create Request to "+url, http.StatusBadRequest)
		return
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Can't Send The Request to "+url, http.StatusBadRequest)
		return
	}
	defer resp.Body.Close()
	fmt.Println("File Uploaded Successfully...")
	w.Write([]byte(response))
}

func main() {
	server := http.Server{
		Addr: "0.0.0.0:9080",
	}
	Slaves = Config()
	fmt.Println("Listen on:9080")
	http.HandleFunc("/Shutdown", goShutdown)
	http.HandleFunc("/Background", goBackground)
	http.HandleFunc("/FileChunk", goFileChunk)

	server.ListenAndServe()

}
