package controller

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"
	"unicode"

	"dicloud/internal/auth/dao"
	"dicloud/models/model"
	"dicloud/util/response"

	"github.com/gin-gonic/gin"
	"github.com/mozillazg/go-pinyin"
)

const (
	passwordPolicyMinLength = 8
	passwordPolicyMaxLength = 15
	passwordPolicyErrorCode = "PASSWORD_POLICY_VIOLATION"
	passwordPolicyMessage   = "密码必须包含大小写字母、数字、特殊符号，长度在8-15位；密码不能连续字符（如123、abc）、密码不能为相同字符（如111、aaa）连续3位或3位以上；数字非1234等规律排列，字母非人名、公司名等易猜测组合"
)

var (
	errPasswordPolicyViolation   = errors.New(passwordPolicyMessage)
	errPasswordPolicyUnavailable = errors.New("密码校验服务暂不可用")
	passwordAsciiTokenPattern    = regexp.MustCompile(`[A-Za-z0-9]+`)
	passwordDigitSegmentPattern  = regexp.MustCompile(`\d+`)
	passwordCommonWeakTokens     = []string{
		"password", "passwd", "admin", "root", "user", "test",
		"qwerty", "qwe", "asdf", "zxcv", "welcome", "abc",
	}
	passwordGuessableInfoTokens = []string{
		"ditoo", "dicloud", "ywcloud", "yewucloud",
		"sctower", "sctel", "tower", "company", "corp", "platform",
		"zhangsan", "lisi", "wangwu", "zhaoliu", "zhangwei", "wangwei",
		"liwei", "zhaomin", "xiaoming", "xiaohong", "xiaoli", "xiaowang",
		"jack", "tom", "jerry", "mike", "david", "alice", "lucy", "lily", "mary", "john",
	}
)

func validatePasswordPolicy(password string, user model.User) error {
	passwordLength := len([]rune(password))
	if passwordLength < passwordPolicyMinLength || passwordLength > passwordPolicyMaxLength {
		return errPasswordPolicyViolation
	}
	if !hasUpperLowerDigitSpecial(password) {
		return errPasswordPolicyViolation
	}
	if hasRepeatedRune(password, 3) {
		return errPasswordPolicyViolation
	}
	if hasSequentialCharacterRun(password, 3) {
		return errPasswordPolicyViolation
	}
	if hasWeakDigitPattern(password) {
		return errPasswordPolicyViolation
	}
	if hasGuessableInfoPattern(password, user) {
		return errPasswordPolicyViolation
	}
	if hasCommonWeakPasswordToken(password) {
		return errPasswordPolicyViolation
	}
	return nil
}

func hasUpperLowerDigitSpecial(password string) bool {
	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, r := range password {
		if r > unicode.MaxASCII || unicode.IsSpace(r) {
			return false
		}
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		default:
			hasSpecial = true
		}
	}
	return hasUpper && hasLower && hasDigit && hasSpecial
}

func hasSequentialCharacterRun(password string, sequenceLength int) bool {
	if sequenceLength < 2 {
		return false
	}

	runes := []rune(strings.ToLower(password))
	runLength := 1
	previousStep := rune(0)
	for i := 1; i < len(runes); i++ {
		previous := runes[i-1]
		current := runes[i]
		sameClass := (isASCIILetter(previous) && isASCIILetter(current)) ||
			(isASCIIDigit(previous) && isASCIIDigit(current))
		step := current - previous
		if sameClass && (step == 1 || step == -1) {
			if runLength == 1 || step == previousStep {
				runLength++
			} else {
				runLength = 2
			}
			previousStep = step
			if runLength >= sequenceLength {
				return true
			}
			continue
		}
		runLength = 1
		previousStep = 0
	}
	return false
}

func isASCIILetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

func isASCIIDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

func hasRepeatedRune(password string, repeatCount int) bool {
	if repeatCount <= 1 {
		return false
	}
	runes := []rune(password)
	if len(runes) < repeatCount {
		return false
	}

	count := 1
	for i := 1; i < len(runes); i++ {
		if runes[i] == runes[i-1] {
			count++
			if count >= repeatCount {
				return true
			}
			continue
		}
		count = 1
	}
	return false
}

func hasWeakDigitPattern(password string) bool {
	segments := passwordDigitSegmentPattern.FindAllString(password, -1)
	for _, segment := range segments {
		if isWeakDigitSegment(segment) {
			return true
		}
	}
	return false
}

func isWeakDigitSegment(segment string) bool {
	if len(segment) < 3 {
		return false
	}

	digits := make([]int, 0, len(segment))
	for _, r := range segment {
		if !unicode.IsDigit(r) {
			return false
		}
		digits = append(digits, int(r-'0'))
	}

	if allSameDigits(digits) {
		return true
	}
	if isArithmeticDigitSequence(digits) {
		return true
	}
	if hasRepeatedDigitBlock(segment) {
		return true
	}
	if hasRepeatedDigitRunPattern(digits) {
		return true
	}

	return false
}

func allSameDigits(digits []int) bool {
	if len(digits) < 3 {
		return false
	}
	first := digits[0]
	for _, digit := range digits[1:] {
		if digit != first {
			return false
		}
	}
	return true
}

func isArithmeticDigitSequence(digits []int) bool {
	if len(digits) < 3 {
		return false
	}
	step := digits[1] - digits[0]
	if step == 0 {
		return false
	}
	for i := 1; i < len(digits); i++ {
		if digits[i]-digits[i-1] != step {
			return false
		}
	}
	return true
}

func hasRepeatedDigitBlock(segment string) bool {
	for blockSize := 1; blockSize <= 3; blockSize++ {
		if len(segment) < blockSize*2 || len(segment)%blockSize != 0 {
			continue
		}
		block := segment[:blockSize]
		match := true
		for i := blockSize; i < len(segment); i += blockSize {
			if segment[i:i+blockSize] != block {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func hasRepeatedDigitRunPattern(digits []int) bool {
	type digitRun struct {
		value int
		count int
	}

	if len(digits) < 3 {
		return false
	}

	runs := make([]digitRun, 0, len(digits))
	for _, digit := range digits {
		if len(runs) == 0 || runs[len(runs)-1].value != digit {
			runs = append(runs, digitRun{value: digit, count: 1})
			continue
		}
		runs[len(runs)-1].count++
	}

	if len(runs) < 3 {
		return false
	}

	baseCount := runs[0].count
	if baseCount < 2 {
		return false
	}
	for _, run := range runs[1:] {
		if run.count != baseCount {
			return false
		}
	}

	values := make([]int, 0, len(runs))
	for _, run := range runs {
		values = append(values, run.value)
	}
	return isArithmeticDigitSequence(values)
}

func hasGuessableInfoPattern(password string, user model.User) bool {
	passwordLower := strings.ToLower(password)
	for _, token := range passwordPinyinTokens(user.TrueName) {
		token = strings.TrimSpace(token)
		if len([]rune(token)) >= 2 && strings.Contains(passwordLower, strings.ToLower(token)) {
			return true
		}
	}

	for _, token := range collectPasswordTokens(user) {
		token = strings.TrimSpace(token)
		if !shouldCheckPasswordToken(token) {
			continue
		}
		tokenLower := strings.ToLower(token)
		if strings.Contains(passwordLower, tokenLower) || strings.Contains(password, token) {
			return true
		}
	}

	for _, token := range passwordGuessableInfoTokens {
		if strings.Contains(passwordLower, token) {
			return true
		}
	}

	return false
}

func hasCommonWeakPasswordToken(password string) bool {
	passwordLower := strings.ToLower(password)
	for _, token := range passwordCommonWeakTokens {
		if strings.Contains(passwordLower, token) {
			return true
		}
	}
	return false
}

func shouldCheckPasswordToken(token string) bool {
	token = strings.TrimSpace(token)
	runes := []rune(token)
	if len(runes) >= 3 {
		return true
	}
	if len(runes) < 2 {
		return false
	}
	for _, r := range runes {
		if r > unicode.MaxASCII && unicode.IsLetter(r) {
			return true
		}
	}
	return false
}

func collectPasswordTokens(user model.User) []string {
	tokens := make([]string, 0, 16)
	addTokens := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		tokens = append(tokens, value)
		if strings.Contains(value, "@") {
			tokens = append(tokens, strings.Split(value, "@")[0])
		}
		tokens = append(tokens, passwordAsciiTokenPattern.FindAllString(value, -1)...)
	}

	addTokens(user.Username)
	addTokens(user.TrueName)
	addTokens(user.Email)
	addTokens(user.Phone)
	addTokens(user.PlatName)
	addTokens(user.PlatType)
	addTokens(user.Renter)
	addTokens(user.ClientId)

	if len(user.Phone) == 11 {
		tokens = append(tokens, user.Phone[7:])
		tokens = append(tokens, user.Phone[:4])
	}

	return dedupeTokens(tokens)
}

func passwordPinyinTokens(value string) []string {
	hasHan := false
	for _, r := range value {
		if unicode.Is(unicode.Han, r) {
			hasHan = true
			break
		}
	}
	if !hasHan {
		return nil
	}

	args := pinyin.NewArgs()
	parts := pinyin.LazyPinyin(value, args)
	if len(parts) == 0 {
		return nil
	}

	fullName := strings.Join(parts, "")
	initials := strings.Builder{}
	for _, part := range parts {
		if part != "" {
			initials.WriteByte(part[0])
		}
	}
	return []string{fullName, initials.String()}
}

func dedupeTokens(tokens []string) []string {
	seen := make(map[string]struct{}, len(tokens))
	result := make([]string, 0, len(tokens))
	for _, token := range tokens {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		key := strings.ToLower(token)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, token)
	}
	return result
}

func validateSubmittedPassword(password string, user model.User) error {
	if err := validatePasswordPolicy(password, user); err != nil {
		return err
	}

	isWeak, err := dao.NewWeakPasswordDAO().Exists(password)
	if err != nil {
		return fmt.Errorf("%w: %v", errPasswordPolicyUnavailable, err)
	}
	if isWeak {
		return errPasswordPolicyViolation
	}
	return nil
}

func respondPasswordPolicyError(ctx *gin.Context, err error) {
	if errors.Is(err, errPasswordPolicyUnavailable) {
		log.Printf("[AUTH] weak-password validation unavailable: %v", err)
		response.Fail(ctx, errPasswordPolicyUnavailable.Error(), nil)
		return
	}
	response.Fail(ctx, passwordPolicyMessage, gin.H{"errorCode": passwordPolicyErrorCode})
}
