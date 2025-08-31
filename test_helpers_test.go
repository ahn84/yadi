package di

// Test interfaces and structs
type Database interface {
	Connect() error
}

type UserService interface {
	GetUser(id int) string
}

type Logger interface {
	Log(message string)
}

type mockDatabase struct {
	connected bool
}

func (m *mockDatabase) Connect() error {
	m.connected = true
	return nil
}

type userServiceImpl struct {
	db Database
}

func (u *userServiceImpl) GetUser(id int) string {
	return "user"
}

type loggerImpl struct {
	messages []string
}

func (l *loggerImpl) Log(message string) {
	l.messages = append(l.messages, message)
}

// Complex service with multiple dependencies
type OrderService interface {
	CreateOrder(userID int) string
}

type orderServiceImpl struct {
	userService UserService
	db          Database
	logger      Logger
}

func (o *orderServiceImpl) CreateOrder(userID int) string {
	o.logger.Log("Creating order")
	user := o.userService.GetUser(userID)
	return "order for " + user
}
