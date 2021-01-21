module github.com/go-msvc/jweb

go 1.14

require (
	github.com/go-msvc/japp v0.0.0-00010101000000-000000000000
	github.com/go-msvc/jcli v0.0.0-00010101000000-000000000000
	github.com/go-msvc/logger v0.0.0-20210121062433-1f3922644bec
	github.com/gorilla/context v1.1.1 // indirect
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/gorilla/pat v1.0.1
	github.com/gorilla/sessions v1.2.1
)

replace github.com/go-msvc/logger => ../logger

replace github.com/go-msvc/japp => ../japp

replace github.com/go-msvc/jcli => ../jcli

replace github.com/go-msvc/jsessions => ../jsessions
