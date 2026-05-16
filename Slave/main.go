package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type ChunkJson struct {
	Content string `json:"content"`
	FileID  int    `json:"file_id"`
	ChunkID int    `json:"chunk_id"`
}

func doShutdown(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "Only Post Methods are allowed", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()

	currentOS := runtime.GOOS
	var cmd *exec.Cmd

	if currentOS == "windows" {
		cmd = exec.Command("shutdown", "-s", "-t", "0")
	} else if currentOS == "linux" || currentOS == "darwin" {
		cmd = exec.Command("shutdown", "now")
	} else {
		fmt.Println("Unsupported OS: " + currentOS)
		http.Error(w, "Unsupported OS: "+currentOS, http.StatusInternalServerError)
		return
	}
	err := cmd.Run()
	if err != nil {
		fmt.Fprintf(w, "Error shutting down: %v", err)
		return
	}
	fmt.Fprintln(w, "Shutting down the computer...")
}

func doFileChunk(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()

	var chunk ChunkJson
	if err := json.NewDecoder(r.Body).Decode(&chunk); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	fmt.Printf("[Slave] Got chunk %d: %s\n", chunk.ChunkID, chunk.Content)

	result := make(map[string]int)
	for _, v := range strings.TrimSpace(chunk.Content) {
		result[string(v)]++
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "ok",
		"result":   result,
		"chunk_id": chunk.ChunkID,
	})
}

func doBackground(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "Only Post Allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, "File is too large", http.StatusBadRequest)
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error in Retreiving file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	fmt.Println("File Received: ", handler.Filename)
	os.MkdirAll(filepath.Join(".", "data", "Background"), os.ModePerm)

	dstPath := "./data/Background/uploaded_" + handler.Filename
	dst, err := os.Create(dstPath)

	if err != nil {
		http.Error(w, "Error in Saving file", http.StatusInternalServerError)
		return
	}

	io.Copy(dst, file)
	dst.Close()

	currentOS := runtime.GOOS
	var cmd *exec.Cmd
	absolutePath, err := filepath.Abs(dstPath)
	if err != nil {
		http.Error(w, "Error Can't Resolve File Path", http.StatusInternalServerError)
		return
	}

	if currentOS == "windows" {
		fmt.Println("OS is Windows")
		normalizedPath := filepath.ToSlash(absolutePath)
		fmt.Printf("Normalized Path: %s\n", normalizedPath)
		psScript := fmt.Sprintf(`$ErrorActionPreference = "Stop"
			$wallpaperPath = "%s"

			if (-not (Test-Path $wallpaperPath)) {
				Write-Host "Error: File not found: $wallpaperPath"
				exit 2
			}

			$code = @'
			using System;
			using System.Runtime.InteropServices;
			public class Win32 {
				[DllImport("user32.dll", CharSet = CharSet.Unicode, SetLastError = true)]
				public static extern int SystemParametersInfo(int uAction, int uParam, string lpvParam, int fuWinIni);
				[DllImport("kernel32.dll")]
				public static extern int GetLastError();
			}
			'@

			try {
				Add-Type -TypeDefinition $code -ErrorAction Stop
				$result = [Win32]::SystemParametersInfo(20, 0, $wallpaperPath, 3)
				
				if ($result -eq 0) {
					$errCode = [Win32]::GetLastError()
					Write-Host "SystemParametersInfo failed. Error Code: $errCode"
					exit $errCode
				}
				
				Write-Host "Success: Wallpaper changed to $wallpaperPath"
				exit 0
			} catch {
				Write-Host "PowerShell Error: $($_.Exception.Message)"
				exit 1
			}`, normalizedPath)

		cmd = exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", psScript)
	} else if currentOS == "linux" {
		fmt.Println("OS is Linux")
		imageURI := "file://" + absolutePath
		// fmt.Printf("%s\n", absolutePath)
		// cmdLight := exec.Command("gsettings", "set", "org.gnome.desktop.background", "picture-uri", imageURI)
		// errcmd := cmdLight.Run()

		// cmd = exec.Command("gsettings", "set", "org.gnome.desktop.background", "picture-uri-dark", imageURI)
		cmd = exec.Command("gsettings", "set", "org.cinnamon.desktop.background", "picture-uri", imageURI)
	} else {
		http.Error(w, "Unsupported OS: "+currentOS, http.StatusInternalServerError)
		return
	}
	if cmd == nil {
		http.Error(w, "Failed to initialize command", http.StatusInternalServerError)
		return
	}
	output, errcmd := cmd.CombinedOutput()
	// fmt.Printf("PowerShell Output:\n%s\n", string(output))

	if errcmd != nil {
		fmt.Printf("PowerShell Error: %v\n", errcmd)
		http.Error(w, "Failed to change background: "+string(output), http.StatusInternalServerError)
		return
	}
	fmt.Println("Set Background Successfully...")
	w.Write([]byte("File Uploaded and Background Set Successfully!"))
}
func main() {
	server := http.Server{
		Addr: "0.0.0.0:9070",
	}

	fmt.Println("Listen on:9070")
	http.HandleFunc("/Shutdown", doShutdown)
	http.HandleFunc("/Background", doBackground)
	http.HandleFunc("/FileChunk", doFileChunk)

	server.ListenAndServe()
}
