package util

/*
1.支持加减乘除和指数运算
2.支持括号
3.忽略空格
4.数值支持整数、小数、正负数、科学计数法
*/

import (
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
)

// 参考: https://zhuanlan.zhihu.com/p/405806428

type Parser struct {
	text string //算式字符串
	pos  int    //记录字符串中的字符位置,0开始
	char byte   //当前字符
}

func NewParser(text string) *Parser {
	return &Parser{text, 0, text[0]}
}

func (p *Parser) Run() (float64, error) {
	v, err := p.expr()
	if err != nil {
		return v, err
	}
	if p.pos < len(p.text)-1 {
		return v, p.error()
	}

	return v, nil
}

func (p *Parser) expr() (float64, error) {
	v, err := p.md()
	if err != nil {
		return v, err
	}
	for {

		switch {
		case p.char == '+':
			p.match('+')
			tem, err := p.md()
			if err != nil {
				return v, err
			}
			v += tem

		case p.char == '-':
			p.match('-')
			tem, err := p.md()
			if err != nil {
				return v, err
			}
			v -= tem

		default:
			return v, nil
		}

	}
}

// 乘除
func (p *Parser) md() (float64, error) {
	v, err := p.expo()
	if err != nil {
		return v, err
	}
	for {

		switch {
		case p.char == '*':
			p.match('*')
			x, err := p.expo()
			if err != nil {
				return v, err
			}
			v *= x

		case p.char == '/':
			p.match('/')
			tem, err := p.expo()
			if err != nil {
				return v, err
			}
			if tem == 0 {
				fmt.Println("除数不能为0")
				os.Exit(1)
			}
			v /= tem

		default:
			return v, nil
		}

	}
}

// 指数
func (p *Parser) expo() (float64, error) {
	v, err := p.fact()
	if err != nil {
		return v, err
	}
	if p.char == '^' {
		p.match('^')
		tem, err := p.expo()
		if err != nil {
			return v, err
		}
		v = math.Pow(float64(v), float64(tem))
	}

	return v, nil
}

// 计算因子
func (p *Parser) fact() (float64, error) {
	if p.char == '(' {
		p.match('(')
		v, err := p.expr()
		if err != nil {
			return v, err
		}
		p.match(')')
		return v, nil
	} else {
		return p.num()
	}
}

/*
数字.
正负整数, 正负小数, 正负科学计数法; 正则表达式: [-+]?\d\d*(\.\d+)?([eE][-+]?\d+(\.\d+)?)?
*/
func (p *Parser) num() (float64, error) {
	v := make([]byte, 0) //字符串数值
	s := 0               //状态值,默认0
	for {
		switch s {
		case 0:
			if p.char == '+' || p.char == '-' {
				s = 1
			} else if isDigit(p.char) {
				s = 2
			} else {
				return 0.0, p.error()
			}
		case 1:
			if isDigit(p.char) {
				s = 2
			} else {
				return 0.0, p.error()
			}
		case 2:
			if isDigit(p.char) {
				s = 2
			} else if p.char == '.' {
				s = 3
			} else if p.char == 'e' || p.char == 'E' {
				s = 5
			} else { //整数
				goto endfor
			}
		case 3:
			if isDigit(p.char) {
				s = 4
			} else {
				return 0.0, p.error()
			}
		case 4:
			if isDigit(p.char) {
				s = 4
			} else if p.char == 'e' || p.char == 'E' {
				s = 5
			} else { //小数
				goto endfor
			}
		case 5:
			if isDigit(p.char) {
				s = 7
			} else if p.char == '+' || p.char == '-' {
				s = 6
			} else {
				return 0.0, p.error()
			}
		case 6:
			if isDigit(p.char) {
				s = 7
			} else {
				return 0.0, p.error()
			}
		case 7:
			if isDigit(p.char) {
				s = 7
			} else if p.char == '.' {
				s = 8
			} else { //科学计数法. e后是整数, 1e2
				goto endfor
			}
		case 8:
			if isDigit(p.char) {
				s = 9
			} else {
				return 0.0, p.error()
			}
		case 9:
			if isDigit(p.char) {
				s = 9
			} else { //科学计数法. e后是小数  9e0.5
				goto endfor
			}
		}

		v = append(v, p.char) //保存字符
		p.match(p.char)       //读取下一个字符
	}
endfor:

	f, _ := strconv.ParseFloat(string(v), 64)

	return f, nil
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func (p *Parser) match(c byte) {
	if p.char == c {
		for {
			if p.pos >= len(p.text)-1 { //结束了
				p.char = ' '
				break
			} else {
				p.pos++
				p.char = p.text[p.pos]
				if p.char != ' ' && p.char != '\t' {
					break
				}
			}

		}
	}
}

func (p *Parser) error() error {
	// panic("")
	log.Println("语法错误, 位置:", p.pos)
	return errors.New(fmt.Sprint("语法错误, 位置: ", p.pos))
}

func ArithmeticNew(x float64, formula string) (res float64, err error) {

	if "" == formula || "x" == formula {
		res = x
		return
	}

	// 处理公式
	formula = strings.ReplaceAll(formula, "x", strconv.FormatFloat(x, 'f', 3, 64))

	// 数据整理, 计算
	p := NewParser(formula)
	res, err = p.Run()
	return
}
