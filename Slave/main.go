package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func doShutdown(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "Only Post Methods are allowed", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()

	os := runtime.GOOS
	var cmd *exec.Cmd

	if os == "windows" {
		cmd = exec.Command("shutdown", "-s", "-t", "0")
	} else if os == "linux" || os == "darwin" {
		cmd = exec.Command("shutdown", "now")
	} else {
		fmt.Println("نظام تشغيل غير مدعوم للإغلاق التلقائي.")
		return
	}
	err := cmd.Run()
	if err != nil {
		fmt.Fprintf(w, "Error shutting down: %v", err)
		return
	}
	fmt.Fprintln(w, "Shutting down the computer...")

}
func doBackground(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		http.Error(w, "Only Post Allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseMultipartForm(10 << 20) //Up to 10 MB is stored in RAM other goes to temp files...
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

	dstPath := "./data/Background/uploaded" + handler.Filename
	dst, err := os.Create(dstPath)

	if err != nil {
		http.Error(w, "Error in Saving file", http.StatusInternalServerError)
		return
	}

	defer dst.Close()

	io.Copy(dst, file)
	// w.Write([]byte("File uploaded Successfully..."))
	os := runtime.GOOS
	var cmd *exec.Cmd
	absolutePath, err := filepath.Abs(dstPath)
	if err != nil {
		// التعامل مع الخطأ
	}

	if os == "windows" {
		fmt.Println("OS is Windows")
		// // استخدمنا %q عشان نضمن وجود كوتيشن حول المسار لو فيه مسافات
		// psScript := fmt.Sprintf(`$code = '
		// 	using System;
		// 	using System.Runtime.InteropServices;
		// 	public class Win32 {
		// 		[DllImport("user32.dll", CharSet = CharSet.Auto)]
		// 		public static extern int SystemParametersInfo(int uAction, int uParam, string lpvParam, int fuWinIni);
		// 	}';
		// 	Add-Type -TypeDefinition $code;
		// 	[Win32]::SystemParametersInfo(20, 0, "%s", 3)`, absolutePath)

		// cmd = exec.Command("powershell", "-Command", psScript)

		dst.Sync()
		dst.Close()
		normalizedPath := filepath.ToSlash(absolutePath)
		psScript := fmt.Sprintf(`$code = @'
		using System;
		using System.Runtime.InteropServices;
		public class Win32 {
			[DllImport("user32.dll", CharSet = CharSet.Unicode)]
			public static extern int SystemParametersInfo(int uAction, int uParam, string lpvParam, int fuWinIni);
		}
		'@
		Add-Type -TypeDefinition $code
		# 20 = SPI_SETDESKWALLPAPER, 3 = SPIF_UPDATEINIFILE -bor SPIF_SENDCHANGE
		[Win32]::SystemParametersInfo(20, 0, @"%s"@, 3)`, normalizedPath)

		cmd = exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", psScript)

	} else if os == "linux" {
		fmt.Println("OS is Linux")
		imageURI := "file://" + absolutePath
		// fmt.Printf("%s\n", absolutePath)
		// cmdLight := exec.Command("gsettings", "set", "org.gnome.desktop.background", "picture-uri", imageURI)
		// errcmd := cmdLight.Run()

		// cmd = exec.Command("gsettings", "set", "org.gnome.desktop.background", "picture-uri-dark", imageURI)
		cmd = exec.Command("gsettings", "set", "org.cinnamon.desktop.background", "picture-uri", imageURI)
	} else {
		fmt.Printf("Error Another OS %s\n", os)
		return
	}
	errcmd := cmd.Run()
	if errcmd != nil {
		fmt.Printf("Error changing background: %v\n", errcmd)
		http.Error(w, "Failed to change background", http.StatusInternalServerError)
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

	server.ListenAndServe()

}
