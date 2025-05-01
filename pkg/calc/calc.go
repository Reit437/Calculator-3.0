package Calc

import (
	"regexp"
	"strconv"
	"strings"
)

func Calc(expression string) (map[string]string, int) {
	var (
		id, bid, eid, sp, ost, cst int
		exp                        string
		mapid                      = make(map[string]string)
	)

	re := regexp.MustCompile(`^[0-9()/*.+\-\s]+$`) //Проверки валидности выражения
	if !re.MatchString(expression) {
		return mapid, 422
	}

	sexp := strings.ReplaceAll(expression, " ", "")

	for i := 0; i < len(sexp); i++ {

		u := string(sexp[i])
		var umi1, upl1 string
		if i < len(sexp)-1 {
			upl1 = string(sexp[i+1])
		}
		if i > 0 {
			umi1 = string(sexp[i-1])
		}

		if u == "+" || (u == "-" && umi1 >= "0" && umi1 <= "9") || u == "*" || u == "/" || u == ")" || u == "(" {
			if u == ")" || u == "(" {
				sp++
				if u == ")" {
					cst++
				} else {
					ost++
				}
			} else {
				sp += 2
			}
		}
		if u == "+" || (u == "-" && umi1 >= "0" && umi1 <= "9") || u == "*" || u == "/" {

			if (upl1 < "0" || upl1 > "9" && upl1 != "(" && upl1 != ")") || (umi1 < "0" || umi1 > "9" && upl1 != "(" && upl1 != ")") {

				if upl1 == ")" && umi1 == "(" && (u == "-" && umi1 != "(") {
					return mapid, 422

				} else if umi1 != ")" && upl1 != "(" && (u == "-" && umi1 != "(") {
					return mapid, 422
				}
			}
		} else if u == "(" || u == ")" {
			if upl1 == "(" || upl1 == ")" || umi1 == "(" || umi1 == ")" {
				return mapid, 422
			}
		}
	}
	if sp != len(expression)-len(sexp) || ost != cst {
		return mapid, 422
	}
	for strings.Index(expression, "(") != -1 { //Начало цикла для замены скобок

		bs := strings.Index(expression, "(") + 1
		es := strings.Index(expression, ")")
		exp = expression[bs:es]
		exp2 := "(" + exp + ")"
		exp = exp + " "

		for strings.Index(exp, "*") != -1 || strings.Index(exp, "/") != -1 { //цикл для замены умножения и деления в скобках

			mult := strings.Index(exp, "*")
			div := strings.Index(exp, "/")

			if (mult < div && mult != -1) || div == -1 {
				for i := mult - 2; i >= 0; i-- {
					if string(exp[i]) == string(" ") {
						bid = i + 1
						break
					}
				}
				for i := mult + 2; i <= len(exp); i++ {
					if string(exp[i]) == string(" ") {
						eid = i + 1
						break
					}
				}

			} else if (mult > div && div != -1) || mult == -1 {
				for i := div - 2; i >= 0; i-- {
					if string(exp[i]) == string(" ") {
						bid = i + 1
						break
					}
				}
				for i := div + 2; i <= len(exp); i++ {
					if string(exp[i]) == string(" ") {
						eid = i + 1
						break
					}
				}
			}

			id++ //замена подвыражений исходя из данных циклов сверху
			sid := "id" + strconv.Itoa(id)
			mapid[sid] = exp[bid:eid]
			exp = strings.Replace(exp, exp[bid:eid], sid+" ", 1)
		}

		for strings.Index(exp, "+") != -1 || strings.Index(exp, " - ") != -1 { //цикл для замены сложения и вычитания в скобках

			add := strings.Index(exp, "+")
			sub := strings.Index(exp, " - ")

			if (add < sub && add != -1) || sub == -1 {
				for i := add - 2; i >= 0; i-- {
					if string(exp[i]) == string(" ") {
						bid = i + 1
						break
					}
				}
				for i := add + 2; i <= len(exp); i++ {
					if string(exp[i]) == string(" ") {
						eid = i + 1
						break
					}
				}

			} else if (add > sub && sub != -1) || add == -1 {
				for i := sub + 1 - 2; i >= 0; i-- {
					if string(exp[i]) == string(" ") {
						bid = i + 1
						break
					}
				}
				for i := sub + 1 + 2; i < len(exp); i++ {
					if string(exp[i]) == string(" ") {
						eid = i + 1
						break
					}
				}
			}

			id++ //замена подвыражений исходя из данных циклов сверху
			sid := "id" + strconv.Itoa(id)
			mapid[sid] = exp[bid:eid]
			exp = strings.Replace(exp, exp[bid:eid], sid+" ", 1)
		}
		lk := "id" + strconv.Itoa(id) //финальная замена скобок на один id
		expression = strings.Replace(expression, exp2, lk, 1)
	}

	exp = " " + expression + " "

	for strings.Index(exp, "*") != -1 || strings.Index(exp, "/") != -1 { //цикл для замены умножения и деления в остальной части выражения

		mult := strings.Index(exp, "*")
		div := strings.Index(exp, "/")

		if (mult < div && mult != -1) || div == -1 {
			for i := mult - 2; i >= 0; i-- {
				if string(exp[i]) == string(" ") {
					bid = i + 1
					break
				}
			}
			for i := mult + 2; i <= len(exp); i++ {
				if string(exp[i]) == string(" ") {
					eid = i + 1
					break
				}
			}
		} else if (mult > div && div != -1) || mult == -1 {
			for i := div - 2; i >= 0; i-- {
				if string(exp[i]) == string(" ") {
					bid = i + 1
					break
				}
			}
			for i := div + 2; i < len(exp); i++ {
				if string(exp[i]) == string(" ") {
					eid = i + 1
					break
				}
			}
		}

		id++ //замена подвыражения исходя из данных цикла сверху
		sid := "id" + strconv.Itoa(id)
		mapid[sid] = exp[bid:eid]
		exp = strings.Replace(exp, exp[bid:eid], sid+" ", 1)
	}

	for strings.Index(exp, "+") != -1 || strings.Index(exp, " - ") != -1 { //цикл для замены сложения и вычитания в остальной части выражения
		add := strings.Index(exp, "+")
		sub := strings.Index(exp, " - ")
		if (add < sub && add != -1) || sub == -1 {
			for i := add - 2; i >= 0; i-- {
				if string(exp[i]) == string(" ") {
					bid = i + 1
					break
				}
			}
			for i := add + 2; i <= len(exp); i++ {
				if string(exp[i]) == string(" ") {
					eid = i + 1
					break
				}
			}

		} else if (add > sub && sub != -1) || add == -1 {
			for i := sub + 1 - 2; i >= 0; i-- {
				if string(exp[i]) == string(" ") {
					bid = i + 1
					break
				}
			}
			for i := sub + 1 + 2; i <= len(exp); i++ {
				if string(exp[i]) == string(" ") {
					eid = i + 1
					break
				}
			}
		}

		id++ //замена подвыражения исходя из данных цикла сверху
		sid := "id" + strconv.Itoa(id)
		mapid[sid] = exp[bid:eid]
		exp = strings.Replace(exp, exp[bid:eid], sid+" ", 1)
	}
	return mapid, 201 //возврат мапы с подвыражениями
}
