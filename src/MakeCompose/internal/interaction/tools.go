package interaction

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

func YesOrNo(question string) bool {
	for {
		var input string
		fmt.Print(question + " (y/n):")

		_, err := fmt.Scanln(&input)
		if err != nil {
			fmt.Printf("Invalid input: %v\n", err)
			continue
		}

		if input == "y" || input == "Y" || input == "yes" || input == "Yes" {
			return true
		} else if input == "n" || input == "N" || input == "no" || input == "No" {
			return false
		}

		fmt.Println("Please enter Y or N, try again")
	}
}

func Number(question string) int {
	for {
		var input string
		fmt.Printf(question + " (number):")

		_, err := fmt.Scanln(&input)
		if err != nil {
			fmt.Printf("Invalid input: %v\n", err)
			continue
		}

		num, err := strconv.Atoi(input)
		if err != nil {
			fmt.Println("Please enter a number")
			continue
		}

		return num
	}
}

func Text(question string) string {
	for {
		var input string
		fmt.Printf(question + ":")

		_, err := fmt.Scanln(&input)
		if err != nil {
			fmt.Printf("Invalid input: %v\n", err)
			continue
		}

		return input
	}
}

func BoolToString(b bool) string {
	if b {
		return "true"
	}

	return "false"
}

func NormalizeForCompose(input string) string {
	// 1. 先把 Windows 风格路径全部变成 Unix 风格
	input = filepath.ToSlash(input)

	// 2. 判断是否为绝对路径
	if filepath.IsAbs(input) {
		return input // 绝对路径直接使用
	}

	// 3. 相对路径：如果不是以 "./" 或 "../" 开头，则强制加 "./"
	if !strings.HasPrefix(input, "./") && !strings.HasPrefix(input, "../") {
		input = "./" + input
	}

	return input
}
