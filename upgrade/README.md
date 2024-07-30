### update
##### update update.exe
```
package main
import "github.com/greedchase/gotools/upgrade"

func main() {
    upgrade.Update()
}
```

### server
##### server server.exe
```
package main
import "github.com/greedchase/gotools/upgrade"
import "github.com/greedchase/gotools/stutil"

func main() {
	stutil.SysDaemon()
    upgrade.StartServer("127.0.0.1:1111")
    upgrade.WaitStopSignal(func() {
        //onclosse
    })
}
```

### client 
##### linuxapp.v1 linuxapp.v3 winapp.exe.v1 winapp.v2
```
package main
import "github.com/greedchase/gotools/upgrade"
import "github.com/greedchase/gotools/stutil"

func main() {
    stutil.SysDaemon()
    
    upgrade.StartClient("127.0.0.1:1111", 0)  //start version 0
    upgrade.WaitStopSignal(func() {
		//onclosse
	})
}
```