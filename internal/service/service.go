package service

import "errors"

// 通用错误定义
var (
	ErrValidation = func(msg string) error { return &ServiceError{Msg: msg, Code: "VALIDATION"} }
	ErrInternal   = func(msg string) error { return &ServiceError{Msg: msg, Code: "INTERNAL"} }
	ErrNotFound   = &ServiceError{Msg: "资源不存在", Code: "NOT_FOUND"}

	ErrNotMember    = errors.New("不是该宝宝的家庭成员")
	ErrNotAdmin     = errors.New("不是该宝宝的管理员")
	ErrForbidden    = errors.New("无权限执行此操作")
	ErrUnauthorized = errors.New("请先登录")
)

// ServiceError 业务错误，包含机器可读的错误码
type ServiceError struct {
	Msg  string
	Code string
}

func (e *ServiceError) Error() string {
	return e.Msg
}
