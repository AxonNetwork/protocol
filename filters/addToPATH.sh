go build encode/main.go
mv main /usr/local/bin/conscience_encode
go build decode/main.go
mv main /usr/local/bin/conscience_decode
go build diff/main.go
mv main /usr/local/bin/conscience_diff
