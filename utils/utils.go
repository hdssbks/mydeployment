package utils

func RandStr(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, n)
	for i := range result {
		result[i] = letters[i%len(letters)]
	}
	return string(result)
}
