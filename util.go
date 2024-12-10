package eventbus

import logger "github.com/kordar/gologger"

func recoverPanic() {
	if r := recover(); r != nil {
		logger.Errorf("catch the exception execution, err = %v", r)
	}
}
