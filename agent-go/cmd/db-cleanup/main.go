package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "modernc.org/sqlite"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: db-cleanup <db-path>")
	}
	db, err := sql.Open("sqlite", "file:"+os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Xóa transfer mồ côi (file_id không tồn tại trong files)
	res, err := db.Exec(`DELETE FROM transfers WHERE file_id NOT IN (SELECT id FROM files)`)
	if err != nil {
		log.Fatal(err)
	}
	n, _ := res.RowsAffected()
	fmt.Println("Deleted orphaned transfers:", n)

	// Xóa transfer "completed" cũ hơn 1 ngày để gọn UI
	res, err = db.Exec(`DELETE FROM transfers WHERE phase = 'completed' AND updated_at < strftime('%s','now') - 86400`)
	if err != nil {
		log.Fatal(err)
	}
	n, _ = res.RowsAffected()
	fmt.Println("Deleted old completed transfers:", n)

	rows, err := db.Query(`SELECT phase, COUNT(*) FROM transfers GROUP BY phase`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	fmt.Println("Remaining:")
	for rows.Next() {
		var phase string
		var n int
		rows.Scan(&phase, &n)
		fmt.Printf("  %s: %d\n", phase, n)
	}
}
