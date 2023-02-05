```
go mod init ppix/dogop
```

go get github.com/go-chi/chi/v5

go build -o build/dogop .

docker build . -t dogop

docker run -p 8080:8080 dogop

go get github.com/jackc/pgx/v5/pgxpool
go get github.com/jackc/pgx/v5

go get github.com/golang-migrate/migrate/v4
go get github.com/golang-migrate/migrate/v4/database/pgx
go get github.com/golang-migrate/migrate/v4/source/iofs

go get github.com/hellofresh/health-go/v5
go get github.com/hellofresh/health-go/v5/checks/pgx4

```go
package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func main() {
	r := chi.NewRouter()
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello DogOp!"))
	})
	http.ListenAndServe(":8080", r)
}
```

INSERT INTO
dealer (user_id)
SELECT
id
FROM
rows RETURNING id INTO l_dealerid;
