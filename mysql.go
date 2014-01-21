package main

// import (
// 	"log"
// 	"github.com/ziutek/mymysql/mysql"
// 	_ "github.com/ziutek/mymysql/thrsafe"
// )

// func main() {

// 	db := mysql.New("tcp", "", "127.0.0.1:3306", "root", "", "test")
// 	err := db.Connect()
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer db.Close()

// 	rows, res, err := db.Query("select * from ResMan")
// 	if err != nil {
// 		panic(err)
// 	}

// 	log.Printf("rows : %#v %v", len(rows), res)

// 	first := res.Map("id")
// 	second := res.Map("resId")

// 	for _, row := range rows {

// 		val1, val2 := row.Int(first), row.Int(second)

// 		log.Printf("row : %v %v", val1, val2)
// 	}

// }
