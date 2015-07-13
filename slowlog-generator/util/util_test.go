package util

import "testing"

func TestRecurseArray(t *testing.T) {
	expected := "[0:teen, 1:age, 2:riot]"
	input := []interface{}{"teen", "age", "riot"}
	result := RecurseArray(input)
	if expected != result {
		t.Fail()
	}

}

func TestRecurseJsonMap(t *testing.T) {
	expected := "first:A, second:Saucerful, sub:{first:Of, second:Secrets}"
	input := map[string]interface{}{"first": "A", "second": "Saucerful", "sub": map[string]interface{}{"first": "Of", "second": "Secrets"}}
	result, _, _ := RecurseJsonMap(input)
	if expected != result {
		t.Fail()
	}
}
