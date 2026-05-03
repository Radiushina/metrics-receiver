package models

// PollCount — имя счётчика числа опросов, который по заданию отправляет агент.
const PollCount = "PollCount"

// CounterNames — известные имена счётчиков (строка id в JSON при type=counter).
var CounterNames = []string{
	PollCount,
}
