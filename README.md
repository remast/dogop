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

## Docker Container bauen

### Cloudnative Buildpacks nutzen

    pack build dogop-cnb --buildpack paketo-buildpacks/go --builder paketobuildpacks/builder-jammy-base

    docker run --network host dogop-cnb