package util

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

/*****************/
/*               */
/*  逻辑计算器    */
/*               */
/*****************/

func LogicalOperators(logicStr string) (value bool, msg string) {

	compar := []byte{}
	digits := []float64{}
	result := []bool{}

	_wei := []int{}
	_formula := []string{}

	for idx := range logicStr {
		switch logicStr[idx] {
		case '<', '>', '=', '|', '&':
			compar = append(compar, logicStr[idx])
			_wei = append(_wei, idx)
		}
	}

	length := len(_wei)

	if 0 == length {
		msg = "请输入正确的比较式"
		return
	}

	for idx := range _wei {
		if 0 == idx {
			_formula = append(_formula, logicStr[0:_wei[idx]])
		} else {
			_formula = append(_formula, logicStr[_wei[idx-1]+1:_wei[idx]])
		}
	}
	_formula = append(_formula, logicStr[_wei[length-1]+1:])

	for idx := range _formula {

		val, err := Calculate(_formula[idx])
		if nil != err {
			msg = err.Error()
			return
		}

		digits = append(digits, val)
	}

	for idx := range compar {
		if '<' == compar[idx] {
			result = append(result, false)
			result = append(result, digits[idx] < digits[idx+1])
			compar[idx] = '?'
		}
		if '>' == compar[idx] {
			result = append(result, false)
			result = append(result, digits[idx] > digits[idx+1])
			compar[idx] = '?'
		}
		if '=' == compar[idx] {
			result = append(result, false)
			result = append(result, digits[idx] == digits[idx+1])
			compar[idx] = '?'
		}
	}
	for idx := range compar {
		if '?' == compar[idx] {
			if 0 == idx {
				compar[idx] = '|'
			} else {
				compar[idx] = compar[idx-1]
				if '&' == compar[idx-1] {
					result[idx] = true
				}
			}
		}
	}

	value = result[0]

	for idx := range compar {
		if '|' == compar[idx] {
			value = value || result[idx+1]
		}
		if '&' == compar[idx] {
			value = value && result[idx+1]
		}
	}

	return
}

/*****************/
/*               */
/*  科学计算器    */
/*               */
/*****************/

func Calculate(formulaStr string) (value float64, err error) {

	digits := []float64{}
	digit := []byte{}
	operator := []byte{}

	_digitFlag := false
	_length := len(formulaStr)
	_pointer := 0
	_ABC := -1

	for {

		if _length == _pointer {
			break
		}

		switch formulaStr[_pointer] {
		case '+', '-', '*', '^', '/':

			if 0 == len(digit) && '-' == formulaStr[_pointer] {
				digit = append(digit, formulaStr[_pointer])
			} else {
				if _digitFlag {
					_digitFlag = false
					digit = []byte{}
					goto NEXT
				}
				if 0 == len(digit) {
					err = errors.New("重复运算符")
					return
				}
				if _digit, err2 := strconv.ParseFloat(string(digit), 64); nil != err2 {
					err = errors.New("系统错误")
					return
				} else {
					digits = append(digits, _digit)
					digit = []byte{}
				}

			NEXT:
				operator = append(operator, formulaStr[_pointer])
			}

		case ')':
		case '(':

			_pointer++

			_formulaStr := ""

			___str := _pointer - 1
			for ; _pointer < _length; _pointer++ {
				if ')' == formulaStr[_pointer] {
					break
				}
				if _length-1 == _pointer || '(' == formulaStr[_pointer] {
					err = errors.New("符号'()'格式错误")
					return
				}
			}

			_formulaStr = formulaStr[___str+1 : _pointer]

			val, err2 := Calculate(_formulaStr)
			valStr := strconv.FormatFloat(val, 'f', 5, 64)
			if nil != err2 {
				err = err2
				return
			}

			// 重定义
			digit = []byte(valStr)
			formulaStr = formulaStr[:___str] + valStr + formulaStr[_pointer+1:]
			_length = len(formulaStr)
			_pointer = ___str + len(valStr) - 1
			_digitFlag = true

		case '}':
		case '{':

			_pointer++

			__cmd, __config := "", []string{}

			if -1 == _ABC {
				err = errors.New("功能式'{}'格式错误")
				return
			}
			// 获取指令符
			__cmd = formulaStr[_ABC : _pointer-1]

			// 获取参数
			___cfg, __brackets := _pointer-1, 1
			for ; _pointer < _length; _pointer++ {
				if '{' == formulaStr[_pointer] {
					__brackets++
				}
				if '}' == formulaStr[_pointer] {
					__brackets--
				}
				if '}' == formulaStr[_pointer] {
					break
				}
				if _length-1 == _pointer {
					err = errors.New("符号'{}'格式错误")
					return
				}
			}
			__config = strings.Split(formulaStr[___cfg+1:_pointer], ",")

			val, err2 := _function(__cmd, __config)

			valStr := strconv.FormatFloat(val, 'f', 5, 64)

			if nil != err2 {
				err = err2
				return
			}

			// 重定义
			digits = append(digits, val)
			formulaStr = formulaStr[:_ABC] + valStr + formulaStr[_pointer+1:]

			_length = len(formulaStr)
			_pointer = _ABC + len(valStr) - 1
			_ABC = -1
			_digitFlag = true

		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '.':

			digit = append(digit, formulaStr[_pointer])

		case 'a', 'b', 'c', 'd', 'e', 'f', 'g',
			'h', 'i', 'j', 'k', 'l', 'm', 'n',
			'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z':

			if -1 == _ABC {
				_ABC = _pointer
			}

		case ' ':
		case '\n':

		default:

			err = errors.New("非法字符")
			return
		}

		_pointer++
	}

	if 0 != len(digit) {
		_digit, err2 := strconv.ParseFloat(string(digit), 64)
		if nil != err2 {
			err = errors.New("系统错误")
			return
		}

		digits = append(digits, _digit)
	}

	if 1 != len(digits)-len(operator) {
		err = errors.New("格式错误")
		return
	}

	// 消三级运算符

	for idx := range operator {
		if '^' == operator[idx] {
			digits[idx+1] = math.Pow(digits[idx], digits[idx+1])
			digits[idx] = 1
			operator[idx] = '?'
		}
	}

	for idx := range operator {
		if '?' == operator[idx] {
			operator[idx] = '*'
		}
	}

	// 消二级运算符
	for idx := range operator {
		if '*' == operator[idx] {
			digits[idx+1] = digits[idx] * digits[idx+1]
			digits[idx] = 0
			operator[idx] = '?'
		}
		if '/' == operator[idx] {
			digits[idx+1] = digits[idx] / digits[idx+1]
			digits[idx] = 0
			operator[idx] = '?'
		}
	}
	for idx := range operator {
		if '?' == operator[idx] {
			if 0 == idx {
				operator[idx] = '+'
			} else {
				operator[idx] = operator[idx-1]
			}
		}
	}

	value = digits[0]

	for idx := range operator {
		if '+' == operator[idx] {
			value += digits[idx+1]
		}
		if '-' == operator[idx] {
			value -= digits[idx+1]
		}
	}

	return
}

func _function(cmd string, config []string) (value float64, err error) {

	switch cmd {
	case "sin": // 正弦

		if 1 != len(config) {
			err = errors.New("参数个数错误")
		}

		value, err = Calculate(config[0])
		if nil != err {
			return
		}

		value = math.Sin(value)

	case "asin": // 反正弦

		if 1 != len(config) {
			err = errors.New("参数个数错误")
		}

		value, err = Calculate(config[0])
		if nil != err {
			return
		}

		value = math.Asin(value)

	case "cos": // 余弦

		if 1 != len(config) {
			err = errors.New("参数个数错误")
		}

		value, err = Calculate(config[0])
		if nil != err {
			return
		}

		value = math.Cos(value)

	case "acos": // 反余弦

		if 1 != len(config) {
			err = errors.New("参数个数错误")
		}

		value, err = Calculate(config[0])
		if nil != err {
			return
		}

		value = math.Acos(value)

	case "tan": // 正切函数

		if 1 != len(config) {
			err = errors.New("参数个数错误")
		}

		value, err = Calculate(config[0])
		if nil != err {
			return
		}

		value = math.Tan(value)

	case "atan": // 反正切函数

		if 1 != len(config) {
			err = errors.New("参数个数错误")
		}

		value, err = Calculate(config[0])
		if nil != err {
			return
		}

		value = math.Atan(value)

	case "cbrt": //立方根

		if 1 != len(config) {
			err = errors.New("参数个数错误")
		}

		value, err = Calculate(config[0])
		if nil != err {
			return
		}

		value = math.Cbrt(value)

	case "pow": //幂

		if 2 != len(config) {
			err = errors.New("参数个数错误")
		}

		_x, _err := Calculate(config[0])
		if nil != err {
			err = _err
			return
		}

		_y, __err := Calculate(config[1])
		if __err != nil {
			err = __err
			return
		}

		value = math.Pow(_x, _y)

	case "sqrt": //平凡根

		if 1 != len(config) {
			err = errors.New("参数个数错误")
		}

		value, err = Calculate(config[0])
		if nil != err {
			return
		}

		value = math.Sqrt(value)

	case "exp": //指数运算

		if 1 != len(config) {
			err = errors.New("参数个数错误")
		}

		value, err = Calculate(config[0])
		if nil != err {
			return
		}

		value = math.Exp(value)

	case "abs": //绝对值

		if 1 != len(config) {
			err = errors.New("参数个数错误")
		}

		value, err = Calculate(config[0])
		if nil != err {
			return
		}

		value = math.Abs(value)

	case "ceil": //向上取整

		if 1 != len(config) {
			err = errors.New("参数个数错误")
		}

		value, err = Calculate(config[0])
		if nil != err {
			return
		}

		value = math.Ceil(value)

	case "floor": //向下取整

		if 1 != len(config) {
			err = errors.New("参数个数错误")
		}

		value, err = Calculate(config[0])
		if nil != err {
			return
		}

		value = math.Floor(value)

	case "mod": //取模

		if 2 != len(config) {
			err = errors.New("参数个数错误")
		}

		_x, _err := Calculate(config[0])
		if err != nil {
			err = _err
			return
		}

		_y, __err := Calculate(config[1])
		if __err != nil {
			err = __err
			return
		}

		value = math.Mod(_x, _y)

	case "max": //取大值

		if 2 != len(config) {
			err = errors.New("参数个数错误")
		}

		_x, _err := Calculate(config[0])
		if nil != err {
			err = _err
			return
		}

		_y, __err := Calculate(config[1])
		if __err != nil {
			err = __err
			return
		}

		value = math.Max(_x, _y)

	case "min": //取小值

		if 2 != len(config) {
			err = errors.New("参数个数错误")
		}

		_x, _err := Calculate(config[0])
		if _err != nil {
			err = _err
			return
		}

		_y, __err := Calculate(config[1])
		if __err != nil {
			err = __err
			return
		}

		value = math.Min(_x, _y)
	}

	return
}

/*****************/
/*               */
/*   四则运算器   */
/*               */
/*****************/

type Stack struct {
	data [1024]string
	top  int
}

func Arithmetic(x float64, formula string) (res float64, err error) {

	if "" == formula || "x" == formula {
		res = x
		return
	}

	formula = strings.Replace(formula, "x", strconv.FormatFloat(x, 'f', 3, 64), -1)

	express := strings.TrimSpace(formula)
	postfixExpress, err := transPostfixExpress(express)
	if err != nil {
		return
	}

	//	后缀表达式求值
	res, err = calc(postfixExpress)
	if err != nil {
		return
	}

	return
}

func (s *Stack) IsEmpty() bool {
	return s.top == 0
}

func (s *Stack) Top() (ret string, err error) {
	if s.top == 0 {
		err = fmt.Errorf("公式格式错误")
		return
	}
	ret = s.data[s.top-1]
	return
}

func (s *Stack) Push(str string) {
	s.data[s.top] = str
	s.top++
}

func (s *Stack) Pop() (ret string, err error) {
	if s.top == 0 {
		err = fmt.Errorf("公式格式错误")
		return
	}
	s.top--
	ret = s.data[s.top]
	return
}

// 将中缀表达式转换成后缀表达式（逆波兰式），postfixExpress：后缀表达式
func transPostfixExpress(express string) (postfixExpress []string, err error) {
	var (
		opStack Stack //	运算符堆栈
		i       int
	)

LABEL:
	for i < len(express) { //	从左至右扫描中缀表达式
		switch {
		//	1. 若读取的是操作数，则将该操作数存入后缀表达式。
		case (express[i] >= '0' && express[i] <= '9') || express[i] == '.':
			var number []byte //	如数字123，由'1'、'2'、'3'组成
			for ; i < len(express); i++ {
				if express[i] < '0' || express[i] > '9' {
					if express[i] != '.' {
						break
					}
				}
				number = append(number, express[i])
			}
			postfixExpress = append(postfixExpress, string(number))

		//	2. 若读取的是运算符：
		//	(1) 该运算符为左括号"("，则直接压入运算符堆栈。
		case express[i] == '(':
			opStack.Push(fmt.Sprintf("%c", express[i]))
			i++

		//	(2) 该运算符为右括号")"，则输出运算符堆栈中的运算符到后缀表达式，直到遇到左括号为止。
		case express[i] == ')':
			for !opStack.IsEmpty() {
				data, _ := opStack.Pop()
				if data[0] == '(' {
					break
				}
				postfixExpress = append(postfixExpress, data)
			}
			i++

		//	(3) 该运算符为非括号运算符:
		case express[i] == '+' || express[i] == '-' || express[i] == '*' || express[i] == '/':
			//	(a)若运算符堆栈为空,则直接压入运算符堆栈。
			if opStack.IsEmpty() {
				opStack.Push(fmt.Sprintf("%c", express[i]))
				i++
				continue LABEL
			}

			data, _ := opStack.Top()
			//	(b)若运算符堆栈栈顶的运算符为括号，则直接压入运算符堆栈。(只可能为左括号这种情况)
			if data[0] == '(' {
				opStack.Push(fmt.Sprintf("%c", express[i]))
				i++
				continue LABEL
			}
			//	(c)若比运算符堆栈栈顶的运算符优先级低或相等，则输出栈顶运算符到后缀表达式,并将当前运算符压入运算符堆栈。
			if (express[i] == '+' || express[i] == '-') ||
				((express[i] == '*' || express[i] == '/') && (data[0] == '*' || data[0] == '/')) {
				postfixExpress = append(postfixExpress, data)
				opStack.Pop()
				opStack.Push(fmt.Sprintf("%c", express[i]))
				i++
				continue LABEL
			}
			//	(d)若比运算符堆栈栈顶的运算符优先级高，则直接压入运算符堆栈。
			opStack.Push(fmt.Sprintf("%c", express[i]))
			i++

		// （4）如果为空格则跳过：
		case express[i] == ' ':
			i++
			continue LABEL

		default:
			err = fmt.Errorf("公式格式错误")
			return
		}
	}

	//	3. 扫描结束，将运算符堆栈中的运算符依次弹出，存入后缀表达式。
	for !opStack.IsEmpty() {
		data, _ := opStack.Pop()
		if data[0] == '#' {
			break
		}
		postfixExpress = append(postfixExpress, data)
	}

	return
}

// 后缀表达式求值
func calc(postfixExpress []string) (result float64, err error) {
	var (
		num1 string
		num2 string
		s    Stack //	操作栈，用于存入操作数，运算符
	)

	//	从左至右扫描后缀表达式
	for i := 0; i < len(postfixExpress); i++ {
		var cur = postfixExpress[i]

		//	1. 若读取的是运算符
		if cur[0] == '+' || cur[0] == '-' || cur[0] == '*' || cur[0] == '/' {
			//	从操作栈中弹出两个数进行运算
			num1, err = s.Pop()
			if err != nil {
				return
			}
			num2, err = s.Pop()
			if err != nil {
				return
			}

			//	先弹出的数为B，后弹出的数为A
			B, _ := strconv.ParseFloat(num1, 32)
			A, _ := strconv.ParseFloat(num2, 32)
			var res float64

			switch cur[0] {
			case '+':
				res = A + B
			case '-':
				res = A - B
			case '*':
				res = A * B
			case '/':
				res = A / B
			default:
				err = fmt.Errorf("公式格式错误")
				return
			}

			//	将中间结果压栈
			s.Push(fmt.Sprintf("%f", res))
			//fmt.Println("mid value = ", res)
		} else {
			//	1. 若读取的是操作数，直接压栈
			s.Push(cur)
		}
	}

	//	计算结束，栈顶保存最后结果
	resultStr, err := s.Top()
	if err != nil {
		return
	}

	result, err = strconv.ParseFloat(resultStr, 64)
	return
}
