# RedSock error handling library with tracing

## Examples

##### Error created via "New" function
```go
package main

import (
	"net/http"
	"os"

	errors "go.redsock.ru/rerrors"
)

func main() {
	err := doer()
	if err != nil {
		_, _ = os.Stderr.Write([]byte(err.Error()))
	}
}

func doer() error {
	resp, _ := http.Get("https://redsock.ru")

	if resp.StatusCode == http.StatusOK {
		return errors.New("Success! Error!")
	}

	panic("this never meant to happen")
}

```

##### Will return following output that is JetBrains compatible and you can simply click to go to function

```text

Success! Error!

main.doer()
        /Users/your-user/path/to/project/main.go:21

```

##### Error enrichment with "Wrap" function 
```go
package main

import (
	"net/http"
	"os"

	errors "go.redsock.ru/rerrors"
)

func main() {
	err := doer()
	if err != nil {
		_, _ = os.Stderr.Write([]byte(err.Error()))
	}
}

func doer() error {
	_, err := http.Get("htps://redsock.ru/42")
	if err != nil {
		return errors.Wrap(err, "unknown host i guess. Maybe you should try use https://redsock.ru")
	}

	panic("this never meant to happen")
}

```

##### Will return following output

```text
Get "htps://redsock.ru/42": unsupported protocol scheme "htps"
unknown host i guess. Maybe you should try use https://redsock.ru

main.doer()
        /Users/alexbukov/Yandex.Disk.localized/redsock/trace-errors/main/main.go:20

```


## Tracing feature is **enabled by default**
In order to disable it - run go builder with tag rscliErrorTracingDisabled

#### Example

```shell
  go build -tags rscliErrorTracingDisabled
```