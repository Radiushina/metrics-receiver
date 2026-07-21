package main

import . "os"

func main() {
	Exit(1) // want "не используйте os.Exit в функции main"
}
