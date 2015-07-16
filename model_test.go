package main

// For posterity.
//
// func TestCreateGroup(t *testing.T) {
// 	t.Error("HI")
// 	b := &db.Binder{}
// 	query := `
// WITH g AS (
//   INSERT INTO "group"(gid) VALUES (DEFAULT) RETURNING *
// ), i AS (
//   INSERT INTO user_group(uid, gid)
//     SELECT ` + b.Bind(1) + `, gid FROM g
// )
// SELECT gid FROM g`
// 	var g Group
// 	err := db.DB.Get(&g, query, b.Items...)
// 	log.Println(g, err)
// }
