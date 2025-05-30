package ctxflow

type ErrAlreadyRunning struct{}

func (ErrAlreadyRunning) Error() string {
	return "already running"
}

type ErrAlreadyNotRunning struct{}

func (ErrAlreadyNotRunning) Error() string {
	return "already not running"
}
