# GoTasks - Distributed Task Manager

A distributed task management system built in Go that allows a master server to control multiple slave machines over HTTP. The system supports remote shutdown, distributed file processing, and remote wallpaper/background changes.

## Architecture

### Master Server (`Master/main.go`)
- Runs on port **9080**
- Acts as the central control node
- Reads slave configuration from `config.json`
- Distributes tasks to slaves and aggregates results

### Slave Server (`Slave/main.go`)
- Runs on port **9070**
- Runs on each target machine
- Receives and executes commands from the master
- Supports Windows and Linux operating systems

## Configuration

Create a `config.json` file in the Master directory:

```json
[
  {
    "ip": "192.168.1.100",
    "slave": 1,
    "slave_name": "pc1"
  },
  {
    "ip": "192.168.1.101",
    "slave": 2,
    "slave_name": "pc2"
  }
]
```

## Endpoints

### Master Endpoints (Port 9080)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/Shutdown` | POST | Shut down a remote machine |
| `/Background` | POST | Change wallpaper on a remote machine |
| `/FileChunk` | POST | Distribute file processing across slaves |

### Slave Endpoints (Port 9070)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/Shutdown` | POST | Execute shutdown command |
| `/Background` | POST | Receive image and set as wallpaper |
| `/FileChunk` | POST | Process a chunk of file data |

## Features

### 1. Remote Shutdown
Send a POST request to the master with the slave name to shut down that machine.

```bash
curl -X POST http://localhost:9080/Shutdown -d "pc1"
```

### 2. File Chunk Processing
Distributes file lines across multiple slaves for parallel character frequency counting.

```bash
curl -X POST http://localhost:9080/FileChunk -d "/path/to/file.txt"
```

### 3. Remote Background Change
Sends an image to a slave and sets it as the desktop wallpaper.

```bash
curl -X POST http://localhost:9080/Background \
  -H "Content-Type: application/json" \
  -d '{"path": "/path/to/image.jpg", "name": "pc1"}'
```

## How It Works

### Shutdown Flow
1. Master receives slave name via `/Shutdown`
2. Master looks up the slave's IP from `config.json`
3. Master forwards the request to `http://<slave_ip>:9070/Shutdown`
4. Slave executes the appropriate OS shutdown command:
   - **Windows**: `shutdown -s -t 0`
   - **Linux/macOS**: `shutdown now`

### File Chunk Flow
1. Master receives a file path via `/FileChunk`
2. Master reads the file line by line
3. Each line is sent to a different slave via `/FileChunk` (round-robin)
4. Each slave counts character frequencies in its chunk
5. Master collects all results and returns aggregated character counts

### Background Change Flow
1. Master receives JSON with image path and slave name via `/Background`
2. Master reads the image file and creates a multipart form request
3. Master sends the image to the slave's `/Background` endpoint
4. Slave saves the image to `./data/Background/uploaded_<filename>`
5. Slave executes OS-specific commands:
   - **Windows**: Uses PowerShell with `SystemParametersInfo` API to set wallpaper
   - **Linux**: Uses `gsettings` to set Cinnamon desktop background

## How to Run

### Start Slave
```bash
cd Slave
go run main.go
```

### Start Master
```bash
cd Master
go run main.go
```

### Build Binaries
```bash
cd Slave && go build -o slave
cd Master && go build -o master
```

## Requirements

- Go 1.18+
- **Windows slaves**: PowerShell available (default on Windows 7+)
- **Linux slaves**: Cinnamon desktop environment with `gsettings`

## Project Structure

```
GoTasks/
├── Master/
│   └── main.go          # Master server code
├── Slave/
│   └── main.go          # Slave server code
├── config.json          # Slave configuration (create manually)
├── README.md            # English documentation
└── README_AR.md         # Arabic documentation
```
