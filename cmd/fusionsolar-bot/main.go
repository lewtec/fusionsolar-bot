package main

// main is the minimal entry point for the application.
// It delegates all CLI logic, including flag parsing and command routing,
// to the Cobra execution model defined in root.go.
func main() {
	Execute()
}
