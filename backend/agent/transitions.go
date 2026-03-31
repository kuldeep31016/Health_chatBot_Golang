package agent

func Transition(current State, success bool, retryCount, maxRetries int) State {
	switch current {
	case StateProcess:
		return StateDecide
	case StateDecide:
		return StateAction
	case StateAction:
		if success {
			return StateSuccess
		}
		return StateRetry
	case StateRetry:
		if retryCount >= maxRetries {
			return StateFail
		}
		return StateAction
	default:
		if success {
			return StateSuccess
		}
		return StateFail
	}
}
