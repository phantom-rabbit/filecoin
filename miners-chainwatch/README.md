### miners-chainwatch
lotus-stats is a small tool to push chain information into mongodb

### Usage
miners-chainwatch will be  to connect to a running lotus daemon and resume collecting stats from last record block height.

```
go build -o miners-chainwatch *.go 
./chainwatch run
```