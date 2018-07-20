go build push-hook/main.go
mv main ../$1/.git/hooks/pre-push
go build pull-hook/main.go
mv main ../$1/.git/hooks/post-merge