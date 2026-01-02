```go
readProcFile("/sys/class/thermal/thermal_zone0/temp")
readProcFile("/proc/stat")
readProcFile("/proc/uptime")
readProcFile("/tmp/resolv.conf.ppp")
readProcFile("/proc/net/route")
readProcFile("/proc/mounts")
readProcFile("/proc/diskstats")
readProcFile("/proc/net/dev")
readProcFile("/proc/meminfo")
readProcFile("/proc/net/nf_conntrack"),readProcFile("/proc/net/ip_conntrack")
readProcFile("/proc/version")
readProcFile("/proc/device-tree/model")
readProcFile("/proc/sys/kernel/hostname")

runCommand("uname", "-r")
runCommand("uname", "-m")
runCommand("ip", "-o", "addr", "show")
```


```go

reader FsReader,


reader.ReadFile(reader.paths.StorageDeviceMounts())

```