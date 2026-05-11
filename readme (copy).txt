curl -X POST http://127.0.0.1:9080/Shutdown -d "Ahmed"
curl -X POST http://127.0.0.1:9080/FileChunk -d "/media/ahmedelewaa/01D8D2F130BAA800/Projects/GoTasks/Slave/data/Files/gene.fna"

curl -X POST http://127.0.0.1:9080/Background -H "Content-Type: application/json" -d '{"path": "/home/ahmedelewaa/Downloads/Gemini_Generated_Image_czlrlhczlrlhczlr.png","name": "Ahmed"}'

gsettings set org.gnome.desktop.background picture-uri-dark "file://./data/Background/uploaded/Gemini_Generated_Image_czlrlhczlrlhczlr.png"
