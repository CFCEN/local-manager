package api

const (
	ErrCodeFileNotFound            = "FILE_NOT_FOUND"
	ErrCodeFileNotReadable         = "FILE_NOT_READABLE"
	ErrCodeFileNotWritable         = "FILE_NOT_WRITABLE"
	ErrCodeFileAlreadyExists       = "FILE_ALREADY_EXISTS"
	ErrCodeInvalidPath             = "INVALID_PATH"
	ErrCodeInvalidCommand          = "INVALID_COMMAND"
	ErrCodeApplicationNotFound     = "APPLICATION_NOT_FOUND"
	ErrCodeApplicationAlreadyRunning = "APPLICATION_ALREADY_RUNNING"
	ErrCodeApplicationNotRunning   = "APPLICATION_NOT_RUNNING"
	ErrCodeApplicationStartFailed  = "APPLICATION_START_FAILED"
	ErrCodeApplicationStopFailed   = "APPLICATION_STOP_FAILED"
	ErrCodeGitProjectNotFound      = "GIT_PROJECT_NOT_FOUND"
	ErrCodeGitScanFailed           = "GIT_SCAN_FAILED"
	ErrCodeGroupNotFound           = "GROUP_NOT_FOUND"
	ErrCodeContextNotFound         = "CONTEXT_NOT_FOUND"
	ErrCodeLogFileNotFound         = "LOG_FILE_NOT_FOUND"
	ErrCodeLogGroupNotFound        = "LOG_GROUP_NOT_FOUND"
	ErrCodeInternal                = "INTERNAL_ERROR"
	ErrCodeBadRequest              = "BAD_REQUEST"
)
