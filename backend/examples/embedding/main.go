package main

import "fmt"

type Human struct {
	Name string
	Age  int
}

func (h Human) Speak() {
	fmt.Printf("Привет, я %s, мне %d лет.\n", h.Name, h.Age)
}

func (h *Human) SetAge(age int) {
	h.Age = age
}

type Action struct {
	Human
	ActionType string
}

func main() {
	act := Action{
		Human: Human{
			Name: "Алиса",
			Age:  30,
		},
		ActionType: "бег",
	}

	act.Speak()
	act.SetAge(31)
	act.Speak()

	fmt.Println("Имя:", act.Name)
}