package service

func DealCmd(msg string) (topic string, dataStr string) {

	nilExist := -1
	for idx := range msg {
		if msg[idx] == ' ' {
			nilExist = idx
			break
		} else if msg[idx] == '{' {
			if nilExist == -1 {
				return
			}
		}
	}

	if nilExist != -1 {
		topic = msg[:nilExist]
		dataStr = msg[nilExist+1:]
	}

	return
}

func DealCmdo(msg string) (topic string, dataStr string) {

	nilExist := -1
	for idx := range msg {
		if msg[idx] == ' ' {
			nilExist = idx
			break
		} else if msg[idx] == '<' {
			if nilExist == -1 {
				return
			}
		}
	}

	if nilExist != -1 {
		topic = msg[:nilExist]
		dataStr = msg[nilExist+1:]
	}

	return
}
