# Go über den Wolken - Code Beispiel

## Nützliche Befehle

- Tests ausführen mit: `go test -v ./...`

- Anwendung bauen: `go build -o build/dogop .`

- Anwendung ausführen: `go run .`

- Ausführen mit Hot Reload über [air](https://github.com/cosmtrek/air): `air`

- Go Dokumentation lesen: `go doc http.HandlerFunc`

- Docker Container bauen: `docker build . -t crossnative/dogop`

## Projekt aufsetzen

Go Modul erstellen mit `go mod init crossnative/dogop`.

Erste Dependency einbinden mit `go get github.com/go-chi/chi/v5`.

