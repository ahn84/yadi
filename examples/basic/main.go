package main

import (
	"fmt"
	"log"

	di "github.com/ahn84/yadi"
)

// Example interfaces and implementations
type Database interface {
	Connect() error
	Query(sql string) string
}

type UserService interface {
	GetUser(id int) string
	CreateUser(name string) int
}

type Logger interface {
	Log(message string)
}

// Implementations
type postgresDB struct {
	connected bool
}

func (p *postgresDB) Connect() error {
	p.connected = true
	fmt.Println("Connected to PostgreSQL")
	return nil
}

func (p *postgresDB) Query(sql string) string {
	return fmt.Sprintf("Result for: %s", sql)
}

type userServiceImpl struct {
	db     Database
	logger Logger
}

func (u *userServiceImpl) GetUser(id int) string {
	u.logger.Log(fmt.Sprintf("Getting user %d", id))
	result := u.db.Query(fmt.Sprintf("SELECT * FROM users WHERE id = %d", id))
	return result
}

func (u *userServiceImpl) CreateUser(name string) int {
	u.logger.Log(fmt.Sprintf("Creating user %s", name))
	u.db.Query(fmt.Sprintf("INSERT INTO users (name) VALUES ('%s')", name))
	return 1
}

type consoleLogger struct{}

func (c *consoleLogger) Log(message string) {
	fmt.Printf("[LOG] %s\n", message)
}

func main() {
	// Create a new DI container
	container := di.New()

	// Bind dependencies using the clean interface{} API
	err := container.Bind(func() Database {
		db := &postgresDB{}
		db.Connect()
		return db
	})
	if err != nil {
		log.Fatalf("Failed to bind Database: %v", err)
	}

	err = container.Bind(func() Logger {
		return &consoleLogger{}
	})
	if err != nil {
		log.Fatalf("Failed to bind Logger: %v", err)
	}

	err = container.Bind(func(db Database, logger Logger) UserService {
		return &userServiceImpl{
			db:     db,
			logger: logger,
		}
	})
	if err != nil {
		log.Fatalf("Failed to bind UserService: %v", err)
	}

	// Resolve dependencies using the clean interface{} API
	var userService UserService
	err = container.Resolve(&userService)
	if err != nil {
		log.Fatalf("Failed to resolve UserService: %v", err)
	}

	// Use the resolved service
	fmt.Println("\n=== Using the DI Container ===")
	user := userService.GetUser(1)
	fmt.Printf("Retrieved user: %s\n", user)

	userID := userService.CreateUser("John Doe")
	fmt.Printf("Created user with ID: %d\n", userID)

	// Demonstrate that transient instances are different
	fmt.Println("\n=== Demonstrating Transient Instances ===")
	var db1, db2 Database
	container.Resolve(&db1)
	container.Resolve(&db2)
	fmt.Printf("DB1 and DB2 are different instances: %v\n", db1 != db2)
}
