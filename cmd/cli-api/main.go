package main

func main() {
	server := NewAPP()
	server.Run()
	server.Shutdown()
}
