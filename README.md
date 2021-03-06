Bandicoot bindings for [Go](http://golang.org). To install run:
``` bash
$ go get github.com/bandilab/bind-go
```

Here is a sample Go program which reads data from a bandicoot instance running on `http://localhost:12345`:
``` Go
import "github.com/bandilab/bind-go"

type Book struct {
    Title string
    Pages int
    Price real
}

func main() {
    bandicoot.URL("http://localhost:12345")

    var books []Book
    if err := bandicoot.Get("ListBooks?maxPrice=10.0", &books); err != nil {
        fmt.Printf("error %v", err)
    }
    for _, b := range books {
        fmt.Printf("%+v\n", b)
    }
}
```

Writes can be handled as follows:

``` Go
books := []Book{Book{Title: "Robinson Crusoe", Pages: 312, Price: 11.21}}
bandicoot.Post("AddBooks", books, nil)
```

See `go doc bandicoot` for more information.
