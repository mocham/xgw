package main

import (
	"fmt"
	"os"
	"strings"
	"path/filepath"
	"html"
	"net"
	"io"
	"net/http"
	"net/url"
)
const (
    socketPath = "/tmp/bar.sock"
)

func doAction(action ...string) {
	data := []byte(strings.Join(action, "\n"))
	fmt.Printf("%s", ipcSend(data))
}

func ipcSend(data []byte) []byte{
    data = append(data, 0)
    // Connect to the Unix domain socket
    conn, err := net.Dial("unix", socketPath)
    if err != nil {
        fmt.Printf("Failed to connect: %v\n", err)
        os.Exit(1)
    }
    defer conn.Close()
    // Send the message
    _, err = conn.Write([]byte(data))
    if err != nil {
        fmt.Printf("Failed to send message: %v\n", err)
        os.Exit(1)
    }
    response, err := io.ReadAll(conn)
    if err != nil {
        fmt.Printf("Failed to read: %v\n", err)
        os.Exit(1)
    }
    return response
}


func main() {
	http.HandleFunc("/files/", filesHandler)
	// Define a handler function for the root path
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, homePage())
	})

	// Start the server on port 8080
	fmt.Println("Server starting on http://localhost:8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("Error starting server:", err)
	}
}

func homePage() string {
	return `
<html>
<meta charset="UTF-8">
<head>
	<title>Home Page</title>
</head>
<body>
<span>
<a href="~/">HOME</a>
</span>
</body>
</html>`
}

func isDir(path string) bool {
    fileInfo, err := os.Stat(path)
    if err != nil {
        return false
    }

    return fileInfo.IsDir()
}

func barOpen(args ...string) {
	if len(args) < 3 { return }
	if strings.HasPrefix(args[2], "rar:///") {
		args[2] = args[2][6:]
	} else if strings.HasPrefix(args[2], "zip:///") {
		args[2] = args[2][6:]
	}
	fmt.Printf("Trying to open %s\n", args[2])
	arr := strings.Split(args[2], "/file://")
	if len(arr) == 0 { return }
	matches, err := matchOSFiles(arr[0])
	fmt.Printf("Matching %s <- %v, err %v\n", arr[0], matches, err)
	if len(matches) == 0 || err != nil { return }
	fn := matches[0]
	if len(arr) > 1 {
		var target string
		if strings.HasSuffix(fn, ".zip") {
			target, err = extractFileFromZip(fn, arr[1], "/tmp/zip-extracted")
			if err != nil { fmt.Println("Error extracting file %v", err); return }
		} else if strings.HasSuffix(fn, ".rar") {
			target, err = extractFileFromRar(fn, arr[1], "/tmp/zip-extracted")
			if err != nil { fmt.Println("Error extracting file %v", err); return }
		}
		ipcSend([]byte(fmt.Sprintf("OS\nOPEN\n%s", target)))
	} else {
		ipcSend([]byte(fmt.Sprintf("OS\nOPEN\n%s", fn)))
	}
}

// filesHandler handles requests to /files/* and lists files in the requested directory
func filesHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the path after /files/
	requestedPath := strings.TrimPrefix(r.URL.Path, "/files/")
	requestedPath , _ = url.PathUnescape(requestedPath)
	fmt.Printf("URL RAW %s", requestedPath)
	if requestedPath == "" {
		requestedPath = os.Getenv("HOME")
	}

	if !strings.HasPrefix(requestedPath, "/") {
		requestedPath = "/" + requestedPath
	}
	if strings.HasSuffix(requestedPath, "TERM") {
		doAction("OS", "CDXTERM", requestedPath[:len(requestedPath)-4])
		w.WriteHeader(http.StatusNoContent)
		return 
	}
	var err error
	var entries []string
	if strings.Contains(requestedPath, "/zip:/") {
		arr := strings.Split(requestedPath, "/zip:/")
		if arr[1] == "" {
			entries = getFileListZip(arr[0])
		} else {
			barOpen("bar", "open", requestedPath[1:])
			w.WriteHeader(http.StatusNoContent)
			return 
		}
	} else {
		if !isDir(requestedPath) {
			doAction("OS", "OPEN", requestedPath)
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// Read directory contents
		entries, err = filepath.Glob(filepath.Join(requestedPath, "*"))
		if err != nil {
			http.Error(w, "Error reading directory", http.StatusInternalServerError)
			return
		}
	}

	// Start building the HTML response
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<meta charset="UTF-8">
<head>
	<title>File Listing</title>
	<style>
		table { border-collapse: collapse; width: 100%%; }
		th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
		th { background-color: #f2f2f2; }
		tr:hover { background-color: #f5f5f5; }
	</style>
</head>
<body>
	<h2>Files in <a href="/files/%s/TERM">/%s</a></h2>
	<table>
		<tr><th>File Name</th></tr>`, requestedPath, html.EscapeString(requestedPath))

	// List files in the directory
	for _, entry := range entries {
		linkPath := filepath.Join(requestedPath, filepath.Base(entry))
		fmt.Printf("Linkpath %s <-> %s", linkPath, entry)

		// Add table row with file link
		if strings.HasSuffix(entry, ".zip") {
			fmt.Fprintf(w, `<tr><td><a href="/files/%s/zip://">%s</a></td></tr>`,
				url.PathEscape(entry), html.EscapeString(filepath.Base(entry)))
		} else {
			arr := strings.Split(entry, "/file:/")
			if len(arr) > 1 {
				fmt.Fprintf(w, `<tr><td><a href="/files/%s">%s</a></td></tr>`,
					url.PathEscape(entry), html.EscapeString(arr[1]))
			} else {
				fmt.Fprintf(w, `<tr><td><a href="/files/%s">%s</a></td></tr>`,
					url.PathEscape(linkPath), html.EscapeString(filepath.Base(entry)))
			}
		}
	}

	// Close HTML
	fmt.Fprint(w, `
	</table>
</body>
</html>`)
}
