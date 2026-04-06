package deploy

// Runtime represents a supported serverless runtime environment.
type Runtime struct {
	ID       string `json:"$id"`
	Name     string `json:"name"`
	Version  string `json:"version"`
	Language string `json:"language"`
	Image    string `json:"image"`   // Docker base image
	Command  string `json:"command"` // Default start command
}

// ListRuntimes returns all supported serverless runtimes.
func ListRuntimes() []Runtime {
	return []Runtime{
		// Node.js
		{ID: "node-18", Name: "Node.js", Version: "18", Language: "javascript", Image: "node:18-alpine", Command: "node index.js"},
		{ID: "node-20", Name: "Node.js", Version: "20", Language: "javascript", Image: "node:20-alpine", Command: "node index.js"},
		{ID: "node-22", Name: "Node.js", Version: "22", Language: "javascript", Image: "node:22-alpine", Command: "node index.js"},

		// Bun
		{ID: "bun-1.0", Name: "Bun", Version: "1.0", Language: "javascript", Image: "oven/bun:1.0-alpine", Command: "bun run index.ts"},
		{ID: "bun-1.1", Name: "Bun", Version: "1.1", Language: "javascript", Image: "oven/bun:1.1-alpine", Command: "bun run index.ts"},

		// Python
		{ID: "python-3.10", Name: "Python", Version: "3.10", Language: "python", Image: "python:3.10-alpine", Command: "python main.py"},
		{ID: "python-3.11", Name: "Python", Version: "3.11", Language: "python", Image: "python:3.11-alpine", Command: "python main.py"},
		{ID: "python-3.12", Name: "Python", Version: "3.12", Language: "python", Image: "python:3.12-alpine", Command: "python main.py"},

		// Go
		{ID: "go-1.23", Name: "Go", Version: "1.23", Language: "go", Image: "golang:1.23-alpine", Command: "go run ."},

		// Dart
		{ID: "dart-3.3", Name: "Dart", Version: "3.3", Language: "dart", Image: "dart:3.3", Command: "dart run bin/main.dart"},
		{ID: "dart-3.5", Name: "Dart", Version: "3.5", Language: "dart", Image: "dart:3.5", Command: "dart run bin/main.dart"},
		{ID: "dart-3.8", Name: "Dart", Version: "3.8", Language: "dart", Image: "dart:3.8", Command: "dart run bin/main.dart"},

		// Ruby
		{ID: "ruby-3.1", Name: "Ruby", Version: "3.1", Language: "ruby", Image: "ruby:3.1-alpine", Command: "ruby main.rb"},
		{ID: "ruby-3.3", Name: "Ruby", Version: "3.3", Language: "ruby", Image: "ruby:3.3-alpine", Command: "ruby main.rb"},

		// PHP
		{ID: "php-8.1", Name: "PHP", Version: "8.1", Language: "php", Image: "php:8.1-cli-alpine", Command: "php index.php"},
		{ID: "php-8.2", Name: "PHP", Version: "8.2", Language: "php", Image: "php:8.2-cli-alpine", Command: "php index.php"},
		{ID: "php-8.3", Name: "PHP", Version: "8.3", Language: "php", Image: "php:8.3-cli-alpine", Command: "php index.php"},

		// Java
		{ID: "java-17", Name: "Java", Version: "17", Language: "java", Image: "eclipse-temurin:17-jre-alpine", Command: "java -jar app.jar"},
		{ID: "java-21", Name: "Java", Version: "21", Language: "java", Image: "eclipse-temurin:21-jre-alpine", Command: "java -jar app.jar"},

		// Kotlin
		{ID: "kotlin-1.9", Name: "Kotlin", Version: "1.9", Language: "kotlin", Image: "eclipse-temurin:21-jre-alpine", Command: "java -jar app.jar"},

		// Swift
		{ID: "swift-5.9", Name: "Swift", Version: "5.9", Language: "swift", Image: "swift:5.9", Command: "swift run"},
		{ID: "swift-5.10", Name: "Swift", Version: "5.10", Language: "swift", Image: "swift:5.10", Command: "swift run"},

		// .NET
		{ID: "dotnet-8.0", Name: ".NET", Version: "8.0", Language: "csharp", Image: "mcr.microsoft.com/dotnet/runtime:8.0-alpine", Command: "dotnet run"},

		// C++
		{ID: "cpp-17", Name: "C++", Version: "17", Language: "cpp", Image: "gcc:13", Command: "g++ -std=c++17 -o app main.cpp && ./app"},
	}
}
