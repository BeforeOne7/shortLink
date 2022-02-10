package main

func main() {
	a := App{}
	a.initialize(getEnv())
	a.Run(":8000")
}
