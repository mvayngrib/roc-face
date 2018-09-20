// Compare to examples/roc_example_host_id.c

package main
import (
    "fmt"
    "os"
)

// #cgo LDFLAGS: -lroc
// #include <roc.h>
import "C"

func main() {
    if len(os.Args) != 1 {
        C.roc_ensure(C.CString("No arguments expected:\n" +
                               "    $ go run roc_example_host_id.go"))
    }

    // Print host id
    var str C.roc_string
    C.roc_get_host_id(&str)
    fmt.Println(C.GoString(str))
    C.roc_free_string(&str)
}
