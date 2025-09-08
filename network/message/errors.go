package message

import "fmt"

// ErrorCode 错误代码类型
type ErrorCode int

const (
	ErrQueueFull ErrorCode = iota + 1000
	ErrMessageExists
	ErrMessageNotFound
	ErrInvalidMessage
	ErrTimeout
	ErrNetworkError
)

// QueueError 队列错误
type QueueError struct {
	Message string
	Code    ErrorCode
}

// NewQueueError 创建队列错误
func NewQueueError(message string, code ErrorCode) *QueueError {
	return &QueueError{
		Message: message,
		Code:    code,
	}
}

// Error 实现error接口
func (e *QueueError) Error() string {
	return fmt.Sprintf("队列错误[%d]: %s", e.Code, e.Message)
}

// GetCode 获取错误代码
func (e *QueueError) GetCode() ErrorCode {
	return e.Code
}

// IsCode 检查是否为指定错误代码
func (e *QueueError) IsCode(code ErrorCode) bool {
	return e.Code == code
}