package toml

import (
	"testing"
	"io/ioutil"
	"time"
	"fmt"
)

func TestToml_Get(t *testing.T) {
	DEBUG = true
	data, _ := ioutil.ReadFile("./test_file.toml")
	toml1, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	data, _ = ioutil.ReadFile("./test_file2.toml")
	toml2, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if toml1.Get("title").(string) != "TOML Example" {
		t.Fatal("title")
	}
	if toml1.Get("database.server").(string) != "192.168.1.1" {
		t.Fatal(" database.server ")
	}
	if toml1.Get("database.ports.0").(int) != 8001 {
		t.Fatal("database.ports.0", toml1.Get("database.ports"))
	}
	n, err := time.Parse(time.RFC3339, "1979-05-27T07:32:00-08:00")
	fmt.Println(n, err)
	if !toml1.Get("owner.dob").(time.Time).Equal(n) {
		t.Fatal(toml1.Get("owner.dob").(time.Time))
	}
	toml1.Combine(toml2)
	if toml1.Get("owner.name").(string) != "TOMMM" {
		t.Fatal(toml1.Get("owner.name"))
	}
}
